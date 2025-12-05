package cmd

import (
	"fmt"
	"os"

	"github.com/manno/baca/internal/backend/k8s"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up the execution backend",
	Long: `Set up the execution backend (Kubernetes cluster).
Creates necessary secrets to allow execution runners to clone git repos,
create pull requests, and run coding agents.

Required credentials (via flags or environment variables):
  GITHUB_TOKEN - GitHub personal access token for:
    - Git repository cloning (Fleet gitcloner)
    - Git push to create branches (git push origin)
    - Pull request creation (gh CLI)
    
    Fine-grained PAT (recommended):
      Generate at: https://github.com/settings/personal-access-tokens/new
      Required permissions: 
        - "Contents" read/write (for cloning and pushing)
        - "Pull requests" read/write (for creating PRs)
        - "Metadata" read (automatically included)
    
    Classic PAT:
      Generate at: https://github.com/settings/tokens/new
      Required scopes: repo, read:org

Copilot CLI authentication (if using copilot-cli agent):
  --copilot-token or COPILOT_TOKEN - GitHub token for Copilot CLI
    
    Fine-grained PAT (recommended):
      Generate at: https://github.com/settings/personal-access-tokens/new
      Required permissions: "Copilot Requests" read/write
    
    Classic PAT:
      Generate at: https://github.com/settings/tokens/new
      Required scopes: repo, read:org (same as GITHUB_TOKEN)
    
    Note: Can use same token as GITHUB_TOKEN if it has all required scopes

Gemini authentication (if using gemini-cli agent, choose one):
  --gemini-api-key or GEMINI_API_KEY - Gemini API key for gemini-cli
    Generate at: https://aistudio.google.com/apikey
  --gemini-oauth - Copy OAuth credentials from ~/.gemini/ directory
    Authenticate gemini CLI first, then use this flag`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		logger.Info("setting up execution backend")

		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")
		githubToken, _ := cmd.Flags().GetString("github-token")
		copilotToken, _ := cmd.Flags().GetString("copilot-token")
		googleAPIKey, _ := cmd.Flags().GetString("gemini-api-key")
		useGeminiOAuth, _ := cmd.Flags().GetBool("gemini-oauth")

		// Fallback to environment variables if flags not provided
		if githubToken == "" {
			githubToken = os.Getenv("GITHUB_TOKEN")
		}
		if copilotToken == "" {
			copilotToken = os.Getenv("COPILOT_TOKEN")
		}
		if googleAPIKey == "" {
			googleAPIKey = os.Getenv("GEMINI_API_KEY")
		}

		if githubToken == "" {
			logger.Error("github token is required")
			return fmt.Errorf("github token is required: use --github-token flag or GITHUB_TOKEN env var")
		}

		// Build credentials map
		credentials := map[string]string{
			"GITHUB_TOKEN": githubToken,
		}

		// Add copilot token if provided (separate from GITHUB_TOKEN)
		if copilotToken != "" {
			credentials["COPILOT_TOKEN"] = copilotToken
			logger.Info("using separate copilot token")
		} else {
			logger.Info("copilot will use GITHUB_TOKEN (ensure it has Copilot Requests permission)")
		}

		// Handle gemini authentication
		if googleAPIKey != "" && useGeminiOAuth {
			logger.Error("cannot use both --gemini-api-key and --gemini-oauth")
			return fmt.Errorf("choose either API key or OAuth authentication for gemini-cli")
		}

		if googleAPIKey != "" {
			credentials["GEMINI_API_KEY"] = googleAPIKey
			logger.Info("using gemini api key authentication")
		}

		// Gemini OAuth files to copy if --gemini-oauth is set
		var geminiFiles map[string]string
		if useGeminiOAuth {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			geminiDir := homeDir + "/.gemini"
			geminiFiles = map[string]string{
				"oauth_creds.json":     geminiDir + "/oauth_creds.json",
				"google_accounts.json": geminiDir + "/google_accounts.json",
				"installation_id":      geminiDir + "/installation_id",
				"settings.json":        geminiDir + "/settings.json",
			}

			// Read and validate files exist
			for key, path := range geminiFiles {
				content, err := os.ReadFile(path)
				if err != nil {
					logger.Warn("failed to read gemini file", "file", path, "error", err)
					return fmt.Errorf("failed to read %s: %w (ensure gemini-cli is authenticated)", key, err)
				}
				credentials["GEMINI_"+key] = string(content)
			}
			logger.Info("using gemini oauth authentication", "files", len(geminiFiles))
		}

		cfg, err := k8s.GetConfig(kubeconfig)
		if err != nil {
			logger.Error("failed to get kubernetes config", "error", err)
			return err
		}

		backend, err := k8s.New(cfg, namespace, logger)
		if err != nil {
			logger.Error("failed to create backend", "error", err)
			return err
		}

		ctx := cmd.Context()
		if err := backend.Setup(ctx, credentials); err != nil {
			logger.Error("failed to setup backend", "error", err)
			return err
		}

		logger.Info("setup completed")
		return nil
	},
}

func init() {
	k8sCmd.AddCommand(setupCmd)

	setupCmd.Flags().String("kubeconfig", "", "path to kubeconfig file")
	setupCmd.Flags().String("namespace", "default", "kubernetes namespace")
	setupCmd.Flags().String("github-token", "", "GitHub token for git/PR operations (defaults to GITHUB_TOKEN env var)")
	setupCmd.Flags().String("copilot-token", "", "GitHub token for Copilot CLI (defaults to COPILOT_TOKEN env var, or uses GITHUB_TOKEN)")
	setupCmd.Flags().String("gemini-api-key", "", "Gemini API key for gemini-cli (defaults to GEMINI_API_KEY env var)")
	setupCmd.Flags().Bool("gemini-oauth", false, "Copy OAuth credentials from ~/.gemini/ for gemini authentication")
}
