package kubernetes

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ebauman/rancher-cluster-id-finder/pkg/types"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sort"
)

var (
	namespace            = "cattle-fleet-system"
	fleetAgentSecretName = "fleet-agent"
	b64                  = base64.StdEncoding

	magicGzip = []byte{0x1f, 0x8b, 0x08}
)

type Kubeclient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
}

func NewKubeClient(kubeconfigFile string) (*Kubeclient, error) {
	cs, ctx, err := buildKubeClient(kubeconfigFile)
	if err != nil {
		return nil, err
	}

	return &Kubeclient{
		clientset: cs,
		ctx:       ctx,
	}, nil
}

func buildKubeClient(kubeconfigFile string) (*kubernetes.Clientset, context.Context, error) {
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
	return clientset, ctx, err
}

func (kc *Kubeclient) GetClusterID() (string, error) {
	// first, let's check if the namespace we need is in existence (cattle-fleet-system)

	_, err := kc.clientset.CoreV1().Namespaces().Get(kc.ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return "", err // _something_ went wrong, we don't really care what since it just means we can't go any further
		// ( as opposed to detecting errors.IsNotFound() and handling that separately )
	}

	// we have the namespace, now let's get the secrets in that ns, but only the helm release kind
	secrets, err := kc.clientset.CoreV1().Secrets(namespace).List(kc.ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	var helmSecrets = []v1.Secret{}
	for _, s := range secrets.Items {
		if s.Type == "helm.sh/release.v1" {
			helmSecrets = append(helmSecrets, s)
		}
	}

	// from these helm secrets, sort by name to get the most recent version
	sort.Slice(helmSecrets, func(i, j int) bool {
		return helmSecrets[i].Name > helmSecrets[j].Name
	})

	// now get the first item from that list, and pull out the helm secret data
	// basically stealing this process from https://github.com/helm/helm/blob/4b18b19a5e0b11450b9dc92edc75bdd7891c1f4e/pkg/storage/driver/util.go
	var rancherClusterID = ""
	if data, ok := helmSecrets[0].Data["release"]; ok {
		// take that data and base64 decode it

		b, err := b64.DecodeString(string(data))
		if err != nil {
			return "", err
		}

		if len(b) > 3 && bytes.Equal(b[0:3], magicGzip) {

			r, err := gzip.NewReader(bytes.NewReader(b))
			if err != nil {
				return "", err
			}
			defer r.Close()
			b2, err := ioutil.ReadAll(r)
			if err != nil {
				return "", err
			}
			b = b2
		}

		var rls types.Release
		if err := json.Unmarshal(b, &rls); err != nil {
			return "", err
		}

		// now locate the rancher cluster id in that helm release data

		rancherClusterID = rls.Config.Global.Fleet.ClusterLabels.ClusterName

	} else {
		return "", fmt.Errorf("couldn't find key release in secret")
	}

	if rancherClusterID == "" {
		return "", fmt.Errorf("rancher cluster id not found")
	}

	return rancherClusterID, nil
}

func (kc *Kubeclient) GetRancherURL() (string, error) {
	// first, let's check if the namespace we need is in existence
	_, err := kc.clientset.CoreV1().Namespaces().Get(kc.ctx, namespace, metav1.GetOptions{})
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
