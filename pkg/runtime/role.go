package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/wrangler/pkg/data/convert"
	"sigs.k8s.io/yaml"
)

var (
	normalizeNames = map[string]string{
		"tlsSans":         "tls-san",
		"nodeName":        "node-name",
		"internalAddress": "internal-address",
		"taints":          "node-taint",
		"labels":          "node-label",
	}
)

func ToBootstrapFile(runtime config.Runtime) (*applyinator.File, error) {
	if runtime != config.RuntimeK3S {
		return nil, nil
	}
	data, err := json.Marshal(map[string]interface{}{
		"cluster-init": "true",
	})
	if err != nil {
		return nil, err
	}
	return &applyinator.File{
		Content: base64.StdEncoding.EncodeToString(data),
		Path:    GetRancherConfigLocation(runtime),
	}, nil
}

func ToFile(config *config.RuntimeConfig, runtime config.Runtime, clusterInit bool) (*applyinator.File, error) {
	data, err := ToConfig(config, clusterInit)
	if err != nil {
		return nil, err
	}
	return &applyinator.File{
		Content: base64.StdEncoding.EncodeToString(data),
		Path:    GetConfigLocation(runtime),
	}, nil
}

func ToConfig(config *config.RuntimeConfig, clusterInit bool) ([]byte, error) {
	configObjects := []interface{}{
		config.ConfigValues,
	}

	if clusterInit {
		configObjects = append(configObjects, config)
	}

	result := map[string]interface{}{}
	for _, data := range configObjects {
		data, err := convert.EncodeToMap(data)
		if err != nil {
			return nil, err
		}
		delete(data, "extraConfig")
		delete(data, "role")
		for oldKey, newKey := range normalizeNames {
			value, ok := data[oldKey]
			if !ok {
				continue
			}
			delete(data, oldKey)
			data[newKey] = value
		}
		for k, v := range data {
			newKey := strings.ReplaceAll(convert.ToYAMLKey(k), "_", "-")
			result[newKey] = v
		}
	}

	return yaml.Marshal(result)
}

func GetConfigLocation(runtime config.Runtime) string {
	return fmt.Sprintf("/etc/rancher/%s/config.yaml.d/40-rancherd.yaml", runtime)
}

func GetRancherConfigLocation(runtime config.Runtime) string {
	return fmt.Sprintf("/etc/rancher/%s/config.yaml.d/50-rancher.yaml", runtime)
}
