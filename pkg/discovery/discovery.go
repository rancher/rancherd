package discovery

import (
	"context"

	"github.com/rancher/rancherd/pkg/config"
)

func FindServer(ctx context.Context, cfg *config.Config) (string, error) {
	return cfg.Server, nil
}
