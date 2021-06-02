package runtime

import (
	"time"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/images"
	"github.com/rancher/system-agent/pkg/applyinator"
)

func ToInstruction(imageOverride string, systemDefaultRegistry string, k8sVersion string) (*applyinator.Instruction, error) {
	runtime := config.GetRuntime(k8sVersion)
	return &applyinator.Instruction{
		Name: string(runtime),
		Env: []string{
			"RESTART_STAMP=" + time.Now().String(),
		},
		Image:      images.GetInstallerImage(imageOverride, systemDefaultRegistry, k8sVersion),
		SaveOutput: true,
	}, nil
}
