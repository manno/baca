package agent

// Config holds configuration for coding agents
type Config struct {
	Name        string   // Logical agent name (e.g., "gemini-cli")
	Command     string   // Actual command to execute (e.g., "gemini")
	Credentials []string // Required credentials for this agent
}

// AgentConfigs maps agent names to their configurations
var AgentConfigs = map[string]Config{
	"gemini-cli": {
		Name:    "gemini-cli",
		Command: "gemini",
		// Credentials: Either GEMINI_API_KEY OR GEMINI_oauth_creds_json + others
		// API Key: https://aistudio.google.com/apikey
		// OAuth: Authenticate via `gemini` CLI first, then use --gemini-oauth
		Credentials: []string{"GEMINI_API_KEY", "GEMINI_oauth_creds_json"},
	},
	"copilot-cli": {
		Name:    "copilot-cli",
		Command: "copilot",
		// Credentials: COPILOT_TOKEN or GITHUB_TOKEN (in order of precedence)
		// Generate at: https://github.com/settings/personal-access-tokens/new
		// Required permissions: "Copilot Requests" read/write
		// Note: If COPILOT_TOKEN not provided, falls back to GITHUB_TOKEN
		//       Ensure your token has "Copilot Requests" permission
		Credentials: []string{"COPILOT_TOKEN", "GITHUB_TOKEN"},
	},
}

// GetConfig returns the configuration for a given agent name
func GetConfig(agentName string) (Config, bool) {
	config, ok := AgentConfigs[agentName]
	return config, ok
}

// GetCommand returns the command to execute for a given agent name
func GetCommand(agentName string) string {
	if config, ok := AgentConfigs[agentName]; ok {
		return config.Command
	}
	// Fallback: return agent name as command
	return agentName
}
