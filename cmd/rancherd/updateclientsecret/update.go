package updateclientsecret

import (
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"

	"github.com/rancher/rancherd/pkg/rancher"
)

func NewUpdateClientSecret() *cobra.Command {
	return cli.Command(&UpdateClientSecret{}, cobra.Command{
		Short: "Update cluster client secret to have API Server URL and CA Certs configured",
	})
}

type UpdateClientSecret struct {
	Kubeconfig string `usage:"Kubeconfig file" env:"KUBECONFIG"`
}

func (s *UpdateClientSecret) Run(cmd *cobra.Command, args []string) error {
	return rancher.UpdateClientSecret(cmd.Context(), &rancher.Options{Kubeconfig: s.Kubeconfig})
}
