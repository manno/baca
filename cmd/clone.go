package cmd

import (
	"github.com/manno/background-coding-agent/internal/git"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone [repo-url]",
	Short: "Clone a git repository",
	Long:  `Clone a git repository in the execution backend.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		repoURL := args[0]

		logger.Info("cloning repository", "url", repoURL)

		output, _ := cmd.Flags().GetString("output")
		branch, _ := cmd.Flags().GetString("branch")

		cloner := git.NewCloner(logger)

		if err := cloner.Clone(repoURL, output, branch); err != nil {
			logger.Error("clone failed", "error", err)
			return err
		}

		logger.Info("clone completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)

	cloneCmd.Flags().String("output", ".", "output directory")
	cloneCmd.Flags().String("branch", "", "branch to clone")
}
