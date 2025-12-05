package cmd

import "github.com/spf13/cobra"

var ghaCmd = &cobra.Command{
	Use:   "gha",
	Short: "Manage the GitHub Actions execution backend",
	Long:  `Commands for setting up and using the GitHub Actions execution backend.`,
}

func init() {
	rootCmd.AddCommand(ghaCmd)
}
