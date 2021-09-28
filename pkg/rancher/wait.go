package rancher

import (
	"fmt"
	"os"

	"github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/rancherd/pkg/self"
	"github.com/rancher/system-agent/pkg/applyinator"
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
		Name:       "wait-rancher-webhook",
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
