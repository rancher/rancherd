package token

import (
	"context"

	"github.com/rancher/rancherd/pkg/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func GetToken(ctx context.Context, kubeconfig string) (string, error) {
	kubeconfig, err := kubectl.GetKubeconfig(kubeconfig)
	if err != nil {
		return "", err
	}

	conf, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return "", err
	}

	client, err := dynamic.NewForConfig(conf)
	if err != nil {
		return "", err
	}

	resource, err := client.Resource(schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "clusterregistrationtokens",
	}).Namespace("local").Get(ctx, "default-token", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	str, _, err := unstructured.NestedString(resource.Object, "status", "token")
	return str, err
}
