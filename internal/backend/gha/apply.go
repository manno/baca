package gha

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v58/github"
	"github.com/manno/baca/internal/change"
	"golang.org/x/oauth2"
)

type GHA struct {
	logger *slog.Logger
	client *github.Client
}

func New(logger *slog.Logger) (*GHA, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable must be set")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GHA{
		logger: logger,
		client: client,
	}, nil
}

func (g *GHA) ApplyChange(ctx context.Context, ch *change.Change, workflowRepo string, workflowFileName string) error {
	parts := strings.Split(workflowRepo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format, expected 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	for _, repoURL := range ch.Spec.Repos {
		g.logger.Info("dispatching workflow for repo", "repo", repoURL)

		inputs := map[string]interface{}{
			"agent":     ch.Spec.Agent,
			"prompt":    ch.Spec.Prompt,
			"branch":    ch.Spec.Branch,
			"agentsmd":  ch.Spec.AgentsMD,
			"resources": strings.Join(ch.Spec.Resources, ","),
		}

		opts := github.CreateWorkflowDispatchEventRequest{
			Ref:    "main", // TODO: should this be configurable?
			Inputs: inputs,
		}

		resp, err := g.client.Actions.CreateWorkflowDispatchEventByFileName(ctx, owner, repo, workflowFileName, opts)
		if err != nil {
			g.logger.Error("failed to dispatch workflow", "error", err, "repo", repoURL)
			// Try to read the response body for more details
			if r, ok := err.(*github.ErrorResponse); ok {
				if r.Response.StatusCode == http.StatusNotFound {
					g.logger.Error("workflow not found, ensure it is in the default branch", "workflow", workflowFileName)
				}
			}
			return err
		}
		g.logger.Info("workflow dispatched successfully", "response_status", resp.Status)
	}
	return nil
}
