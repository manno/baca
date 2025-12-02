package git

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Inject GITHUB_TOKEN into URL if available
	authenticatedURL, err := c.injectToken(repoURL)
	if err != nil {
		return fmt.Errorf("failed to prepare repository URL: %w", err)
	}

	args := []string{"clone"}

	if branch != "" {
		args = append(args, "--branch", branch)
	}

	args = append(args, authenticatedURL, outputDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	c.logger.Info("clone completed successfully", "path", outputDir)
	return nil
}

// injectToken injects GITHUB_TOKEN into HTTPS URLs for authentication
func (c *Cloner) injectToken(repoURL string) (string, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		c.logger.Debug("no GITHUB_TOKEN found, cloning without authentication")
		return repoURL, nil
	}

	// Only inject token for HTTPS URLs
	if !strings.HasPrefix(repoURL, "https://") {
		return repoURL, nil
	}

	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repository URL: %w", err)
	}

	// Inject token as username in URL (GitHub uses token as username with empty password)
	u.User = url.UserPassword(token, "")

	return u.String(), nil
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
