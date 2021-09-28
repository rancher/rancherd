package runtime

import (
	"fmt"
	"os"

	"github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/rancherd/pkg/self"
	"github.com/rancher/system-agent/pkg/applyinator"
)

func ToWaitKubernetesInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "wait-kubernetes-provisioned",
		SaveOutput: true,
		Args: []string{"retry", kubectl.Command(k8sVersion), "-n", "fleet-local", "wait",
			"--for=condition=Provisioned=true", "clusters.provisioning.cattle.io", "local"},
		Env:     kubectl.Env(k8sVersion),
		Command: cmd,
	}, nil
}
