package config

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	v1 "github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1"
	"github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1/plan"
	"github.com/rancher/wharfie/pkg/registries"
	"github.com/rancher/wrangler/pkg/data"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/yaml"
	"github.com/sirupsen/logrus"
)

var (
	implicitPaths = []string{
		"/usr/share/oem/rancher/rancherd/config.yaml",
		"/usr/share/rancher/rancherd/config.yaml",
		// RancherOS userdata
		"/oem/userdata",
		// RancherOS installation yip config
		"/oem/99_custom.yaml",
		// RancherOS oem location
		"/oem/rancher/rancherd/config.yaml",
		// Standard cloud-config
		"/var/lib/cloud/instance/data/user-data.txt",
	}

	manifests = []string{
		"/usr/share/oem/rancher/rancherd/manifests",
		"/usr/share/rancher/rancherd/manifests",
		"/etc/rancher/rancherd/manifests",
		// RancherOS OEM
		"/oem/rancher/rancherd/manifests",
	}
)

type Config struct {
	RuntimeConfig
	KubernetesVersion string           `json:"kubernetesVersion,omitempty"`
	RancherVersion    string           `json:"rancherVersion,omitempty"`
	Server            string           `json:"server,omitempty"`
	Discovery         *DiscoveryConfig `json:"discovery,omitempty"`
	Role              string           `json:"role,omitempty"`

	RancherValues    map[string]interface{} `json:"rancherValues,omitempty"`
	PreInstructions  []plan.Instruction     `json:"preInstructions,omitempty"`
	PostInstructions []plan.Instruction     `json:"postInstructions,omitempty"`
	// Deprecated, use Resources instead
	BootstrapResources []v1.GenericMap `json:"bootstrapResources,omitempty"`
	Resources          []v1.GenericMap `json:"resources,omitempty"`

	RuntimeInstallerImage string                `json:"runtimeInstallerImage,omitempty"`
	RancherInstallerImage string                `json:"rancherInstallerImage,omitempty"`
	SystemDefaultRegistry string                `json:"systemDefaultRegistry,omitempty"`
	Registries            *registries.Registry  `json:"registries,omitempty"`
	Git                   *v1alpha1.GitRepoSpec `json:"git,omitempty"`
}

type DiscoveryConfig struct {
	Params          map[string]string `json:"params,omitempty"`
	ExpectedServers int               `json:"expectedServers,omitempty"`
	// ServerCacheDuration will remember discovered servers for this amount of time.  This
	// helps with some discovery protocols like mDNS that can be unreliable
	ServerCacheDuration string `json:"serverCacheDuration,omitempty"`
}

func paths() (result []string) {
	for _, file := range implicitPaths {
		result = append(result, file)

		files, err := ioutil.ReadDir(file)
		if err != nil {
			continue
		}

		for _, entry := range files {
			if isYAML(entry.Name()) {
				result = append(result, filepath.Join(file, entry.Name()))
			}
		}
	}
	return
}

func Load(path string) (result Config, err error) {
	var (
		values = map[string]interface{}{}
	)

	if err := populatedSystemResources(&result); err != nil {
		return result, err
	}

	for _, file := range paths() {
		newValues, err := mergeFile(values, file)
		if err == nil {
			values = newValues
		} else {
			logrus.Infof("failed to parse %s, skipping file: %v", file, err)
		}
	}

	if path != "" {
		values, err = mergeFile(values, path)
		if err != nil {
			return
		}
	}

	err = convert.ToObj(values, &result)
	return
}

func populatedSystemResources(config *Config) error {
	resources, err := loadResources(manifests...)
	if err != nil {
		return err
	}
	config.Resources = append(config.Resources, config.BootstrapResources...)
	config.Resources = append(config.Resources, resources...)

	return nil
}

func isYAML(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

func loadResources(dirs ...string) (result []v1.GenericMap, _ error) {
	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !isYAML(path) {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			objs, err := yaml.ToObjects(f)
			if err != nil {
				return err
			}

			for _, obj := range objs {
				apiVersion, kind := obj.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()
				if apiVersion == "" || kind == "" {
					continue
				}
				data, err := convert.EncodeToMap(obj)
				if err != nil {
					return err
				}
				result = append(result, v1.GenericMap{
					Data: data,
				})
			}

			return nil
		})
		if os.IsNotExist(err) {
			continue
		}
	}

	return
}

func mergeFile(result map[string]interface{}, file string) (map[string]interface{}, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	files, err := dotDFiles(file)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{}
	if len(bytes) > 0 {
		logrus.Infof("Loading config file [%s]", file)
		if err := yaml.Unmarshal(bytes, &values); err != nil {
			return nil, err
		}
	}

	if v, ok := values["rancherd"].(map[string]interface{}); ok {
		values = v
	}

	result = data.MergeMapsConcatSlice(result, values)
	for _, file := range files {
		result, err = mergeFile(result, file)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func dotDFiles(basefile string) (result []string, _ error) {
	files, err := ioutil.ReadDir(basefile + ".d")
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() || (!strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml")) {
			continue
		}
		result = append(result, filepath.Join(basefile+".d", file.Name()))
	}
	return
}
