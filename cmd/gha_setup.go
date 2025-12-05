package cmd

import (
	"fmt"

	"github.com/manno/baca/internal/backend/gha"
	"github.com/spf13/cobra"
)

var ghaSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create the BACA GitHub Actions workflow file",
	Long: `Creates a GitHub Actions workflow file to be used for running transformations.
The file should be committed to the repository where you want to run transformations.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()

		workflowPath, _ := cmd.Flags().GetString("workflow-path")

		if err := gha.WriteWorkflowFile(workflowPath); err != nil {
			logger.Error("failed to write workflow file", "error", err)
			return err
		}

		logger.Info("workflow file created successfully", "path", workflowPath)
		fmt.Println("Please commit this file to your repository and add the required secrets (COPILOT_TOKEN, GEMINI_API_KEY) to the repository settings.")

		return nil
	},
}

func init() {
	ghaCmd.AddCommand(ghaSetupCmd)

	ghaSetupCmd.Flags().String("workflow-path", ".github/workflows/baca-execute.yml", "Path to write the workflow file")
	// The --repo flag is not needed if we're just writing the file locally.
}
