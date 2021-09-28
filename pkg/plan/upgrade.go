package plan

import (
	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/rancher"
	"github.com/rancher/rancherd/pkg/runtime"
	"github.com/rancher/system-agent/pkg/applyinator"
)

func Upgrade(cfg *config.Config, k8sVersion, rancherVersion, dataDir string) (*applyinator.Plan, error) {
	p := plan{}

	if rancherVersion != "" {
		if err := p.addInstruction(rancher.ToUpgradeInstruction("", cfg.SystemDefaultRegistry, k8sVersion, rancherVersion, dataDir)); err != nil {
			return nil, err
		}
		if err := p.addInstruction(rancher.ToWaitRancherInstruction("", cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
			return nil, err
		}
	}

	if k8sVersion != "" {
		if err := p.addInstruction(runtime.ToUpgradeInstruction(k8sVersion)); err != nil {
			return nil, err
		}
		if err := p.addInstruction(runtime.ToWaitKubernetesInstruction("", cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
			return nil, err
		}
	}

	return (*applyinator.Plan)(&p), nil
}
