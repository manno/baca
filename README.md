# Background Coding Agent (BCA)

A platform for executing complex code transformations across multiple repositories using AI coding agents and natural language prompts.

## ⚠️  Security Notice

**CRITICAL: API tokens and credentials have very sensitive permissions!**

Never commit tokens to source control. Always use environment variables or secure secret management. See [Security Best Practices](AGENTS.md#security-best-practices) for complete guidance.

## Prerequisites

Before using BCA, ensure you have:

- **Kubernetes Cluster**: Local (k3d, minikube) or remote cluster
- **kubectl**: Configured to access your cluster
- **GitHub Token**: Personal Access Token with permissions:
  - `Contents`: read/write (for git operations)
  - `Pull requests`: read/write (for creating PRs)
- **Copilot Token** (if using copilot-cli): Fine-grained PAT with:
  - `Copilot Requests`: read/write
  - Generate at: https://github.com/settings/personal-access-tokens/new
- **Gemini API Key** (if using gemini-cli):
  - Generate at: https://aistudio.google.com/apikey
  - OR authenticate gemini CLI locally for OAuth

## Quick Start

### 1. Install BCA

```bash
# Clone the repository
git clone https://github.com/manno/background-coding-agent
cd background-coding-agent

# Build the CLI
go build -o bca .
```

### 2. Setup Credentials

**For Copilot CLI:**
```bash
export GITHUB_TOKEN=ghp_your_git_pr_token      # For git/PR operations
export COPILOT_TOKEN=github_pat_your_copilot   # For Copilot CLI
```

**For Gemini CLI (API Key):**
```bash
export GITHUB_TOKEN=ghp_your_git_pr_token      # For git/PR operations
export GEMINI_API_KEY=your_gemini_key          # For Gemini CLI
```

**For Gemini CLI (OAuth):**
```bash
export GITHUB_TOKEN=ghp_your_git_pr_token      # For git/PR operations
gemini auth                                     # Authenticate Gemini locally
```

### 3. Setup the Backend

```bash
# For Copilot
bca setup --namespace bca-jobs

# For Gemini with API key
bca setup --namespace bca-jobs --gemini-api-key $GEMINI_API_KEY

# For Gemini with OAuth
bca setup --namespace bca-jobs --gemini-oauth
```

This creates:
- Kubernetes namespace: `bca-jobs`
- Secret `bca-credentials` with your tokens
- Environment ready for running coding agent jobs

### 4. Create a Change Definition

Create `my-change.yaml`:

```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Add comprehensive error handling to all HTTP handlers"
  repos:
  - https://github.com/myorg/repo1
  - https://github.com/myorg/repo2
  agent: copilot-cli  # or gemini-cli
```

### 5. Apply the Change

```bash
bca apply my-change.yaml --namespace bca-jobs
```

This will:
1. Create one Kubernetes Job per repository
2. Each job: clones repo → runs AI agent → creates pull request
3. Monitor and report job status
4. Jobs auto-cleanup after 1 hour

## Commands

### bca setup

Setup the Kubernetes backend with credentials.

```bash
bca setup --namespace <namespace> [options]
```

**Options:**
- `--github-token` - GitHub token for git/PR (default: $GITHUB_TOKEN)
- `--copilot-token` - Copilot token (default: $COPILOT_TOKEN, falls back to GITHUB_TOKEN)
- `--gemini-api-key` - Gemini API key (default: $GEMINI_API_KEY)
- `--gemini-oauth` - Copy OAuth creds from ~/.gemini/
- `--kubeconfig` - Path to kubeconfig file

**Examples:**
```bash
# Copilot with separate tokens
bca setup --namespace prod --copilot-token $COPILOT_TOKEN

# Gemini with API key
bca setup --namespace dev --gemini-api-key $GEMINI_API_KEY

# Gemini with OAuth
bca setup --namespace dev --gemini-oauth
```

### bca apply

Apply a Change definition to execute code transformations.

```bash
bca apply <change-file> --namespace <namespace> [options]
```

**Options:**
- `--wait` - Wait for jobs to complete (default: true)
- `--kubeconfig` - Path to kubeconfig file

**Examples:**
```bash
# Apply and wait for completion
bca apply my-change.yaml --namespace prod

# Apply and return immediately
bca apply my-change.yaml --namespace prod --wait=false
```

## Change Definition Reference

```yaml
kind: Change
apiVersion: v1
spec:
  # Natural language prompt (REQUIRED)
  prompt: "Detailed description of code transformation task"

  # Target repositories (REQUIRED)
  repos:
  - https://github.com/org/repo1
  - https://github.com/org/repo2

  # AI agent to use (REQUIRED)
  agent: copilot-cli  # or gemini-cli

  # Additional context (OPTIONAL)
  agentsmd: https://example.com/agent-instructions.md
  resources:
  - https://example.com/docs/style-guide.md
  - https://example.com/docs/architecture.md

  # Custom runner image (OPTIONAL)
  image: ghcr.io/manno/background-coder:latest
```

### Fields

- **prompt**: Natural language description of the transformation
- **repos**: List of GitHub repository URLs to transform
- **agent**: AI agent to use (`copilot-cli` or `gemini-cli`)
- **agentsmd**: URL to markdown file with agent-specific instructions
- **resources**: List of URLs to additional documentation
- **image**: Container image for the runner (default: ghcr.io/manno/background-coder:latest)

## How It Works

```
User creates Change → BCA creates K8s Jobs → Each Job:
                                            ├─ Clones repository
                                            ├─ Downloads resources
                                            ├─ Runs AI coding agent
                                            ├─ Creates pull request
                                            └─ Auto-cleanup (1 hour)
```

### Job Lifecycle

1. **Creation**: One job per repository
2. **Execution**: Clone → Transform → PR creation
3. **Monitoring**: Real-time status updates
4. **Retry**: Up to 3 automatic retries on failure
5. **Cleanup**: Jobs auto-delete 1 hour after completion

## Supported AI Agents

### Copilot CLI

**Requirements:**
- GitHub token with "Copilot Requests" permission
- Node.js v20+ in runner image

**Example:**
```yaml
agent: copilot-cli
```

### Gemini CLI

**Requirements:**
- Gemini API key OR OAuth authentication
- Node.js v20+ in runner image

**Example with API key:**
```yaml
agent: gemini-cli
```

**Example with OAuth:**
```bash
# Authenticate locally first
gemini auth
# Then use --gemini-oauth during setup
bca setup --gemini-oauth
```

## Troubleshooting

### Authentication failures

**Problem:** Git clone or PR creation fails

**Solution:** Verify token permissions:
```bash
# Test GitHub token
gh auth status

# Verify token in cluster
kubectl get secret bca-credentials -n <namespace> -o jsonpath='{.data}' | jq 'keys'
```

### Jobs accumulating in cluster

**Problem:** Old jobs not cleaning up

**Solution:** Jobs auto-delete after 1 hour (TTL=3600). To clean up immediately:
```bash
kubectl delete jobs -n <namespace> --all
```

## Development

### Building from Source

```bash
# Build CLI
./scripts/build-release.sh

# Build runner image
./scripts/build-runner-image.sh

# Import to k3d for testing
./scripts/import-image-k3d.sh
```

### Running Tests

```bash
# Unit tests
go test ./internal/...

# Integration tests (requires envtest)
ginkgo -v ./tests/...
```

See [Development Guide](AGENTS.md) for detailed instructions.

## Architecture

BCA uses a Kubernetes-native architecture:

- **CLI**: User interface for setup and change application
- **Backend**: Kubernetes controllers and job management
- **Runner**: Container that executes transformations
- **Agents**: AI coding assistants (Copilot, Gemini)

## Security Best Practices

🔐 **Token Security:**
- Never commit tokens to source control
- Use environment variables or secret managers
- Rotate tokens regularly (90 days recommended)
- Use separate tokens for dev/staging/production
- Enable Kubernetes secret encryption at rest

## Known Issues

- **Private Repos**: Currently optimized for public repos (Fleet gitcloner supports auth)
