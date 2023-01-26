package main

import (
	"github.com/ebauman/rancher-cluster-id-finder/cmd"
)

func main() {
	cmd.Execute()
}

//package main
//
//import (
//	"bytes"
//	"compress/gzip"
//	"context"
//	"encoding/base64"
//	"flag"
//	"fmt"
//	"io/ioutil"
//	v1 "k8s.io/api/core/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/util/json"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/rest"
//	"k8s.io/client-go/tools/clientcmd"
//	"os"
//	"sort"
//)
//
//var (
//	kubeconfigFile     string
//	configMapName      string
//	configMapNamespace string
//	configMapKey       string
//	filePath           string
//	debug              bool
//
//	namespace string = "cattle-fleet-system"
//
//	b64 = base64.StdEncoding
//
//	magicGzip = []byte{0x1f, 0x8b, 0x08}
//)
//
//type Release struct {
//	Config struct {
//		Global struct {
//			Fleet struct {
//				ClusterLabels struct {
//					ClusterName string `json:"management.cattle.io/cluster-name"`
//				} `json:"clusterLabels"`
//			} `json:"fleet"`
//		} `json:"global"`
//	} `json:"config"`
//}
//
//func init() {
//	flag.StringVar(&kubeconfigFile, "kubeconfig", "", "path to kubeconfig file")
//	flag.StringVar(&configMapName, "configmap-name", "", "name of configmap to create")
//	flag.StringVar(&configMapNamespace, "configmap-namespace", "", "namespace in which to place configmap")
//	flag.StringVar(&configMapKey, "configmap-key", "", "key in configmap")
//	flag.StringVar(&filePath, "write-file", "", "path to write cluster id")
//	flag.BoolVar(&debug, "debug", false, "output debug logs")
//
//	flag.Parse()
//}
//
//func log(log string, vals ...interface{}) {
//	if debug {
//		fmt.Println(fmt.Sprintf(log, vals...))
//	}
//}
//
//func main() {
//	log("starting application")
//	var config *rest.Config
//	var err error
//
//	var ctx = context.Background()
//
//	if kubeconfigFile != "" {
//		log("using kubeconfig file: %s", kubeconfigFile)
//		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigFile)
//		if err != nil {
//			panic(err)
//		}
//	} else {
//		log("using in-cluster config")
//		config, err = rest.InClusterConfig()
//		if err != nil {
//			panic(err)
//		}
//	}
//
//	log("successfully build config")
//	clientset := kubernetes.NewForConfigOrDie(config)
//	log("successfully built kubernetes clientset")
//
//	// first, let's check if the namespace we need is in existence (cattle-fleet-system)
//	log("attempting to get namespace %s", namespace)
//	_, err = clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
//	if err != nil {
//		panic(err) // _something_ went wrong, we don't really care what since it just means we can't go any further
//		// ( as opposed to detecting errors.IsNotFound() and handling that separately )
//	}
//
//	log("namespace %s found", namespace)
//
//	// we have the namespace, now let's get the secrets in that ns, but only the helm release kind
//	log("attempting to get secrets in namespace %s", namespace)
//	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
//	if err != nil {
//		panic(err)
//	}
//
//	log("listed %d secrets in namespace %s", len(secrets.Items), namespace)
//	log("filtering out only helm secrets")
//	var helmSecrets = []v1.Secret{}
//	for _, s := range secrets.Items {
//		log("checking secrets %s", s.Name)
//		if s.Type == "helm.sh/release.v1" {
//			log("secret %s is a helm secret, adding it to candidate secrets", s.Name)
//			helmSecrets = append(helmSecrets, s)
//		} else {
//			log("secret %s is not a helm secret, discardinga", s.Name)
//		}
//	}
//
//	// from these helm secrets, sort by name to get the most recent version
//	log("sorting helm secrets by name, reverse")
//	sort.Slice(helmSecrets, func(i, j int) bool {
//		return helmSecrets[i].Name > helmSecrets[j].Name
//	})
//
//	// now get the first item from that list, and pull out the helm secret data
//	// basically stealing this process from https://github.com/helm/helm/blob/4b18b19a5e0b11450b9dc92edc75bdd7891c1f4e/pkg/storage/driver/util.go
//	var rancherClusterID = ""
//	log("secret chosen is %s", helmSecrets[0].Name)
//	if data, ok := helmSecrets[0].Data["release"]; ok {
//		// take that data and base64 decode it
//		log("decoding base64 string from secret %s", helmSecrets[0].Name)
//		b, err := b64.DecodeString(string(data))
//		if err != nil {
//			panic(err)
//		}
//
//		if len(b) > 3 && bytes.Equal(b[0:3], magicGzip) {
//			log("data is gzipped, un-gzipping")
//			r, err := gzip.NewReader(bytes.NewReader(b))
//			if err != nil {
//				panic(err)
//			}
//			defer r.Close()
//			b2, err := ioutil.ReadAll(r)
//			if err != nil {
//				panic(err)
//			}
//			b = b2
//		}
//
//		log("attempting to decode helm release struct from secret data")
//		var rls Release
//		if err := json.Unmarshal(b, &rls); err != nil {
//			panic(err)
//		}
//
//		log("decode success")
//		// now locate the rancher cluster id in that helm release data
//		log("attempting to pull config out of secret")
//		rancherClusterID = rls.Config.Global.Fleet.ClusterLabels.ClusterName
//
//	} else {
//		panic("couldn't find key release in secret")
//	}
//
//	if rancherClusterID == "" {
//		fmt.Println("rancher id not found")
//		os.Exit(1)
//	}
//
//	if configMapName != "" {
//		// write a configMap
//
//		cm := &v1.ConfigMap{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      configMapName,
//				Namespace: configMapNamespace,
//			},
//			Data: map[string]string{
//				configMapKey: rancherClusterID,
//			},
//		}
//
//		cm, err := clientset.CoreV1().ConfigMaps(configMapNamespace).Create(ctx, cm, metav1.CreateOptions{})
//		if err != nil {
//			panic(err)
//		}
//	}
//
//	if filePath != "" {
//		err = os.WriteFile(filePath, []byte(rancherClusterID), 0666)
//		if err != nil {
//			panic(err)
//		}
//	}
//
//	fmt.Print(rancherClusterID)
//	os.Exit(0)
//}
