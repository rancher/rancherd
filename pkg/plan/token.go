package plan

import (
	"io/ioutil"
	"os"

	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/runtime"
	"github.com/rancher/rancherd/pkg/versions"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"github.com/rancher/wrangler/pkg/yaml"
)

func assignTokenIfUnset(cfg *config.Config) error {
	if cfg.Token != "" {
		return nil
	}

	token, err := existingToken(cfg)
	if err != nil {
		return err
	}

	if token == "" {
		token, err = randomtoken.Generate()
		if err != nil {
			return err
		}
	}

	cfg.Token = token
	return nil
}

func existingToken(cfg *config.Config) (string, error) {
	k8sVersion, err := versions.K8sVersion(cfg.KubernetesVersion)
	if err != nil {
		return "", err
	}

	cfgFile := runtime.GetConfigLocation(config.GetRuntime(k8sVersion))
	data, err := ioutil.ReadFile(cfgFile)
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	configMap := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return "", err
	}

	return convert.ToString(configMap["token"]), nil
}
