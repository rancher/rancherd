package versions

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/rancher/rancherd/pkg/config"
	"gopkg.in/yaml.v3"
)

var (
	cachedK8sVersion     = map[string]string{}
	cachedRancherVersion = map[string]string{}
	cachedLock           sync.Mutex
	redirectClient       = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

func getVersionOrURL(urlFormat, def, version string) (_ string, isURL bool) {
	if version == "" {
		version = def
	}

	if strings.HasPrefix(version, "v") && len(strings.Split(version, ".")) > 2 {
		return version, false
	}

	channelURL := version
	if !strings.HasPrefix(channelURL, "https://") &&
		!strings.HasPrefix(channelURL, "http://") {
		if strings.HasSuffix(channelURL, "-head") {
			return channelURL, false
		}
		channelURL = fmt.Sprintf(urlFormat, version)
	}

	return channelURL, true
}

func K8sVersion(config *config.Config) (string, error) {
	cachedLock.Lock()
	defer cachedLock.Unlock()

	cached, ok := cachedK8sVersion[config.KubernetesVersion]
	if ok {
		return cached, nil
	}

	versionOrURL, isURL := getVersionOrURL("https://update.k3s.io/v1-release/channels/%s", "testing", config.KubernetesVersion)
	if !isURL {
		return versionOrURL, nil
	}

	resp, err := redirectClient.Get(versionOrURL)
	if err != nil {
		return "", fmt.Errorf("getting channel version from (%s): %w", versionOrURL, err)
	}
	defer resp.Body.Close()

	url, err := resp.Location()
	if err != nil {
		return "", fmt.Errorf("getting channel version URL from (%s): %w", versionOrURL, err)
	}

	versionOrURL = path.Base(url.Path)
	cachedK8sVersion[config.KubernetesVersion] = versionOrURL
	return versionOrURL, nil
}

func RancherVersion(config *config.Config) (string, error) {
	cachedLock.Lock()
	defer cachedLock.Unlock()

	cached, ok := cachedRancherVersion[config.RancherVersion]
	if ok {
		return cached, nil
	}

	versionOrURL, isURL := getVersionOrURL("https://releases.rancher.com/server-charts/%s/index.yaml", "master-head", config.RancherVersion)
	if !isURL {
		return versionOrURL, nil
	}

	resp, err := http.Get(versionOrURL)
	if err != nil {
		return "", fmt.Errorf("getting rancher channel version from (%s): %w", versionOrURL, err)
	}
	defer resp.Body.Close()

	index := &chartIndex{}
	if err := yaml.NewDecoder(resp.Body).Decode(index); err != nil {
		return "", fmt.Errorf("unmarshalling rancher channel version from (%s): %w", versionOrURL, err)
	}

	versions := index.Entries["rancher"]
	if len(versions) == 0 {
		return "", fmt.Errorf("failed to find version for rancher chart at (%s)", versionOrURL)
	}

	cachedRancherVersion[config.RancherVersion] = versions[0].Version
	return versions[0].Version, nil
}

type chartIndex struct {
	Entries map[string][]struct {
		Version string `yaml:"version"`
	} `yaml:"entries"`
}