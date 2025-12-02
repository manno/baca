package cmd

import (
	"fmt"
	"os"

	"github.com/manno/background-coding-agent/internal/backend"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up the execution backend",
	Long: `Set up the execution backend (Kubernetes cluster).
Creates necessary secrets to allow execution runners to clone git repos,
create pull requests, and run coding agents.

Required environment variables:
  GITHUB_TOKEN - GitHub personal access token for cloning repos and creating PRs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		logger.Info("setting up execution backend")

		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")
		githubToken, _ := cmd.Flags().GetString("github-token")

		// Fallback to environment variable if flag not provided
		if githubToken == "" {
			githubToken = os.Getenv("GITHUB_TOKEN")
		}

		if githubToken == "" {
			logger.Error("github token is required")
			return fmt.Errorf("github token is required: use --github-token flag or GITHUB_TOKEN env var")
		}

		cfg, err := backend.GetConfig(kubeconfig)
		if err != nil {
			logger.Error("failed to get kubernetes config", "error", err)
			return err
		}

		backend, err := backend.New(cfg, namespace, logger)
		if err != nil {
			logger.Error("failed to create backend", "error", err)
			return err
		}

		ctx := cmd.Context()
		if err := backend.Setup(ctx, githubToken); err != nil {
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
	setupCmd.Flags().String("github-token", "", "GitHub token (defaults to GITHUB_TOKEN env var)")
}
