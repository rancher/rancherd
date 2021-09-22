module github.com/rancher/rancherd

go 1.16

replace (
	k8s.io/api => k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0
	k8s.io/apiserver => k8s.io/apiserver v0.21.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.0
	k8s.io/client-go => github.com/rancher/client-go v0.21.0-rancher.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.0
	k8s.io/code-generator => k8s.io/code-generator v0.21.0
	k8s.io/component-base => k8s.io/component-base v0.21.0
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.0
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.0
	k8s.io/cri-api => k8s.io/cri-api v0.21.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.0
	k8s.io/kubectl => k8s.io/kubectl v0.21.0
	k8s.io/kubelet => k8s.io/kubelet v0.21.0
	k8s.io/kubernetes => k8s.io/kubernetes v1.21.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.0
	k8s.io/metrics => k8s.io/metrics v0.21.0
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.0
)

require (
	github.com/hashicorp/go-discover v0.0.0-20201029210230-738cb3105cd0
	github.com/pkg/errors v0.9.1
	github.com/rancher/fleet/pkg/apis v0.0.0-20210920165712-667022c1d661
	github.com/rancher/rancher/pkg/apis v0.0.0-20210920193801-79027c456224
	github.com/rancher/system-agent v0.0.1-alpha30
	github.com/rancher/wharfie v0.3.2
	github.com/rancher/wrangler v0.8.6-0.20210819203859-0babd42fbad8
	github.com/rancher/wrangler-cli v0.0.0-20210217230406-95cfa275f52f
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/yaml v1.2.0
)
