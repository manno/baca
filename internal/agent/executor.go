package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mm/background-coding-agent/internal/change"
)

type Executor struct {
	logger  *slog.Logger
	workDir string
}

func NewExecutor(workDir string, logger *slog.Logger) *Executor {
	return &Executor{
		workDir: workDir,
		logger:  logger,
	}
}

func (e *Executor) Execute(ctx context.Context, c *change.Change) error {
	e.logger.Info("executing change", "agent", c.Spec.Agent, "workDir", e.workDir)

	if err := e.downloadResources(ctx, c); err != nil {
		return fmt.Errorf("failed to download resources: %w", err)
	}

	if err := e.runAgent(ctx, c); err != nil {
		return fmt.Errorf("failed to run agent: %w", err)
	}

	e.logger.Info("execution completed successfully")
	return nil
}

func (e *Executor) downloadResources(ctx context.Context, c *change.Change) error {
	if c.Spec.AgentsMD != "" {
		e.logger.Info("downloading agents.md", "url", c.Spec.AgentsMD)
		if err := e.downloadFile(ctx, c.Spec.AgentsMD, filepath.Join(e.workDir, "agents.md")); err != nil {
			return err
		}
	}

	for i, resource := range c.Spec.Resources {
		e.logger.Info("downloading resource", "url", resource, "index", i)
		filename := fmt.Sprintf("resource-%d.md", i)
		if err := e.downloadFile(ctx, resource, filepath.Join(e.workDir, filename)); err != nil {
			return err
		}
	}

	return nil
}

func (e *Executor) downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status %d", url, resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (e *Executor) runAgent(ctx context.Context, c *change.Change) error {
	e.logger.Info("running coding agent", "agent", c.Spec.Agent, "prompt", c.Spec.Prompt)

	var cmd *exec.Cmd
	switch c.Spec.Agent {
	case "gemini-cli", "copilot-cli":
		cmd = exec.CommandContext(ctx, c.Spec.Agent, c.Spec.Prompt)
	default:
		return fmt.Errorf("unsupported agent: %s", c.Spec.Agent)
	}

	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	return nil
}
