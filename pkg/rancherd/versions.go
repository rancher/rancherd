package rancherd

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"runtime"
	"strings"

	"github.com/rancher/rancherd/pkg/kubectl"
	data2 "github.com/rancher/wrangler/pkg/data"
	"github.com/rancher/wrangler/pkg/data/convert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func (r *Rancherd) getExistingVersions(ctx context.Context) (rancherVersion, k8sVersion, rancherOSVersion string) {
	kubeConfig, err := kubectl.GetKubeconfig("")
	if err != nil {
		return "", "", ""
	}

	data, err := ioutil.ReadFile(kubeConfig)
	if err != nil {
		return "", "", ""
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return "", "", ""
	}

	k8s, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", "", ""
	}

	return getRancherVersion(ctx, k8s), getK8sVersion(ctx, k8s), getRancherOSVersion()
}

func getRancherVersion(ctx context.Context, k8s kubernetes.Interface) string {
	secrets, err := k8s.CoreV1().Secrets("cattle-system").List(ctx, metav1.ListOptions{
		LabelSelector: "name=rancher,status=deployed",
	})
	if err != nil || len(secrets.Items) == 0 {
		return ""
	}

	data, err := base64.StdEncoding.DecodeString(string(secrets.Items[0].Data["release"]))
	if err != nil {
		return ""
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return ""
	}

	release := map[string]interface{}{}
	if err := json.NewDecoder(gz).Decode(&release); err != nil {
		return ""
	}

	version := convert.ToString(data2.GetValueN(release, "chart", "metadata", "version"))
	if version == "" {
		return ""
	}

	return "v" + version
}

func getK8sVersion(ctx context.Context, k8s kubernetes.Interface) string {
	nodes, err := k8s.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/control-plane=true",
	})
	if err != nil || len(nodes.Items) == 0 {
		return ""
	}
	return nodes.Items[0].Status.NodeInfo.KubeletVersion
}

func getRancherOSVersion() string {
	data, err := ioutil.ReadFile("/usr/lib/rancheros-release")
	if err != nil {
		return ""
	}

	scan := bufio.NewScanner(bytes.NewBuffer(data))
	for scan.Scan() {
		if strings.HasPrefix(scan.Text(), "IMAGE=") {
			return strings.TrimSuffix(strings.TrimPrefix(scan.Text(), "IMAGE="), "-"+runtime.GOARCH)
		}
	}
	return ""
}
