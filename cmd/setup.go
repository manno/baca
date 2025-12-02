package cmd

import (
	"github.com/mm/background-coding-agent/internal/backend"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up the execution backend",
	Long: `Set up the execution backend (Kubernetes cluster).
Creates necessary secrets to allow execution runners to clone git repos,
create pull requests, and run coding agents.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		logger.Info("setting up execution backend")

		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")

		backend := backend.NewKubernetesBackend(namespace, kubeconfig, logger)

		ctx := cmd.Context()
		if err := backend.Setup(ctx); err != nil {
			logger.Error("failed to setup backend", "error", err)
			return err
		}

		logger.Info("setup completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	setupCmd.Flags().String("kubeconfig", "", "path to kubeconfig file")
	setupCmd.Flags().String("namespace", "default", "kubernetes namespace")
}
