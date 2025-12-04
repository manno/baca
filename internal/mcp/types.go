package mcp

import "time"

// Source represents an MCP source type
type Source string

const (
	SourceGitHub Source = "github"
	SourceSlack  Source = "slack"
	SourceManual Source = "manual"
)

// ContextItem represents a piece of context gathered from an MCP source
type ContextItem struct {
	Source     Source            // Where this context came from
	Type       string            // Type of context (issue, pr, message, etc.)
	ID         string            // Unique identifier (issue number, PR number, etc.)
	URL        string            // URL to the source
	Title      string            // Title or summary
	Content    string            // Full content
	Author     string            // Author/creator
	Metadata   map[string]string // Additional metadata
	GatheredAt time.Time         // When this was gathered
}

// Client defines the interface for MCP clients
type Client interface {
	// Name returns the source name
	Name() Source

	// GatherContext gathers context based on a query
	GatherContext(query string, repos []string) ([]ContextItem, error)

	// IsAvailable checks if the client is properly configured
	IsAvailable() bool
}
