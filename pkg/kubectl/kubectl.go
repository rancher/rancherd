package kubectl

import (
	"fmt"
	"os"

	"github.com/rancher/rancherd/pkg/config"
)

var (
	kubeconfigs = []string{
		"/etc/rancher/k3s/k3s.yaml",
		"/etc/rancher/rke2/rke2.yaml",
	}
)

func Env(k8sVersion string) []string {
	runtime := config.GetRuntime(k8sVersion)
	return []string{
		fmt.Sprintf("KUBECONFIG=/etc/rancher/%s/%s.yaml", runtime, runtime),
	}
}

func Command(k8sVersion string) string {
	kubectl := "/usr/local/bin/kubectl"
	runtime := config.GetRuntime(k8sVersion)
	if runtime == config.RuntimeRKE2 {
		kubectl = "/var/lib/rancher/rke2/bin"
	}
	return kubectl
}

func GetKubeconfig(kubeconfig string) (string, error) {
	if kubeconfig != "" {
		return kubeconfig, nil
	}

	for _, kubeconfig := range kubeconfigs {
		if _, err := os.Stat(kubeconfig); err == nil {
			return kubeconfig, nil
		}
	}
	return "", fmt.Errorf("failed to find kubeconfig file at %v", kubeconfigs)
}
