package cmd

import (
	"github.com/manno/background-coding-agent/internal/backend"
	"github.com/manno/background-coding-agent/internal/change"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [change-file]",
	Short: "Apply a Change definition",
	Long: `Read a Change definition and execute it.
Creates one job per repository defined in the Change.
Monitors job status and reports when all jobs are done.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		changeFile := args[0]

		logger.Info("applying change", "file", changeFile)

		ch, err := change.LoadFromFile(changeFile)
		if err != nil {
			logger.Error("failed to load change", "error", err)
			return err
		}

		logger.Info("loaded change", "repos", len(ch.Spec.Repos), "agent", ch.Spec.Agent)

		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")

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
		if err := backend.ApplyChange(ctx, ch); err != nil {
			logger.Error("failed to apply change", "error", err)
			return err
		}

		logger.Info("apply completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().String("kubeconfig", "", "path to kubeconfig file")
	applyCmd.Flags().String("namespace", "default", "kubernetes namespace")
}
