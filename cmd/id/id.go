package id

import (
	"fmt"
	"github.com/ebauman/rancher-cluster-id-finder/pkg/flags"
	"github.com/ebauman/rancher-cluster-id-finder/pkg/kubernetes"
	"github.com/spf13/cobra"
	"os"
)

var IdCmd = &cobra.Command{
	Use:   "id",
	Short: "find id for rancher cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		kc, err := kubernetes.NewKubeClient(flags.KubeconfigFile)
		if err != nil {
			return err
		}

		rancherClusterId, valid := kc.CheckLocalCluster()
		if ! valid {
			rancherClusterId, valid = kc.GetClusterIDFromConfigMap()
			if ! valid {
				rancherClusterId, valid = kc.GetClusterIDFromSecret()
				if ! valid {
					rancherClusterId, valid = kc.GetClusterIDFromAnnotations()
				}
			}
		}

		if ! valid {
			return fmt.Errorf("ERROR: Could not get Cluster ID.")
		}

		if flags.ConfigMapName != "" {
			err = kc.WriteConfigMap(rancherClusterId, flags.ConfigMapName, flags.ConfigMapNamespace, flags.ConfigMapKey)
			if err != nil {
				return err
			}
		}

		if flags.WriteFile != "" {
			err = os.WriteFile(flags.WriteFile, []byte(rancherClusterId), 0666)
		}

		fmt.Print(rancherClusterId)

		return nil
	},
}
