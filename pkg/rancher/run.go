package rancher

import (
	"encoding/base64"
	"fmt"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/images"
	"github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/wrangler/pkg/data"
	"sigs.k8s.io/yaml"
)

var defaultValues = map[string]interface{}{
	"ingress": map[string]interface{}{
		"enabled": false,
	},
	"features":       "multi-cluster-management=false",
	"antiAffinity":   "required",
	"replicas":       -3,
	"tls":            "external",
	"hostPort":       8443,
	"noDefaultAdmin": true,
}

func GetRancherValues(dataDir string) string {
	return fmt.Sprintf("%s/rancher/values.yaml", dataDir)
}

func ToFile(cfg *config.Config, dataDir string) (*applyinator.File, error) {
	values := data.MergeMaps(defaultValues, map[string]interface{}{
		"systemDefaultRegistry": cfg.SystemDefaultRegistry,
	})
	values = data.MergeMaps(values, cfg.RancherValues)

	data, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshalling Rancher values.yaml: %w", err)
	}

	return &applyinator.File{
		Content: base64.StdEncoding.EncodeToString(data),
		Path:    GetRancherValues(dataDir),
	}, nil
}

func ToInstruction(imageOverride, systemDefaultRegistry, k8sVersion, rancherVersion, dataDir string) (*applyinator.Instruction, error) {
	return &applyinator.Instruction{
		Name:       "rancher",
		SaveOutput: true,
		Image:      images.GetRancherInstallerImage(imageOverride, systemDefaultRegistry, rancherVersion),
		Env:        append(kubectl.Env(k8sVersion), fmt.Sprintf("RANCHER_VALUES=%s", GetRancherValues(dataDir))),
	}, nil
}

func ToUpgradeInstruction(imageOverride, systemDefaultRegistry, k8sVersion, rancherVersion, dataDir string) (*applyinator.Instruction, error) {
	return &applyinator.Instruction{
		Name:       "rancher",
		SaveOutput: true,
		Image:      images.GetRancherInstallerImage(imageOverride, systemDefaultRegistry, rancherVersion),
		Env:        kubectl.Env(k8sVersion),
	}, nil
}
