package cmd

import (
	"fmt"

	"github.com/manno/baca/internal/backend/k8s"
	"github.com/manno/baca/internal/change"
	"github.com/manno/baca/internal/mcp"
	"github.com/manno/baca/internal/workflow"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Run interactive workflow to refine and apply changes",
	Long: `The workflow command provides an interactive session to help refine your task
before executing it. The workflow agent will ask clarifying questions, gather
additional context, and produce a refined prompt that will be passed to the
coding agent for execution.

Example:
  baca workflow --change change.yaml
  baca workflow --change change.yaml --skip-interactive
  baca workflow --change change.yaml --wait`,
	RunE: runWorkflow,
}

var (
	workflowChangeFile      string
	workflowSkipInteractive bool
	workflowWait            bool
	workflowMCPSources      string
)

func init() {
	rootCmd.AddCommand(workflowCmd)

	workflowCmd.Flags().StringVarP(&workflowChangeFile, "change", "c", "", "Path to Change YAML file (required)")
	workflowCmd.Flags().BoolVar(&workflowSkipInteractive, "skip-interactive", false, "Skip interactive refinement and apply directly")
	workflowCmd.Flags().BoolVar(&workflowWait, "wait", false, "Wait for jobs to complete")
	workflowCmd.Flags().StringVar(&workflowMCPSources, "mcp", "", "MCP sources to gather context from (comma-separated: github,slack)")
	workflowCmd.Flags().String("kubeconfig", "", "path to kubeconfig file")
	workflowCmd.Flags().String("namespace", "default", "kubernetes namespace")
	workflowCmd.MarkFlagRequired("change")
}

func runWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	logger := GetLogger()

	logger.Info("starting workflow command",
		"change_file", workflowChangeFile,
		"skip_interactive", workflowSkipInteractive)

	// Parse the Change definition
	ch, err := change.LoadFromFile(workflowChangeFile)
	if err != nil {
		logger.Error("failed to parse change file", "error", err)
		return fmt.Errorf("failed to parse change file: %w", err)
	}

	logger.Info("parsed change definition",
		"repos", len(ch.Spec.Repos),
		"agent", ch.Spec.Agent)

	// Get agent command from agent config
	agentCommand := ch.Spec.Agent
	if ch.Spec.Agent == "copilot-cli" {
		agentCommand = "copilot"
	} else if ch.Spec.Agent == "gemini-cli" {
		agentCommand = "gemini"
	}

	// Create workflow agent
	workflowAgent := workflow.NewAgent(agentCommand, logger)

	// Setup MCP if sources specified
	if workflowMCPSources != "" {
		mcpSources, err := mcp.ParseSources(workflowMCPSources)
		if err != nil {
			logger.Error("failed to parse MCP sources", "error", err)
			return fmt.Errorf("failed to parse MCP sources: %w", err)
		}

		if len(mcpSources) > 0 {
			logger.Info("setting up MCP context gathering", "sources", mcpSources)
			mcpManager := mcp.NewManager(logger)

			// Register available clients
			for _, source := range mcpSources {
				switch source {
				case mcp.SourceGitHub:
					ghClient := mcp.NewGitHubClient(logger)
					mcpManager.RegisterClient(ghClient)
				case mcp.SourceSlack:
					logger.Warn("Slack MCP client not yet implemented")
					// TODO: Implement Slack client
				}
			}

			workflowAgent = workflowAgent.WithMCP(mcpManager, mcpSources)
		}
	}

	// Run workflow (interactive or non-interactive)
	var refinedChange *change.Change
	if workflowSkipInteractive {
		refinedChange, err = workflowAgent.RunNonInteractive(ctx, ch)
	} else {
		refinedChange, err = workflowAgent.Run(ctx, ch)
	}
	if err != nil {
		logger.Error("workflow failed", "error", err)
		return fmt.Errorf("workflow failed: %w", err)
	}

	// Now apply the refined change using the existing backend
	logger.Info("applying refined change to backend")

	kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
	namespace, _ := cmd.Flags().GetString("namespace")

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

	// Apply the change (no retries for workflow - it's meant to be interactive)
	if err := b.ApplyChange(ctx, refinedChange, workflowWait, 0, ""); err != nil {
		logger.Error("failed to apply change", "error", err)
		return fmt.Errorf("failed to apply change: %w", err)
	}

	logger.Info("workflow completed successfully")
	fmt.Println("\n✓ Workflow completed successfully")
	fmt.Println("Jobs have been created in the Kubernetes cluster")

	if !workflowWait {
		fmt.Printf("Run 'kubectl get jobs -n %s' to check status\n", namespace)
	}

	return nil
}
