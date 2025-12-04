package cmd

import (
	"github.com/manno/baca/internal/backend/k8s"
	"github.com/manno/baca/internal/change"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [change-file]",
	Short: "Apply a Change definition",
	Long: `Read a Change definition and execute it.
Creates one job per repository defined in the Change.
Monitors job status and reports when all jobs are done.`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
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
		wait, _ := cmd.Flags().GetBool("wait")
		retries, _ := cmd.Flags().GetInt32("retries")
		forkOrg, _ := cmd.Flags().GetString("fork-org")

		cfg, err := k8s.GetConfig(kubeconfig)
		if err != nil {
			logger.Error("failed to get kubernetes config", "error", err)
			return err
		}

		b, err := k8s.New(cfg, namespace, logger)
		if err != nil {
			logger.Error("failed to create backend", "error", err)
			return err
		}

		ctx := cmd.Context()
		if err := b.ApplyChange(ctx, ch, wait, retries, forkOrg); err != nil {
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
	applyCmd.Flags().Bool("wait", true, "wait for jobs to complete")
	applyCmd.Flags().Int32("retries", 0, "number of times to retry failed jobs (BackoffLimit)")
	applyCmd.Flags().String("fork-org", "", "GitHub organization/user to create forks under (default: authenticated user)")
}
