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
	"strings"

	"github.com/manno/background-coding-agent/internal/change"
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

	// Generate PR metadata after agent completes
	if err := e.generatePRMetadata(ctx, c); err != nil {
		e.logger.Error("failed to generate PR metadata", "error", err)
		// Don't fail the job if PR metadata generation fails
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

	agentCommand := GetCommand(c.Spec.Agent)

	var cmd *exec.Cmd

	switch c.Spec.Agent {
	case "copilot-cli":
		// For copilot-cli, combine all args: add directories, prompt, and allow tools
		e.logger.Info("running copilot in interactive mode")
		cmd = exec.CommandContext(ctx, agentCommand,
			"--add-dir", "/workspace",
			"--add-dir", "/tmp",
			"--silent",
			"-p", c.Spec.Prompt,
			"--allow-all-tools")

	case "gemini-cli":
		// For gemini-cli, pass prompt directly
		cmd = exec.CommandContext(ctx, agentCommand, c.Spec.Prompt)

	default:
		return fmt.Errorf("unsupported agent: %s", c.Spec.Agent)
	}

	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	e.logger.Info("executing agent command")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	return nil
}

func (e *Executor) generatePRMetadata(ctx context.Context, c *change.Change) error {
	e.logger.Info("generating PR metadata")

	// Extract prompt without everything after ---
	promptClean := c.Spec.Prompt
	if idx := strings.Index(promptClean, "\n---\n"); idx != -1 {
		promptClean = promptClean[:idx]
	}

	// Prepare prompt for agent to generate PR description
	prPrompt := fmt.Sprintf(`Review the git diff and create a pull request title and description.

Requirements:
- Title: One line, clear and descriptive (max 72 chars)
- Body: Summarize what changed and why (2-4 sentences)
- Add a "## Prompt" section at the end with the original prompt

Original prompt:
%s

Format your response EXACTLY as:
TITLE: <your title here>
BODY:
<your description here>

## Prompt
%s`, promptClean, promptClean)

	// Run the agent to generate PR description
	agentCommand := GetCommand(c.Spec.Agent)
	var cmd *exec.Cmd

	switch c.Spec.Agent {
	case "copilot-cli":
		cmd = exec.CommandContext(ctx, agentCommand,
			"--add-dir", e.workDir,
			"-p", prPrompt,
			"--allow-all-tools")
	case "gemini-cli":
		cmd = exec.CommandContext(ctx, agentCommand, prPrompt)
	default:
		return fmt.Errorf("unsupported agent: %s", c.Spec.Agent)
	}

	cmd.Dir = e.workDir
	cmd.Stderr = os.Stderr // Send stderr to logs, not to PR metadata
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("agent failed to generate PR metadata: %w", err)
	}

	// Write the output to /workspace/pr-metadata.txt
	metadataPath := "/workspace/pr-metadata.txt"
	if err := os.WriteFile(metadataPath, output, 0600); err != nil {
		return fmt.Errorf("failed to write PR metadata: %w", err)
	}

	e.logger.Info("PR metadata generated", "path", metadataPath)
	return nil
}
