package gettoken

import (
	"fmt"

	"github.com/rancher/rancherd/pkg/token"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewGetToken() *cobra.Command {
	return cli.Command(&GetToken{}, cobra.Command{
		Short: "Print token to join nodes to the cluster",
	})
}

type GetToken struct {
	Kubeconfig string `usage:"Kubeconfig file" env:"KUBECONFIG"`
}

func (p *GetToken) Run(cmd *cobra.Command, args []string) error {
	str, err := token.GetToken(cmd.Context(), p.Kubeconfig)
	if err != nil {
		return err
	}
	fmt.Println(str)
	return nil
}
