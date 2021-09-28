package info

import (
	"github.com/rancher/rancherd/pkg/rancherd"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewInfo() *cobra.Command {
	return cli.Command(&Info{}, cobra.Command{
		Short: "Print installation versions",
	})
}

type Info struct {
}

func (b *Info) Run(cmd *cobra.Command, args []string) error {
	r := rancherd.New(rancherd.Config{
		DataDir:    rancherd.DefaultDataDir,
		ConfigPath: rancherd.DefaultConfigFile,
	})
	return r.Info(cmd.Context())
}
