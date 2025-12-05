package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// GitHubClient gathers context from GitHub using gh CLI
type GitHubClient struct {
	logger *slog.Logger
}

// NewGitHubClient creates a new GitHub MCP client
func NewGitHubClient(logger *slog.Logger) *GitHubClient {
	return &GitHubClient{
		logger: logger,
	}
}

// Name returns the source name
func (c *GitHubClient) Name() Source {
	return SourceGitHub
}

// IsAvailable checks if gh CLI is available
func (c *GitHubClient) IsAvailable() bool {
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		c.logger.Debug("gh CLI not available or not authenticated", "error", err)
		return false
	}
	return true
}

// GatherContext gathers issues and PRs from GitHub repositories
func (c *GitHubClient) GatherContext(query string, repos []string) ([]ContextItem, error) {
	ctx := context.Background()
	var items []ContextItem

	for _, repo := range repos {
		// Extract owner/repo from URL
		ownerRepo := extractOwnerRepo(repo)
		if ownerRepo == "" {
			c.logger.Warn("could not extract owner/repo from URL", "repo", repo)
			continue
		}

		c.logger.Info("gathering GitHub context", "repo", ownerRepo, "query", query)

		// Gather issues
		issues, err := c.searchIssues(ctx, ownerRepo, query)
		if err != nil {
			c.logger.Error("failed to search issues", "repo", ownerRepo, "error", err)
			// Continue with other repos
		} else {
			items = append(items, issues...)
		}

		// Gather PRs
		prs, err := c.searchPullRequests(ctx, ownerRepo, query)
		if err != nil {
			c.logger.Error("failed to search PRs", "repo", ownerRepo, "error", err)
			// Continue with other repos
		} else {
			items = append(items, prs...)
		}
	}

	c.logger.Info("gathered GitHub context", "items", len(items))
	return items, nil
}

// searchIssues searches for issues using gh CLI
func (c *GitHubClient) searchIssues(ctx context.Context, repo, query string) ([]ContextItem, error) {
	cmd := exec.CommandContext(ctx, "gh", "issue", "list",
		"--repo", repo,
		"--search", query,
		"--limit", "5",
		"--json", "number,title,body,url,author,state,createdAt")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh issue list failed: %w", err)
	}

	var ghIssues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		URL    string `json:"url"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		State     string    `json:"state"`
		CreatedAt time.Time `json:"createdAt"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &ghIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issue JSON: %w", err)
	}

	items := make([]ContextItem, 0, len(ghIssues))
	for _, issue := range ghIssues {
		items = append(items, ContextItem{
			Source:  SourceGitHub,
			Type:    "issue",
			ID:      fmt.Sprintf("#%d", issue.Number),
			URL:     issue.URL,
			Title:   issue.Title,
			Content: fmt.Sprintf("**Issue #%d: %s** (State: %s)\n\n%s", issue.Number, issue.Title, issue.State, issue.Body),
			Author:  issue.Author.Login,
			Metadata: map[string]string{
				"number": fmt.Sprintf("%d", issue.Number),
				"state":  issue.State,
				"repo":   repo,
			},
			GatheredAt: time.Now(),
		})
	}

	return items, nil
}

// searchPullRequests searches for PRs using gh CLI
func (c *GitHubClient) searchPullRequests(ctx context.Context, repo, query string) ([]ContextItem, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "list",
		"--repo", repo,
		"--search", query,
		"--limit", "5",
		"--json", "number,title,body,url,author,state,createdAt")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh pr list failed: %w", err)
	}

	var ghPRs []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		URL    string `json:"url"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		State     string    `json:"state"`
		CreatedAt time.Time `json:"createdAt"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &ghPRs); err != nil {
		return nil, fmt.Errorf("failed to parse PR JSON: %w", err)
	}

	items := make([]ContextItem, 0, len(ghPRs))
	for _, pr := range ghPRs {
		items = append(items, ContextItem{
			Source:  SourceGitHub,
			Type:    "pull_request",
			ID:      fmt.Sprintf("#%d", pr.Number),
			URL:     pr.URL,
			Title:   pr.Title,
			Content: fmt.Sprintf("**PR #%d: %s** (State: %s)\n\n%s", pr.Number, pr.Title, pr.State, pr.Body),
			Author:  pr.Author.Login,
			Metadata: map[string]string{
				"number": fmt.Sprintf("%d", pr.Number),
				"state":  pr.State,
				"repo":   repo,
			},
			GatheredAt: time.Now(),
		})
	}

	return items, nil
}

// extractOwnerRepo extracts "owner/repo" from a GitHub URL
func extractOwnerRepo(repoURL string) string {
	// Remove trailing .git
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Handle https://github.com/owner/repo
	if strings.Contains(repoURL, "github.com/") {
		parts := strings.Split(repoURL, "github.com/")
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// Handle git@github.com:owner/repo
	if strings.Contains(repoURL, "git@github.com:") {
		parts := strings.Split(repoURL, "git@github.com:")
		if len(parts) == 2 {
			return parts[1]
		}
	}

	return ""
}
