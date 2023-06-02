package kubernetes

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/ebauman/rancher-cluster-id-finder/pkg/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespace               = "cattle-fleet-system"
	impersonationNamespace  = "cattle-impersonation-system"
	localNamespace          = "cattle-system"
	fleetAgentSecretName    = "fleet-agent"
	fleetAgentConfigMapName = "fleet-agent"
	b64                     = base64.StdEncoding
)

type Kubeclient struct {
	clientset *kubernetes.Clientset
	dynamic   dynamic.Interface
	ctx       context.Context
}

func NewKubeClient(kubeconfigFile string) (*Kubeclient, error) {
	cs, dn, ctx, err := buildKubeClient(kubeconfigFile)
	if err != nil {
		return nil, err
	}

	return &Kubeclient{
		clientset: cs,
		dynamic:   dn,
		ctx:       ctx,
	}, nil
}

func buildKubeClient(kubeconfigFile string) (*kubernetes.Clientset, dynamic.Interface, context.Context, error) {
	var config *rest.Config
	var err error

	var ctx = context.Background()
	if kubeconfigFile != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigFile)
		if err != nil {
			panic(err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	dynamic, err := dynamic.NewForConfig(config)
	return clientset, dynamic, ctx, err
}

func (kc *Kubeclient) GetClusterIDFromAnnotations() (string, bool) {
	// first, let's check if the namespace we need is in existence (cattle-impersonation-system)
	_, err := kc.clientset.CoreV1().Namespaces().Get(kc.ctx, impersonationNamespace, metav1.GetOptions{})
	if err != nil {
		return "", false // _something_ went wrong, we don't really care what since it just means we can't go any further
		// ( as opposed to detecting errors.IsNotFound() and handling that separately )
	}

	secrets, err := kc.clientset.CoreV1().Secrets(impersonationNamespace).List(kc.ctx, metav1.ListOptions{})
	if err != nil {
		return "", false
	}

	for _, secret := range secrets.Items {
		annotations := secret.ObjectMeta.GetAnnotations()
		val, ok := annotations["field.cattle.io/projectId"]
		if ok {
			rancherClusterID := strings.Split(val, ":")[0]
			return rancherClusterID, true
		}
	}

	return "", false

}

func (kc *Kubeclient) CheckLocalCluster() (string, bool) {
	// check to see if we're in the local cluster first
	// this is indicated by the presence of a deployment/rancher in cattle-system namespace
	if _, err := kc.clientset.AppsV1().Deployments(localNamespace).Get(kc.ctx, "rancher", metav1.GetOptions{}); err == nil {
		return "local", true
	}

	return "", false
}

func (kc *Kubeclient) GetClusterIDFromSecret() (string, bool) {
	// first, let's check if the namespace we need is in existence (cattle-fleet-system)
	_, err := kc.clientset.CoreV1().Namespaces().Get(kc.ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return "", false // _something_ went wrong, we don't really care what since it just means we can't go any further
		// ( as opposed to detecting errors.IsNotFound() and handling that separately )
	}

	secret, err := kc.clientset.CoreV1().Secrets(namespace).Get(kc.ctx, fleetAgentSecretName, metav1.GetOptions{})
	if err != nil {
		return "", false
	}

	// now grab the annotation in the secret that contains the cluster id
	annotations := secret.ObjectMeta.GetAnnotations()
	val, ok := annotations["field.cattle.io/projectId"]
	if !ok {
		return "", false
	}

	rancherClusterID := strings.Split(val, ":")[0]

	if rancherClusterID == "" {
		return "", false
	}

	return rancherClusterID, true
}

func (kc *Kubeclient) GetClusterIDFromConfigMap() (string, bool) {
	// first, let's check if the namespace we need is in existence (cattle-fleet-system)
	_, err := kc.clientset.CoreV1().Namespaces().Get(kc.ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return "", false // _something_ went wrong, we don't really care what since it just means we can't go any further
		// ( as opposed to detecting errors.IsNotFound() and handling that separately )
	}

	// we have the namespace, grab the "fleet-agent" configmap
	configmap, err := kc.clientset.CoreV1().ConfigMaps(namespace).Get(kc.ctx, fleetAgentConfigMapName, metav1.GetOptions{})
	if err != nil {
		return "", false
	}

	// now grab the label in the configmap that contains the cluster id
	labels := configmap.ObjectMeta.GetLabels()
	rancherClusterID, ok := labels["management.cattle.io/cluster-name"]
	if !ok {
		return "", false
	}

	if rancherClusterID == "" {
		return "", false
	}

	return rancherClusterID, true
}

func (kc *Kubeclient) GetRancherURL() (string, error) {
	// check if this is the local cluster
	// local cluster is indicated by the presence of settings.management.cattle.io/server-url
	settingGvr := schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "settings",
	}
	serverUrlSetting, err := kc.dynamic.Resource(settingGvr).Get(kc.ctx, "server-url", metav1.GetOptions{})
	if err == nil && serverUrlSetting != nil {
		// no errors, and server-url resource exists
		if val, ok := serverUrlSetting.Object["value"]; ok {
			// pulled out the "value" field of the resource from Unstructured
			if stringVal, ok := val.(string); ok {
				if stringVal != "" {
					// value successfully converted to string, return this.
					return stringVal, nil
				}
			}
		}
	}

	// first, let's check if the namespace we need is in existence
	_, err = kc.clientset.CoreV1().Namespaces().Get(kc.ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// we have the namespace, now let's get the fleet-agent configuration secret
	secret, err := kc.clientset.CoreV1().Secrets(namespace).Get(kc.ctx, fleetAgentSecretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var kubeconfig types.Kubeconfig
	err = yaml.Unmarshal(secret.Data["kubeconfig"], &kubeconfig)
	if err != nil {
		return "", err
	}

	return kubeconfig.Clusters[0].Cluster.Server, nil
}

func (kc *Kubeclient) WriteConfigMap(value string,
	configMapName string, configMapNamespace string, configMapKey string) error {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: configMapNamespace,
		},
		Data: map[string]string{
			configMapKey: value,
		},
	}

	cm, err := kc.clientset.CoreV1().ConfigMaps(configMapNamespace).Create(kc.ctx, cm, metav1.CreateOptions{})

	return err
}
