package workflow

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/manno/baca/internal/change"
	"github.com/manno/baca/internal/mcp"
)

// Agent orchestrates the interactive workflow
type Agent struct {
	logger     *slog.Logger
	refiner    *Refiner
	mcpManager *mcp.Manager
	mcpSources []mcp.Source
}

// NewAgent creates a new workflow agent
func NewAgent(agentCommand string, logger *slog.Logger) *Agent {
	return &Agent{
		logger:     logger,
		refiner:    NewRefiner(agentCommand),
		mcpManager: nil,
		mcpSources: nil,
	}
}

// WithMCP configures the agent to use MCP context gathering
func (a *Agent) WithMCP(manager *mcp.Manager, sources []mcp.Source) *Agent {
	a.mcpManager = manager
	a.mcpSources = sources
	return a
}

// Run executes the interactive workflow session
func (a *Agent) Run(ctx context.Context, ch *change.Change) (*change.Change, error) {
	a.logger.Info("starting interactive workflow session",
		"initial_prompt", ch.Spec.Prompt,
		"repos", len(ch.Spec.Repos))

	// Create a new session
	session := NewSession(ch.Spec.Prompt)

	// Generate initial questions
	questions := a.refiner.GenerateQuestions(ch.Spec.Prompt)

	// Interactive loop
	fmt.Println("\n=== BACA Workflow Agent ===")
	fmt.Println("I'll help you refine your task before executing it.")
	fmt.Println("Answer the questions below, or type 'skip' to skip a question.")
	fmt.Println("Type 'done' when you're ready to proceed.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for i, question := range questions {
		fmt.Printf("Q%d: %s\n", i+1, question)
		session.AddMessage("agent", question)

		fmt.Print("> ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		answer = strings.TrimSpace(answer)

		if strings.ToLower(answer) == "done" {
			fmt.Println("\nProceeding with refinement...")
			break
		}

		if strings.ToLower(answer) == "skip" || answer == "" {
			session.AddMessage("user", "[skipped]")
			fmt.Println()
			continue
		}

		session.AddMessage("user", answer)
		fmt.Println()
	}

	// Ask if user wants to add any additional context
	fmt.Println("Do you have any additional context or requirements? (Enter to skip)")
	fmt.Print("> ")
	additionalContext, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	additionalContext = strings.TrimSpace(additionalContext)
	if additionalContext != "" {
		session.AddMessage("user", additionalContext)
		session.AddContext(Context{
			Source:  "manual",
			Type:    "additional_context",
			Content: additionalContext,
		})
	}

	// Gather MCP context if configured
	if a.mcpManager != nil && len(a.mcpSources) > 0 {
		fmt.Println("\nGathering context from external sources...")
		mcpItems, err := a.mcpManager.GatherContext(a.mcpSources, ch.Spec.Prompt, ch.Spec.Repos)
		if err != nil {
			a.logger.Warn("failed to gather MCP context", "error", err)
		} else if len(mcpItems) > 0 {
			fmt.Printf("Found %d relevant items from %v\n", len(mcpItems), a.mcpSources)
			for _, item := range mcpItems {
				session.AddContext(Context{
					Source:   string(item.Source),
					Type:     item.Type,
					URL:      item.URL,
					Content:  item.Content,
					Metadata: item.Metadata,
				})
			}
		} else {
			fmt.Println("No additional context found from external sources")
		}
	}

	// Refine the prompt
	a.logger.Info("refining prompt based on conversation",
		"messages", len(session.ConversationLog),
		"context_items", len(session.GatheredContext))

	refinedPrompt, err := a.refiner.RefinePrompt(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to refine prompt: %w", err)
	}

	session.SetRefinedPrompt(refinedPrompt)

	// Show refined prompt and ask for confirmation
	fmt.Println("\n=== Refined Prompt ===")
	fmt.Println(refinedPrompt)
	fmt.Println("======================")
	fmt.Println()

	fmt.Print("Proceed with this refined prompt? (y/n): ")
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read confirmation: %w", err)
	}

	confirmation = strings.ToLower(strings.TrimSpace(confirmation))
	if confirmation != "y" && confirmation != "yes" {
		return nil, fmt.Errorf("workflow cancelled by user")
	}

	// Create a new Change with the refined prompt
	refinedChange := &change.Change{
		Kind:       ch.Kind,
		APIVersion: ch.APIVersion,
		Spec: change.ChangeSpec{
			Prompt:    refinedPrompt,
			Repos:     ch.Spec.Repos,
			Agent:     ch.Spec.Agent,
			AgentsMD:  ch.Spec.AgentsMD,
			Resources: ch.Spec.Resources,
			Image:     ch.Spec.Image,
			Branch:    ch.Spec.Branch,
		},
	}

	a.logger.Info("workflow session completed",
		"session_id", session.ID,
		"original_prompt_length", len(ch.Spec.Prompt),
		"refined_prompt_length", len(refinedPrompt))

	return refinedChange, nil
}

// RunNonInteractive runs the workflow without user interaction (passthrough)
func (a *Agent) RunNonInteractive(ctx context.Context, ch *change.Change) (*change.Change, error) {
	a.logger.Info("running in non-interactive mode, skipping refinement")
	return ch, nil
}
