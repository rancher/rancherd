package resetadmin

import (
	"github.com/rancher/rancherd/pkg/auth"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewResetAdmin() *cobra.Command {
	return cli.Command(&ResetAdmin{}, cobra.Command{
		Short: "Bootstrap and reset admin password",
	})
}

type ResetAdmin struct {
	Password     string `usage:"Password for Rancher login" env:"PASSWORD"`
	PasswordFile string `usage:"Password for Rancher login, from file" env:"PASSWORD_FILE"`
	Kubeconfig   string `usage:"Kubeconfig file" env:"KUBECONFIG"`
}

func (p *ResetAdmin) Run(cmd *cobra.Command, args []string) error {
	return auth.ResetAdmin(cmd.Context(), &auth.Options{
		Password:     p.Password,
		PasswordFile: p.PasswordFile,
		Kubeconfig:   p.Kubeconfig,
	})
}
