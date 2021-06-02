package retry

import (
	"time"

	"github.com/rancher/rancherd/pkg/retry"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewRetry() *cobra.Command {
	return cli.Command(&Retry{}, cobra.Command{
		Short:              "Retry command until it succeeds",
		DisableFlagParsing: true,
	})
}

type Retry struct {
}

func (p *Retry) Run(cmd *cobra.Command, args []string) error {
	return retry.Retry(cmd.Context(), 5*time.Second, args)
}
