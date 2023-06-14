package plan

import (
	"context"
	"fmt"

	"github.com/rancher/system-agent/pkg/applyinator"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/discovery"
	"github.com/rancher/rancherd/pkg/join"
	"github.com/rancher/rancherd/pkg/kubectl"
	"github.com/rancher/rancherd/pkg/probe"
	"github.com/rancher/rancherd/pkg/rancher"
	"github.com/rancher/rancherd/pkg/registry"
	"github.com/rancher/rancherd/pkg/resources"
	"github.com/rancher/rancherd/pkg/runtime"
	"github.com/rancher/rancherd/pkg/versions"
)

type plan applyinator.Plan

func toInitPlan(config *config.Config, dataDir string) (*applyinator.Plan, error) {
	if err := assignTokenIfUnset(config); err != nil {
		return nil, err
	}

	plan := plan{}
	if err := plan.addFiles(config, dataDir); err != nil {
		return nil, err
	}

	if err := plan.addInstructions(config, dataDir); err != nil {
		return nil, err
	}

	if err := plan.addProbes(config); err != nil {
		return nil, err
	}

	return (*applyinator.Plan)(&plan), nil
}

func toJoinPlan(cfg *config.Config, dataDir string) (*applyinator.Plan, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("server is required in config for all roles besides cluster-init")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required in config for all roles besides cluster-init")
	}

	plan := plan{}
	if err := plan.addFile(join.ToScriptFile(cfg, dataDir)); err != nil {
		return nil, err
	}
	if err := plan.addInstruction(join.ToInstruction(cfg, dataDir)); err != nil {
		return nil, err
	}
	if err := plan.addInstruction(probe.ToInstruction()); err != nil {
		return nil, err
	}
	if err := plan.addProbesForJoin(cfg); err != nil {
		return nil, err
	}

	return (*applyinator.Plan)(&plan), nil
}

func ToPlan(ctx context.Context, config *config.Config, dataDir string) (*applyinator.Plan, error) {
	newCfg := *config
	if err := discovery.DiscoverServerAndRole(ctx, &newCfg); err != nil {
		return nil, err
	}
	if newCfg.Role == "cluster-init" {
		return toInitPlan(&newCfg, dataDir)
	}
	return toJoinPlan(&newCfg, dataDir)
}

func (p *plan) addInstructions(cfg *config.Config, dataDir string) error {
	k8sVersion, err := versions.K8sVersion(cfg.KubernetesVersion)
	if err != nil {
		return err
	}

	if err := p.addInstruction(runtime.ToInstruction(cfg.RuntimeInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(probe.ToInstruction()); err != nil {
		return err
	}

	rancherVersion, err := versions.RancherVersion(cfg.RancherVersion)
	if err != nil {
		return err
	}
	if err := p.addInstruction(rancher.ToInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion, rancherVersion, dataDir)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToWaitRancherInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToWaitRancherWebhookInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToWaitClusterClientSecretInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToScaleDownFleetControllerInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToUpdateClientSecretInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToScaleUpFleetControllerInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(resources.ToInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion, dataDir)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToWaitSUCInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(rancher.ToWaitSUCPlanInstruction(cfg.RancherInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	if err := p.addInstruction(runtime.ToWaitKubernetesInstruction(cfg.RuntimeInstallerImage, cfg.SystemDefaultRegistry, k8sVersion)); err != nil {
		return err
	}

	p.addPrePostInstructions(cfg, k8sVersion)
	return nil
}

func (p *plan) addPrePostInstructions(cfg *config.Config, k8sVersion string) {
	var instructions []applyinator.Instruction

	for _, inst := range cfg.PreInstructions {
		if k8sVersion != "" {
			inst.Env = append(inst.Env, kubectl.Env(k8sVersion)...)
		}
		instructions = append(instructions, inst)
	}

	instructions = append(instructions, p.Instructions...)

	for _, inst := range cfg.PostInstructions {
		inst.Env = append(inst.Env, kubectl.Env(k8sVersion)...)
		instructions = append(instructions, inst)
	}

	p.Instructions = instructions
	return
}

func (p *plan) addInstruction(instruction *applyinator.Instruction, err error) error {
	if err != nil || instruction == nil {
		return err
	}

	p.Instructions = append(p.Instructions, *instruction)
	return nil
}

func (p *plan) addFiles(cfg *config.Config, dataDir string) error {
	k8sVersions, err := versions.K8sVersion(cfg.KubernetesVersion)
	if err != nil {
		return err
	}
	runtimeName := config.GetRuntime(k8sVersions)

	// config.yaml
	if err := p.addFile(runtime.ToFile(&cfg.RuntimeConfig, runtimeName, true)); err != nil {
		return err
	}

	// bootstrap config.yaml
	if err := p.addFile(runtime.ToBootstrapFile(runtimeName)); err != nil {
		return err
	}

	// registries.yaml
	if err := p.addFile(registry.ToFile(cfg.Registries, runtimeName)); err != nil {
		return err
	}

	// bootstrap manifests
	if err := p.addFile(resources.ToBootstrapFile(cfg, resources.GetBootstrapManifests(dataDir))); err != nil {
		return err
	}

	// rancher values.yaml
	return p.addFile(rancher.ToFile(cfg, dataDir))
}

func (p *plan) addFile(file *applyinator.File, err error) error {
	if err != nil || file == nil {
		return err
	}
	p.Files = append(p.Files, *file)
	return nil
}

func (p *plan) addProbesForJoin(cfg *config.Config) error {
	p.Probes = probe.ProbesForJoin(&cfg.RuntimeConfig)
	return nil
}

func (p *plan) addProbes(cfg *config.Config) error {
	k8sVersion, err := versions.K8sVersion(cfg.KubernetesVersion)
	if err != nil {
		return err
	}
	p.Probes = probe.AllProbes(config.GetRuntime(k8sVersion))
	return nil
}
