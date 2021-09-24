package rancherd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/plan"
	"github.com/rancher/rancherd/pkg/versions"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

const DefaultDataDir = "/var/lib/rancher/rancherd"

type Config struct {
	Force      bool
	DataDir    string
	ConfigPath string
}

type Rancherd struct {
	cfg Config
}

func New(cfg Config) *Rancherd {
	return &Rancherd{
		cfg: cfg,
	}
}

func (r *Rancherd) execute(ctx context.Context) error {
	cfg, err := config.Load(r.cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", r.cfg.ConfigPath, err)
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

	return r.setDone(cfg)
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
