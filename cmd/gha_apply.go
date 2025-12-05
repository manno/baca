package cmd

import (
	"github.com/manno/baca/internal/backend/gha"
	"github.com/manno/baca/internal/change"
	"github.com/spf13/cobra"
)

var ghaApplyCmd = &cobra.Command{
	Use:   "apply [change-file]",
	Short: "Apply a Change definition using GitHub Actions",
	Long: `Read a Change definition and trigger a GitHub Actions workflow for each repository.
The 'baca-execute.yml' workflow must be present in the repository specified with --repo.`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		changeFile := args[0]

		logger.Info("applying change via GHA", "file", changeFile)

		ch, err := change.LoadFromFile(changeFile)
		if err != nil {
			logger.Error("failed to load change", "error", err)
			return err
		}

		workflowRepo, _ := cmd.Flags().GetString("repo")
		if workflowRepo == "" {
			logger.Error("--repo is required")
			return err
		}
		workflowFile, _ := cmd.Flags().GetString("workflow-file")

		logger.Info("loaded change", "repos", len(ch.Spec.Repos), "agent", ch.Spec.Agent)

		b, err := gha.New(logger)
		if err != nil {
			logger.Error("failed to create gha backend", "error", err)
			return err
		}

		ctx := cmd.Context()
		if err := b.ApplyChange(ctx, ch, workflowRepo, workflowFile); err != nil {
			logger.Error("failed to apply change", "error", err)
			return err
		}

		logger.Info("gha apply completed")
		return nil
	},
}

func init() {
	ghaCmd.AddCommand(ghaApplyCmd)

	ghaApplyCmd.Flags().String("repo", "", "The repository where the workflow is located (e.g., owner/repo)")
	_ = ghaApplyCmd.MarkFlagRequired("repo")
	ghaApplyCmd.Flags().String("workflow-file", "baca-execute.yml", "The name of the workflow file")
	// TODO: Add --wait flag
}
