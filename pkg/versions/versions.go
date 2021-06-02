package versions

import "github.com/rancher/rancherd/pkg/config"

func K8sVersion(config *config.Config) string {
	if config.KubernetesVersion == "" {
		return "v1.21.1+k3s1"
	}
	return config.KubernetesVersion
}

func RancherVersion(config *config.Config) string {
	if config.RancherVersion == "" {
		return "master-head"
	}
	return config.RancherVersion
}
