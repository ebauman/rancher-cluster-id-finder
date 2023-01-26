package cmd

import (
	"fmt"
	"github.com/ebauman/rancher-cluster-id-finder/cmd/id"
	"github.com/ebauman/rancher-cluster-id-finder/cmd/url"
	"github.com/ebauman/rancher-cluster-id-finder/pkg/flags"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "rcidf",
	Short: "rcidf: rancher cluster id finder. also finds rancher url!",
	Long:  "rcidf is used to find the cluster id of a downstream rancher cluster. it can also grab a rancher url",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVar(&flags.KubeconfigFile, "kubeconfig", "", "path to kubeconfig file")
	rootCmd.PersistentFlags().StringVar(&flags.ConfigMapName, "configmap-name", "", "name of configmap to create")
	rootCmd.PersistentFlags().StringVar(&flags.ConfigMapNamespace, "configmap-namespace", "", "namespace of configmap")
	rootCmd.PersistentFlags().StringVar(&flags.ConfigMapKey, "configmap-key", "", "key in configmap")
	rootCmd.PersistentFlags().StringVar(&flags.WriteFile, "write-file", "", "path to write output")

	rootCmd.AddCommand(id.IdCmd)
	rootCmd.AddCommand(url.UrlCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
