package registry

import (
	"encoding/base64"
	"fmt"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/wharfie/pkg/registries"
	"sigs.k8s.io/yaml"
)

func ToFile(registry *registries.Registry, runtime config.Runtime) (*applyinator.File, error) {
	if registry == nil {
		return nil, nil
	}

	data, err := yaml.Marshal(registry)
	if err != nil {
		return nil, err
	}

	return &applyinator.File{
		Content:     base64.StdEncoding.EncodeToString(data),
		Path:        GetConfigFile(runtime),
		Permissions: "0400",
	}, nil

}

func GetConfigFile(runtime config.Runtime) string {
	return fmt.Sprintf("/etc/rancher/%s/registries.yaml", runtime)
}
