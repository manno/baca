package cmd

import (
	"encoding/json"

	agentpkg "github.com/manno/baca/internal/agent"
	"github.com/manno/baca/internal/change"
	"github.com/spf13/cobra"
)

var executeCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute a coding agent in the execution backend",
	Long: `Execute a coding agent inside the execution backend.
This runs as part of a Kubernetes job and performs the actual code transformation.
Accepts a JSON config string with agent, prompt, and resources.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()

		configJSON, _ := cmd.Flags().GetString("config")
		workDir, _ := cmd.Flags().GetString("work-dir")

		logger.Info("executing agent", "workDir", workDir)

		// Parse JSON config into ChangeSpec
		var spec change.ChangeSpec
		if err := json.Unmarshal([]byte(configJSON), &spec); err != nil {
			logger.Error("failed to parse config JSON", "error", err)
			return err
		}

		// Build Change from parsed config
		ch := &change.Change{
			Spec: spec,
		}

		logger.Info("parsed config", "agent", spec.Agent)

		executor := agentpkg.NewExecutor(workDir, logger)

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
	executeCmd.Flags().String("config", "", "JSON configuration with agent, prompt, agentsmd, and resources")

	_ = executeCmd.MarkFlagRequired("config")
}
