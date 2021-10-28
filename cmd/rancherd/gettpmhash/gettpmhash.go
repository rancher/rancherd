package gettpmhash

import (
	"fmt"

	"github.com/rancher/rancherd/pkg/tpm"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewGetTPMHash() *cobra.Command {
	return cli.Command(&GetTPMHash{}, cobra.Command{
		Use:   "get-tpm-hash",
		Short: "Print TPM hash to identify this machine",
	})
}

type GetTPMHash struct {
}

func (p *GetTPMHash) Run(cmd *cobra.Command, args []string) error {
	str, err := tpm.GetPubHash()
	if err != nil {
		return err
	}
	fmt.Println(str)
	return nil
}
