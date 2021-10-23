package upgrade

import (
	"github.com/rancher/rancherd/pkg/rancherd"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewUpgrade() *cobra.Command {
	return cli.Command(&Upgrade{}, cobra.Command{
		Short: "Upgrade Rancher and Kubernetes",
	})
}

type Upgrade struct {
	RancherVersion    string `usage:"Target Rancher version" short:"r" default:"stable"`
	RancherOSVersion  string `usage:"Target RancherOS version" short:"o" default:"latest" name:"rancher-os-version"`
	KubernetesVersion string `usage:"Target Kubernetes version" short:"k" default:"stable"`
	Force             bool   `usage:"Run without prompting for confirmation" short:"f"`
}

func (b *Upgrade) Run(cmd *cobra.Command, args []string) error {
	r := rancherd.New(rancherd.Config{
		Force:      b.Force,
		DataDir:    rancherd.DefaultDataDir,
		ConfigPath: rancherd.DefaultConfigFile,
	})
	return r.Upgrade(cmd.Context(), rancherd.UpgradeConfig{
		RancherVersion:    b.RancherVersion,
		KubernetesVersion: b.KubernetesVersion,
		RancherOSVersion:  b.RancherOSVersion,
	})
}
