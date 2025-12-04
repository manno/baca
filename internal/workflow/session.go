package workflow

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NewSession creates a new interactive workflow session
func NewSession(initialPrompt string) *Session {
	return &Session{
		ID:              uuid.New().String(),
		InitialPrompt:   initialPrompt,
		ConversationLog: make([]Message, 0),
		GatheredContext: make([]Context, 0),
		Complete:        false,
		CreatedAt:       time.Now(),
	}
}

// AddMessage adds a message to the conversation log
func (s *Session) AddMessage(role, content string) {
	s.ConversationLog = append(s.ConversationLog, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// AddContext adds gathered context to the session
func (s *Session) AddContext(ctx Context) {
	ctx.GatheredAt = time.Now()
	s.GatheredContext = append(s.GatheredContext, ctx)
}

// SetRefinedPrompt sets the final refined prompt and marks session complete
func (s *Session) SetRefinedPrompt(prompt string) {
	s.RefinedPrompt = prompt
	s.Complete = true
}

// IsComplete returns whether the workflow session is complete
func (s *Session) IsComplete() bool {
	return s.Complete
}

// GetConversationHistory returns the full conversation as a formatted string
func (s *Session) GetConversationHistory() string {
	var history string
	for _, msg := range s.ConversationLog {
		history += fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp.Format(time.RFC3339), msg.Role, msg.Content)
	}
	return history
}

// GetContextSummary returns a summary of all gathered context
func (s *Session) GetContextSummary() string {
	if len(s.GatheredContext) == 0 {
		return "No additional context gathered."
	}

	var summary string
	summary += fmt.Sprintf("Gathered %d context items:\n", len(s.GatheredContext))
	for i, ctx := range s.GatheredContext {
		summary += fmt.Sprintf("%d. [%s:%s] %s\n", i+1, ctx.Source, ctx.Type, ctx.Content[:min(100, len(ctx.Content))])
		if ctx.URL != "" {
			summary += fmt.Sprintf("   URL: %s\n", ctx.URL)
		}
	}
	return summary
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
