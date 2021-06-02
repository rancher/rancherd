package images

import (
	"fmt"
	"strings"

	"github.com/rancher/rancherd/pkg/config"
)

const (
	defaultSystemImagePrefix = "rancher/system-agent-installer"
)

func GetRancherInstallerImage(imageOverride, imagePrefix, rancherVersion string) string {
	return getInstallerImage(imageOverride, imagePrefix, "rancher", rancherVersion)
}

func GetInstallerImage(imageOverride, imagePrefix, kubernetesVersion string) string {
	return getInstallerImage(imageOverride, imagePrefix, string(config.GetRuntime(kubernetesVersion)), kubernetesVersion)
}

func getInstallerImage(imageOverride, imagePrefix, component, version string) string {
	if imageOverride != "" {
		return imageOverride
	}
	if imagePrefix == "" {
		imagePrefix = defaultSystemImagePrefix
	}

	tag := strings.ReplaceAll(version, "+", "-")
	if tag == "" {
		tag = "latest"
	}
	return fmt.Sprintf("%s-%s:%s", imagePrefix, component, tag)
}
