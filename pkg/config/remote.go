package config

import (
	"encoding/json"
	"fmt"

	"github.com/rancher/rancherd/pkg/cacerts"
	"github.com/rancher/wrangler/pkg/data"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/sirupsen/logrus"
)

func processRemote(cfg Config) (Config, error) {
	if cfg.Role != "" || cfg.Server == "" || cfg.Token == "" {
		return cfg, nil
	}

	logrus.Infof("server and token set but required role is not set. Trying to bootstrapping config from machine inventory")
	resp, _, err := cacerts.MachineGet(cfg.Server, cfg.Token, "/v1-rancheros/inventory")
	if err != nil {
		return cfg, fmt.Errorf("from machine inventory: %w", err)
	}

	config := map[string]interface{}{}
	if err := json.Unmarshal(resp, &config); err != nil {
		return cfg, fmt.Errorf("inventory response: %s: %w", resp, err)
	}

	currentConfig, err := convert.EncodeToMap(cfg)
	if err != nil {
		return cfg, err
	}

	var (
		newConfig = data.MergeMapsConcatSlice(currentConfig, config)
		result    Config
	)

	if err := convert.ToObj(newConfig, &result); err != nil {
		return result, err
	}

	copyConfig := result
	copyConfig.Token = "--redacted--"
	downloadedConfig, err := json.Marshal(copyConfig)
	if err == nil {
		logrus.Infof("Downloaded config: %s", downloadedConfig)
	}

	return result, nil
}
