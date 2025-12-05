package mcp

import (
	"fmt"
	"log/slog"
	"strings"
)

// Manager manages multiple MCP clients
type Manager struct {
	clients map[Source]Client
	logger  *slog.Logger
}

// NewManager creates a new MCP manager
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		clients: make(map[Source]Client),
		logger:  logger,
	}
}

// RegisterClient registers an MCP client
func (m *Manager) RegisterClient(client Client) {
	m.clients[client.Name()] = client
	m.logger.Info("registered MCP client", "source", client.Name())
}

// GetClient returns a client by source name
func (m *Manager) GetClient(source Source) (Client, bool) {
	client, ok := m.clients[source]
	return client, ok
}

// GatherContext gathers context from specified sources
func (m *Manager) GatherContext(sources []Source, query string, repos []string) ([]ContextItem, error) {
	var allItems []ContextItem

	for _, source := range sources {
		client, ok := m.clients[source]
		if !ok {
			m.logger.Warn("MCP client not registered", "source", source)
			continue
		}

		if !client.IsAvailable() {
			m.logger.Warn("MCP client not available", "source", source)
			continue
		}

		items, err := client.GatherContext(query, repos)
		if err != nil {
			m.logger.Error("failed to gather context from source",
				"source", source,
				"error", err)
			continue
		}

		allItems = append(allItems, items...)
	}

	m.logger.Info("gathered context from all sources",
		"total_items", len(allItems),
		"sources", len(sources))

	return allItems, nil
}

// GetAvailableSources returns list of available sources
func (m *Manager) GetAvailableSources() []Source {
	var available []Source
	for source, client := range m.clients {
		if client.IsAvailable() {
			available = append(available, source)
		}
	}
	return available
}

// ParseSources parses a comma-separated list of source names
func ParseSources(sourcesStr string) ([]Source, error) {
	if sourcesStr == "" {
		return nil, nil
	}

	parts := strings.Split(sourcesStr, ",")
	sources := make([]Source, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		source := Source(part)

		// Validate source
		switch source {
		case SourceGitHub, SourceSlack:
			sources = append(sources, source)
		default:
			return nil, fmt.Errorf("unknown MCP source: %s", part)
		}
	}

	return sources, nil
}
