package join

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/rancher/rancherd/pkg/cacerts"
	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/rancherd/pkg/roles"
	"github.com/rancher/system-agent/pkg/applyinator"
)

func addEnv(env []string, key, value string) []string {
	return append(env, fmt.Sprintf("%s=%s", key, value))
}

func GetInstallScriptFile(dataDir string) string {
	return fmt.Sprintf("%s/install.sh", dataDir)
}

func ToScriptFile(config *config.Config, dataDir string) (*applyinator.File, error) {
	data, _, err := cacerts.Get(config.Server, config.Token, "/system-agent-install.sh")
	if err != nil {
		return nil, err
	}

	return &applyinator.File{
		Content: base64.StdEncoding.EncodeToString(data),
		Path:    GetInstallScriptFile(dataDir),
	}, nil
}

func ToInstruction(config *config.Config, dataDir string) (*applyinator.Instruction, error) {
	var (
		etcd         = roles.IsEtcd(config.Role)
		controlPlane = roles.IsControlPlane(config.Role)
		worker       = roles.IsWorker(config.Role)
	)

	if !etcd && !controlPlane && !worker {
		return nil, fmt.Errorf("invalid role (%s) defined", config.Role)
	}

	_, caChecksum, err := cacerts.CACerts(config.Server, config.Token)
	if err != nil {
		return nil, err
	}

	var env []string
	env = addEnv(env, "CATTLE_SERVER", config.Server)
	env = addEnv(env, "CATTLE_TOKEN", config.Token)
	env = addEnv(env, "CATTLE_CA_CHECKSUM", caChecksum)
	env = addEnv(env, "CATTLE_ADDRESS", config.Address)
	env = addEnv(env, "CATTLE_INTERNAL_ADDRESS", config.InternalAddress)
	env = addEnv(env, "CATTLE_LABELS", strings.Join(config.Labels, ","))
	env = addEnv(env, "CATTLE_TAINTS", strings.Join(config.Taints, ","))
	env = addEnv(env, "CATTLE_ROLE_ETCD", fmt.Sprint(etcd))
	env = addEnv(env, "CATTLE_ROLE_CONTROLPLANE", fmt.Sprint(controlPlane))
	env = addEnv(env, "CATTLE_ROLE_WORKER", fmt.Sprint(worker))

	return &applyinator.Instruction{
		Name:       "join",
		SaveOutput: true,
		Env:        env,
		Args: []string{
			"sh", GetInstallScriptFile(dataDir),
		},
		Command: "/usr/bin/env",
	}, nil
}
