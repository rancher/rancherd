package resources

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	v1 "github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1"
	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/images"
	kubectl "github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/rancherd/pkg/self"
	"github.com/rancher/rancherd/pkg/versions"
	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"github.com/rancher/wrangler/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func ToBootstrapFile(config *config.Config, path string) (*applyinator.File, error) {
	nodeName := config.NodeName
	if nodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("looking up hostname: %w", err)
		}
		nodeName = strings.Split(hostname, ".")[0]
	}

	k8sVersion, err := versions.K8sVersion(config)
	if err != nil {
		return nil, err
	}

	token := config.Token
	if token == "" {
		token, err = randomtoken.Generate()
		if err != nil {
			return nil, err
		}
	}

	return ToFile(append(config.BootstrapResources, v1.GenericMap{
		Data: map[string]interface{}{
			"kind":       "Node",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name": nodeName,
				"labels": map[string]interface{}{
					"node-role.kubernetes.io/etcd": "true",
				},
			},
		},
	}, v1.GenericMap{
		Data: map[string]interface{}{
			"kind":       "Namespace",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name": "fleet-local",
			},
		},
	}, v1.GenericMap{
		Data: map[string]interface{}{
			"kind":       "Cluster",
			"apiVersion": "provisioning.cattle.io/v1",
			"metadata": map[string]interface{}{
				"name":      "local",
				"namespace": "fleet-local",
			},
			"spec": map[string]interface{}{
				"kubernetesVersion": k8sVersion,
				"rkeConfig":         map[string]interface{}{},
			},
		},
	}, v1.GenericMap{
		Data: map[string]interface{}{
			"kind":       "Secret",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name":      "local-rke-state",
				"namespace": "fleet-local",
			},
			"data": map[string]interface{}{
				"serverToken": []byte(token),
				"agentToken":  []byte(token),
			},
		},
	}, v1.GenericMap{
		Data: map[string]interface{}{
			"kind":       "ClusterRegistrationToken",
			"apiVersion": "management.cattle.io/v3",
			"metadata": map[string]interface{}{
				"name":      "default-token",
				"namespace": "local",
			},
			"spec": map[string]interface{}{
				"clusterName": "local",
			},
			"status": map[string]interface{}{
				"token": token,
			},
		},
	}), path)
}
func ToFile(resources []v1.GenericMap, path string) (*applyinator.File, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	var objs []runtime.Object
	for _, resource := range resources {
		objs = append(objs, &unstructured.Unstructured{
			Object: resource.Data,
		})
	}

	data, err := yaml.ToBytes(objs)
	if err != nil {
		return nil, err
	}

	return &applyinator.File{
		Content: base64.StdEncoding.EncodeToString(data),
		Path:    path,
	}, nil
}

func GetBootstrapManifests(dataDir string) string {
	return fmt.Sprintf("%s/bootstrapmanifests/rancherd.yaml", dataDir)
}

func GetManifests(runtime config.Runtime) string {
	return fmt.Sprintf("/var/lib/rancher/%s/server/manifests/rancherd.yaml", runtime)
}

func ToInstruction(imageOverride, systemDefaultRegistry, k8sVersion, dataDir string) (*applyinator.Instruction, error) {
	bootstrap := GetBootstrapManifests(dataDir)
	kubectl := kubectl.Command(k8sVersion)

	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "bootstrap",
		SaveOutput: true,
		Image:      images.GetInstallerImage(imageOverride, systemDefaultRegistry, k8sVersion),
		Args:       []string{"retry", kubectl, "apply", "--validate=false", "-f", bootstrap},
		Command:    cmd,
	}, nil
}
