package workflow

import "time"

// Session represents an interactive workflow session
type Session struct {
	ID              string
	InitialPrompt   string
	ConversationLog []Message
	GatheredContext []Context
	RefinedPrompt   string
	Complete        bool
	CreatedAt       time.Time
}

// Message represents a single interaction in the workflow
type Message struct {
	Role      string // "agent" or "user"
	Content   string
	Timestamp time.Time
}

// Context represents gathered context from various sources
type Context struct {
	Source     string            // "github", "slack", "manual", etc.
	Type       string            // "issue", "pr", "message", "file", etc.
	URL        string            // Optional URL to the source
	Content    string            // The actual content
	Metadata   map[string]string // Additional metadata
	GatheredAt time.Time
}
