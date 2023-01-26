package types

type Release struct {
	Config struct {
		Global struct {
			Fleet struct {
				ClusterLabels struct {
					ClusterName string `json:"management.cattle.io/cluster-name"`
				} `json:"clusterLabels"`
			} `json:"fleet"`
		} `json:"global"`
	} `json:"config"`
}

type Kubeconfig struct {
	APIVersion string `yaml:"apiVersion"`
	Clusters   []struct {
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificateAuthorityData"`
			Server                   string `yaml:"server"`
		} `yaml:"cluster"`
		Name string `yaml:"name"`
	} `yaml:"clusters"`
}
