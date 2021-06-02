package main

import (
	"github.com/rancher/rancherd/cmd/rancherd/bootstrap"
	"github.com/rancher/rancherd/cmd/rancherd/gettoken"
	"github.com/rancher/rancherd/cmd/rancherd/probe"
	"github.com/rancher/rancherd/cmd/rancherd/resetadmin"
	"github.com/rancher/rancherd/cmd/rancherd/retry"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

type App struct {
}

func (a *App) Run(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

func main() {
	root := cli.Command(&App{}, cobra.Command{
		Long: "Bootstrappin' the whole Ranch",
	})
	root.AddCommand(
		bootstrap.NewBootstrap(),
		gettoken.NewGetToken(),
		resetadmin.NewResetAdmin(),
		probe.NewProbe(),
		retry.NewRetry(),
	)
	cli.Main(root)
}
