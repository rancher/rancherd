package os

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/rancherd/pkg/self"
	"github.com/rancher/system-agent/pkg/applyinator"
)

func ToUpgradeInstruction(k8sVersion, rancherOSVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	patch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"osImage": rancherOSVersion,
		},
	})
	if err != nil {
		return nil, err
	}
	return &applyinator.Instruction{
		Name:       "patch-rancher-os-version",
		SaveOutput: true,
		Args:       []string{"retry", kubectl.Command(k8sVersion), "--type=merge", "-n", "fleet-local", "patch", "managedosimages.rancheros.cattle.io", "default-os-image", "-p", string(patch)},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}
