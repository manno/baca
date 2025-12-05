package workflow

import (
	"context"
	"fmt"
	"strings"
)

// Refiner handles prompt refinement based on conversation and context
type Refiner struct {
	agentCommand string // The agent to use for refinement (e.g., "copilot", "gemini")
}

// NewRefiner creates a new prompt refiner
func NewRefiner(agentCommand string) *Refiner {
	return &Refiner{
		agentCommand: agentCommand,
	}
}

// RefinePrompt takes a session and generates a refined prompt
func (r *Refiner) RefinePrompt(ctx context.Context, session *Session) (string, error) {
	// For now, use a simple template-based approach
	// In the future, this could use an LLM to synthesize the refined prompt

	var builder strings.Builder

	// Start with initial prompt
	builder.WriteString("# Task Overview\n")
	builder.WriteString(session.InitialPrompt)
	builder.WriteString("\n\n")

	// Add conversation context
	if len(session.ConversationLog) > 0 {
		builder.WriteString("# Clarifications from Discussion\n")
		for _, msg := range session.ConversationLog {
			if msg.Role == "user" {
				builder.WriteString(fmt.Sprintf("- %s\n", msg.Content))
			}
		}
		builder.WriteString("\n")
	}

	// Add gathered context
	if len(session.GatheredContext) > 0 {
		builder.WriteString("# Additional Context\n")
		for _, ctx := range session.GatheredContext {
			builder.WriteString(fmt.Sprintf("## %s (%s)\n", ctx.Type, ctx.Source))
			if ctx.URL != "" {
				builder.WriteString(fmt.Sprintf("Source: %s\n", ctx.URL))
			}
			builder.WriteString(ctx.Content)
			builder.WriteString("\n\n")
		}
	}

	return builder.String(), nil
}

// GenerateQuestions generates clarifying questions based on the initial prompt
func (r *Refiner) GenerateQuestions(initialPrompt string) []string {
	// Simple heuristic-based questions
	// In the future, could use LLM to generate context-aware questions

	questions := []string{
		"Which specific files or directories should be modified?",
		"Are there any related issues, PRs, or documentation to reference?",
		"What is the expected behavior after the changes?",
		"Are there any constraints or requirements to consider?",
	}

	return questions
}
