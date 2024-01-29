package rancher

import (
	"fmt"
	"os"

	"github.com/rancher/system-agent/pkg/applyinator"

	"github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/rancherd/pkg/self"
)

func ToWaitRancherInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "wait-rancher",
		SaveOutput: true,
		Args:       []string{"retry", kubectl.Command(k8sVersion), "-n", "cattle-system", "rollout", "status", "-w", "deploy/rancher"},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}

func ToWaitRancherWebhookInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "wait-rancher-webhook",
		SaveOutput: true,
		Args:       []string{"retry", kubectl.Command(k8sVersion), "-n", "cattle-system", "rollout", "status", "-w", "deploy/rancher-webhook"},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}

func ToWaitSUCInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "wait-system-upgrade-controller",
		SaveOutput: true,
		Args:       []string{"retry", kubectl.Command(k8sVersion), "-n", "cattle-system", "rollout", "status", "-w", "deploy/system-upgrade-controller"},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}

func ToWaitSUCPlanInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "wait-suc-plan-resolved",
		SaveOutput: true,
		Args: []string{"retry", kubectl.Command(k8sVersion), "-n", "cattle-system", "wait",
			"--for=condition=LatestResolved=true", "plans.upgrade.cattle.io", "system-agent-upgrader"},
		Env:     kubectl.Env(k8sVersion),
		Command: cmd,
	}, nil
}

func ToWaitClusterClientSecretInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "wait-cluster-client-secret-resolved",
		SaveOutput: true,
		Args: []string{"retry", kubectl.Command(k8sVersion), "-n", clusterNamespace, "get",
			"secret", clusterClientSecret},
		Env:     kubectl.Env(k8sVersion),
		Command: cmd,
	}, nil
}

func ToUpdateClientSecretInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "update-client-secret",
		SaveOutput: true,
		Args:       []string{"update-client-secret"},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}

func ToScaleDownFleetControllerInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "scale-down-fleet-controller",
		SaveOutput: true,
		Args:       []string{"retry", kubectl.Command(k8sVersion), "-n", "cattle-fleet-system", "scale", "--replicas", "0", "deploy/fleet-controller"},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}

func ToScaleUpFleetControllerInstruction(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "scale-up-fleet-controller",
		SaveOutput: true,
		Args:       []string{"retry", kubectl.Command(k8sVersion), "-n", "cattle-fleet-system", "scale", "--replicas", "1", "deploy/fleet-controller"},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}

// Needs to patch status subresource
// k patch cluster.provisioning local -n fleet-local --subresource=status --type=merge --patch '{"status":{"fleetWorkspaceName": "fleet-local"}}'
func PatchLocalProvisioningClusterStatus(imageOverride, systemDefaultRegistry, k8sVersion string) (*applyinator.Instruction, error) {
	cmd, err := self.Self()
	if err != nil {
		return nil, fmt.Errorf("resolving location of %s: %w", os.Args[0], err)
	}
	return &applyinator.Instruction{
		Name:       "wait-suc-plan-resolved",
		SaveOutput: true,
		Args:       []string{"retry", kubectl.Command(k8sVersion), "-n", "fleet-local", "patch", "cluster.provisioning", "local", "--subresource=status", "--type=merge", "--patch", "{\"status\":{\"fleetWorkspaceName\": \"fleet-local\"}}"},
		Env:        kubectl.Env(k8sVersion),
		Command:    cmd,
	}, nil
}
