package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone [repo-url]",
	Short: "Clone a git repository",
	Long:  `Clone a git repository in the execution backend using fleet gitcloner.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		repoURL := args[0]

		logger.Info("cloning repository", "url", repoURL)

		output, _ := cmd.Flags().GetString("output")
		branch, _ := cmd.Flags().GetString("branch")

		// Use fleet gitcloner
		fleetArgs := []string{"gitcloner", repoURL, output}
		if branch != "" {
			fleetArgs = append(fleetArgs, "--branch", branch)
		}

		fleetCmd := exec.Command("fleet", fleetArgs...)
		fleetCmd.Stdout = os.Stdout
		fleetCmd.Stderr = os.Stderr

		if err := fleetCmd.Run(); err != nil {
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
