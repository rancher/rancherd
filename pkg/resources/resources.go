package resources

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	v1 "github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1"
	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/images"
	"github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/rancherd/pkg/self"
	"github.com/rancher/rancherd/pkg/versions"
	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"github.com/rancher/wrangler/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func writeCattleID(id string) error {
	if err := os.MkdirAll("/etc/rancher", 0755); err != nil {
		return fmt.Errorf("mkdir /etc/rancher: %w", err)
	}
	if err := os.MkdirAll("/etc/rancher/agent", 0700); err != nil {
		return fmt.Errorf("mkdir /etc/rancher/agent: %w", err)
	}
	return ioutil.WriteFile("/etc/rancher/agent/cattle-id", []byte(id), 0400)
}

func getCattleID() (string, error) {
	data, err := ioutil.ReadFile("/etc/rancher/agent/cattle-id")
	if os.IsNotExist(err) {
	} else if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		id, err = randomtoken.Generate()
		if err != nil {
			return "", err
		}
		return id, writeCattleID(id)
	}
	return id, nil
}

func ToBootstrapFile(config *config.Config, path string) (*applyinator.File, error) {
	nodeName := config.NodeName
	if nodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("looking up hostname: %w", err)
		}
		nodeName = strings.Split(hostname, ".")[0]
	}

	k8sVersion, err := versions.K8sVersion(config.KubernetesVersion)
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

	id, err := getCattleID()
	if err != nil {
		return nil, err
	}

	resources := config.Resources
	if config.Git != nil {
		resources = append(resources, v1.GenericMap{
			Data: map[string]interface{}{
				"kind":       "GitRepo",
				"apiVersion": "fleet.cattle.io/v1alpha1",
				"metadata": map[string]interface{}{
					"name":      "rancherd-bootstrap",
					"namespace": "fleet-local",
				},
				"spec": config.Git,
			},
		})
	}
	return ToFile(append(resources, v1.GenericMap{
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
				"labels": map[string]interface{}{
					"rke.cattle.io/init-node-machine-id": id,
				},
			},
			"spec": map[string]interface{}{
				"kubernetesVersion": k8sVersion,
				"rkeConfig": map[string]interface{}{
					"controlPlaneConfig": config.ConfigValues,
				},
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

func ToInstruction(imageOverride, systemDefaultRegistry, k8sVersion, dataDir string) (*applyinator.Instruction, error) {
	bootstrap := GetBootstrapManifests(dataDir)
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "bootstrap",
		SaveOutput: true,
		Image:      images.GetInstallerImage(imageOverride, systemDefaultRegistry, k8sVersion),
		Args:       []string{"retry", kubectl.Command(k8sVersion), "apply", "--validate=false", "-f", bootstrap},
		Command:    cmd,
		Env:        kubectl.Env(k8sVersion),
	}, nil
}
