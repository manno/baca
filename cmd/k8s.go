package cmd

import "github.com/spf13/cobra"

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Manage the Kubernetes execution backend",
	Long:  `Commands for setting up and using the Kubernetes execution backend.`,
}

func init() {
	rootCmd.AddCommand(k8sCmd)
}
