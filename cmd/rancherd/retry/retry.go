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
		Hidden:             true,
	})
}

type Retry struct {
	SleepFirst bool `usage:"Sleep 5 seconds before running command"`
}

func (p *Retry) Run(cmd *cobra.Command, args []string) error {
	if p.SleepFirst {
		time.Sleep(5 * time.Second)
	}
	return retry.Retry(cmd.Context(), 15*time.Second, args)
}
