package url

import (
	"fmt"
	"github.com/ebauman/rancher-cluster-id-finder/pkg/flags"
	"github.com/ebauman/rancher-cluster-id-finder/pkg/kubernetes"
	"github.com/spf13/cobra"
	"os"
)

var UrlCmd = &cobra.Command{
	Use:   "url",
	Short: "find url for rancher",
	RunE: func(cmd *cobra.Command, args []string) error {
		kc, err := kubernetes.NewKubeClient(flags.KubeconfigFile)
		if err != nil {
			return err
		}

		url, err := kc.GetRancherURL()
		if err != nil {
			return err
		}

		if flags.ConfigMapName != "" {
			err = kc.WriteConfigMap(url, flags.ConfigMapName, flags.ConfigMapNamespace, flags.ConfigMapKey)
			if err != nil {
				return err
			}
		}

		if flags.WriteFile != "" {
			err = os.WriteFile(flags.WriteFile, []byte(url), 0666)
		}

		fmt.Print(url)

		return nil
	},
}
