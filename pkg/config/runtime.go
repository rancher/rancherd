package config

import "strings"

var (
	RuntimeRKE2    Runtime = "rke2"
	RuntimeK3S     Runtime = "k3s"
	RuntimeUnknown Runtime = "unknown"
)

type Runtime string

type RuntimeConfig struct {
	Role            string                 `json:"role,omitempty"`
	SANS            []string               `json:"tlsSans,omitempty"`
	NodeName        string                 `json:"nodeName,omitempty"`
	Address         string                 `json:"address,omitempty"`
	InternalAddress string                 `json:"internalAddress,omitempty"`
	Taints          []string               `json:"taints,omitempty"`
	Labels          []string               `json:"labels,omitempty"`
	Token           string                 `json:"token,omitempty"`
	ConfigValues    map[string]interface{} `json:"extraConfig,omitempty"`
}

func GetRuntime(kubernetesVersion string) Runtime {
	if isRKE2(kubernetesVersion) {
		return RuntimeRKE2
	}
	return RuntimeK3S
}

func isRKE2(kubernetesVersion string) bool {
	return strings.Contains(kubernetesVersion, string(RuntimeRKE2))
}
