package git

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

type Cloner struct {
	logger *slog.Logger
}

func NewCloner(logger *slog.Logger) *Cloner {
	return &Cloner{logger: logger}
}

func (c *Cloner) Clone(repoURL, outputDir, branch string) error {
	c.logger.Info("cloning repository", "url", repoURL, "output", outputDir, "branch", branch)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	args := []string{"clone"}

	if branch != "" {
		args = append(args, "--branch", branch)
	}

	args = append(args, repoURL, outputDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	c.logger.Info("clone completed successfully", "path", outputDir)
	return nil
}

func (c *Cloner) CreateBranch(repoPath, branchName string) error {
	c.logger.Info("creating branch", "path", repoPath, "branch", branchName)

	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

func (c *Cloner) ExtractRepoName(repoURL string) string {
	base := filepath.Base(repoURL)
	if len(base) > 4 && base[len(base)-4:] == ".git" {
		return base[:len(base)-4]
	}
	return base
}
