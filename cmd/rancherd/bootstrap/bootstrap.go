package bootstrap

import (
	"github.com/rancher/rancherd/pkg/rancherd"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewBootstrap() *cobra.Command {
	return cli.Command(&Bootstrap{}, cobra.Command{
		Short: "Run Rancher and Kubernetes bootstrap",
	})
}

type Bootstrap struct {
	Force bool `usage:"Run bootstrap even if already bootstrapped"`
	//DataDir string `usage:"Path to rancherd state" default:"/var/lib/rancher/rancherd"`
	Config string `usage:"Custom config path" default:"/etc/rancher/rancherd/config.yaml" short:"c"`
}

func (b *Bootstrap) Run(cmd *cobra.Command, args []string) error {
	r := rancherd.New(rancherd.Config{
		Force:      b.Force,
		DataDir:    rancherd.DefaultDataDir,
		ConfigPath: b.Config,
	})
	return r.Run(cmd.Context())
}
