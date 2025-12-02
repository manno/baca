package cmd

import (
	"github.com/manno/background-coding-agent/internal/agent"
	"github.com/manno/background-coding-agent/internal/change"
	"github.com/spf13/cobra"
)

var executeCmd = &cobra.Command{
	Use:   "execute [change-file]",
	Short: "Execute a Change definition in the execution backend",
	Long: `Execute a Change definition inside the execution backend.
This runs as part of a Kubernetes job and performs the actual code transformation.`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		changeFile := args[0]

		logger.Info("executing change", "file", changeFile)

		ch, err := change.LoadFromFile(changeFile)
		if err != nil {
			logger.Error("failed to load change", "error", err)
			return err
		}

		workDir, _ := cmd.Flags().GetString("work-dir")

		executor := agent.NewExecutor(workDir, logger)

		ctx := cmd.Context()
		if err := executor.Execute(ctx, ch); err != nil {
			logger.Error("execution failed", "error", err)
			return err
		}

		logger.Info("execution completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(executeCmd)

	executeCmd.Flags().String("work-dir", ".", "working directory")
}
