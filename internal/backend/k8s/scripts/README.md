# Kubernetes Backend Scripts

This directory contains bash scripts that are embedded into the BACA binary using Go's `//go:embed` directive.

## Scripts

### fork-setup.sh
Runs in the **fork-setup init container**. Creates or syncs a fork of the target repository.

**Environment Variables:**
- `ORIGINAL_REPO_URL`: Target repository URL (e.g., `https://github.com/org/repo`)
- `FORK_ORG`: (Optional) Organization to create fork under (default: authenticated user)
- `GITHUB_TOKEN`: GitHub token for authentication

**Outputs:**
- `/workspace/fork-url.txt`: Fork URL for next container

**Exit Codes:**
- `0`: Success (fork created or synced)
- `1`: Error (e.g., repo exists but is not a fork)

### job-runner.sh
Runs in the **main job container**. Executes the AI agent, commits changes, and creates a pull request.

**Environment Variables:**
- `CONFIG`: JSON config with agent, prompt, resources
- `ORIGINAL_REPO_URL`: Target repository URL
- `GITHUB_TOKEN`: GitHub token for git operations
- `COPILOT_TOKEN`: (Optional) Copilot-specific token
- `PROMPT`: Natural language prompt for agent

**Outputs:**
- Creates branch in fork
- Commits changes
- Pushes to fork
- Creates PR to original repository

**Exit Codes:**
- `0`: Success (PR created) or no changes
- Non-zero: Error

## Testing Scripts Locally

You can test these scripts standalone:

```bash
# Test fork setup
export ORIGINAL_REPO_URL="https://github.com/some/repo"
export GITHUB_TOKEN="ghp_xxx"
export FORK_ORG="my-org"  # optional
bash fork-setup.sh

# Test job runner (requires cloned repo in /workspace/repo)
export CONFIG='{"agent":"copilot-cli","prompt":"test"}'
export ORIGINAL_REPO_URL="https://github.com/some/repo"
export GITHUB_TOKEN="ghp_xxx"
bash job-runner.sh
```

## Linting

Use shellcheck to validate scripts:

```bash
shellcheck internal/backend/k8s/scripts/*.sh
```

## Embedding

These scripts are embedded at compile time via:

```go
//go:embed scripts/fork-setup.sh
var forkSetupScript string

//go:embed scripts/job-runner.sh
var jobScript string
```

This means:
- Scripts are baked into the binary (no runtime file dependencies)
- Single binary deployment
- Scripts must be present when building BACA
