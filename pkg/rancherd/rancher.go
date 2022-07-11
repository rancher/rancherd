package rancherd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/plan"
	"github.com/rancher/rancherd/pkg/version"
	"github.com/rancher/rancherd/pkg/versions"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

const (
	// DefaultDataDir is the location of all state for rancherd
	DefaultDataDir = "/var/lib/rancher/rancherd"
	// DefaultConfigFile is the location of the rancherd config
	DefaultConfigFile = "/etc/rancher/rancherd/config.yaml"
)

type Config struct {
	Force      bool
	DataDir    string
	ConfigPath string
}

type UpgradeConfig struct {
	RancherVersion    string
	KubernetesVersion string
	RancherOSVersion  string
	Force             bool
}

type Rancherd struct {
	cfg Config
}

func New(cfg Config) *Rancherd {
	return &Rancherd{
		cfg: cfg,
	}
}

func (r *Rancherd) Info(ctx context.Context) error {
	rancherVersion, k8sVersion, rancherOSVersion := r.getExistingVersions(ctx)

	fmt.Printf("    Rancher:    %s\n", rancherVersion)
	fmt.Printf("    Kubernetes: %s\n", k8sVersion)
	if rancherOSVersion != "" {
		fmt.Printf("    RancherOS:  %s\n", rancherOSVersion)
	}
	fmt.Printf("    Rancherd:   %s\n\n", version.FriendlyVersion())
	return nil
}

func (r *Rancherd) Upgrade(ctx context.Context, upgradeConfig UpgradeConfig) error {
	cfg, err := config.Load(r.cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	rancherVersion, err := versions.RancherVersion(upgradeConfig.RancherVersion)
	if err != nil {
		return err
	}

	k8sVersion, err := versions.K8sVersion(upgradeConfig.KubernetesVersion)
	if err != nil {
		return err
	}

	rancherOSVersion, err := versions.RancherOSVersion(upgradeConfig.RancherOSVersion)
	if err != nil {
		return err
	}

	existingRancherVersion, existingK8sVersion, existingRancherOSVersion := r.getExistingVersions(ctx)
	if existingRancherVersion == rancherVersion &&
		existingK8sVersion == k8sVersion &&
		(existingRancherOSVersion == "" || existingRancherOSVersion == rancherOSVersion) {
		fmt.Printf("\nNothing to upgrade:\n\n")
		fmt.Printf("    Rancher:    %s\n", rancherVersion)
		if existingRancherOSVersion != "" {
			fmt.Printf("    RancherOS:  %s\n", rancherOSVersion)
		}
		fmt.Printf("    Kubernetes: %s\n\n", k8sVersion)
		return nil
	}

	if existingRancherVersion == rancherVersion {
		rancherVersion = ""
	}
	if existingK8sVersion == k8sVersion {
		k8sVersion = ""
	}
	if existingRancherOSVersion == "" || existingRancherOSVersion == rancherOSVersion {
		rancherOSVersion = ""
	}

	if k8sVersion != "" && existingK8sVersion != "" {
		existingRuntime := config.GetRuntime(existingK8sVersion)
		newRuntime := config.GetRuntime(k8sVersion)
		if existingRuntime != newRuntime {
			return fmt.Errorf("existing %s version %s is not compatible with %s version %s",
				existingRuntime, existingK8sVersion, newRuntime, k8sVersion)
		}
	}

	fmt.Printf("\nUpgrading to:\n\n")
	if rancherVersion != "" {
		fmt.Printf("    Rancher:    %s => %s\n", existingRancherVersion, rancherVersion)
	}
	if k8sVersion != "" {
		fmt.Printf("    Kubernetes: %s => %s\n", existingK8sVersion, k8sVersion)
	}
	if rancherOSVersion != "" {
		fmt.Printf("    RancherOS:  %s => %s\n", existingRancherOSVersion, rancherOSVersion)
	}

	if !r.cfg.Force {
		go func() {
			<-ctx.Done()
			logrus.Fatalf("Aborting")
		}()

		fmt.Printf("\nPress any key to continue, or CTRL+C to cancel\n")
		_, err := os.Stdin.Read(make([]byte, 1))
		if err != nil {
			return err
		}
	}

	nodePlan, err := plan.Upgrade(&cfg, k8sVersion, rancherVersion, rancherOSVersion, DefaultDataDir)
	if err != nil {
		return err
	}

	return plan.RunWithKubernetesVersion(ctx, k8sVersion, nodePlan, DefaultDataDir)
}

func (r *Rancherd) execute(ctx context.Context) error {
	cfg, err := config.Load(r.cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := r.setWorking(cfg); err != nil {
		return fmt.Errorf("saving working config to %s: %w", r.WorkingStamp(), err)
	}

	if cfg.Role == "" {
		logrus.Infof("No role defined, skipping bootstrap")
		return nil
	}

	k8sVersion, err := versions.K8sVersion(cfg.KubernetesVersion)
	if err != nil {
		return err
	}

	rancherVersion, err := versions.RancherVersion(cfg.RancherVersion)
	if err != nil {
		return err
	}

	logrus.Infof("Bootstrapping Rancher (%s/%s)", rancherVersion, k8sVersion)

	nodePlan, err := plan.ToPlan(ctx, &cfg, r.cfg.DataDir)
	if err != nil {
		return fmt.Errorf("generating plan: %w", err)
	}

	if err := plan.Run(ctx, &cfg, nodePlan, r.cfg.DataDir); err != nil {
		return fmt.Errorf("running plan: %w", err)
	}

	if err := r.setDone(cfg); err != nil {
		return err
	}

	logrus.Infof("Successfully Bootstrapped Rancher (%s/%s)", rancherVersion, k8sVersion)
	return nil
}

func (r *Rancherd) Run(ctx context.Context) error {
	if done, err := r.done(); err != nil {
		return fmt.Errorf("checking done stamp [%s]: %w", r.DoneStamp(), err)
	} else if done {
		logrus.Infof("System is already bootstrapped. To force the system to be bootstrapped again run with the --force flag")
		return nil
	}

	for {
		err := r.execute(ctx)
		if err == nil {
			return nil
		}
		logrus.Infof("failed to bootstrap system, will retry: %v", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(15 * time.Second):
		}
	}
}

func (r *Rancherd) writeConfig(path string, cfg config.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0600); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func (r *Rancherd) setWorking(cfg config.Config) error {
	return r.writeConfig(r.WorkingStamp(), cfg)
}

func (r *Rancherd) setDone(cfg config.Config) error {
	return r.writeConfig(r.DoneStamp(), cfg)
}

func (r *Rancherd) done() (bool, error) {
	if r.cfg.Force {
		_ = os.Remove(r.DoneStamp())
		return false, nil
	}
	_, err := os.Stat(r.DoneStamp())
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (r *Rancherd) DoneStamp() string {
	return filepath.Join(r.cfg.DataDir, "bootstrapped")
}

func (r *Rancherd) WorkingStamp() string {
	return filepath.Join(r.cfg.DataDir, "working")
}
