# Background Automated Coding Agent (BACA)

![BAKA!](docs/baka.jpg)

BACA is a declarative, prompt-driven code transformation platform that orchestrates AI coding agents (Copilot or Gemini) across multiple repositories simultaneously. Write a natural language prompt, specify your repositories, and BACA can use either Kubernetes Jobs or GitHub Actions to clone, transform, and submit pull requests automatically.

**Use cases:**
- Apply security fixes across dozens of microservices
- Refactor common patterns organization-wide
- Update dependencies with code changes
- Migrate APIs across all consuming services

## Execution Backends

BACA supports two execution backends:

- **Kubernetes (`k8s`)**: (Default) Executes transformations in Kubernetes Jobs. Powerful, scalable, and isolated. Requires access to a Kubernetes cluster.
- **GitHub Actions (`gha`)**: Executes transformations using GitHub Actions workflows. Lower barrier to entry, no Kubernetes required.

## Quick Start (Kubernetes)

### 1. Build BACA

```bash
git clone https://github.com/manno/baca
cd baca
go build -o baca .
```

### 2. Setup Credentials

```bash
export GITHUB_TOKEN=ghp_xxx         # Required: git clone, fork, PR creation
export COPILOT_TOKEN=github_pat_xxx # OR
export GEMINI_API_KEY=xxx           # Choose your agent

baca k8s setup --namespace baca-jobs
```

**Token requirements:**
- GitHub: `Contents` read/write, `Pull requests` read/write, `Metadata` read
- Copilot: `Copilot Requests` read/write (or reuse GitHub token)
- Gemini: API key from https://aistudio.google.com/apikey

### 3. Create Change Definition

`my-change.yaml`:
```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Add comprehensive error handling to all HTTP handlers"
  repos:
  - https://github.com/myorg/repo1
  - https://github.com/myorg/repo2
  agent: copilot-cli  # or gemini-cli
  branch: main        # optional, defaults to main
```

**Note:** Specify the target repositories you want to modify (e.g., `https://github.com/myorg/repo1`). BACA will automatically create forks in your account if they don't exist. If a repository with the same name already exists in your account but is NOT a fork, the job will fail with an error.

### 4. Apply

```bash
baca k8s apply my-change.yaml --namespace baca-jobs
```

Creates one Kubernetes job per repository.

## Quick Start (GitHub Actions)

### 1. Build BACA

```bash
git clone https://github.com/manno/baca
cd baca
go build -o baca .
```

### 2. Setup Workflow

In the repository where you want to run the transformations (or a central one), create the workflow file:

```bash
# This creates the workflow file locally
baca gha setup --workflow-path .github/workflows/baca-execute.yml

# Commit and push this file to your repository's default branch.
```

Then, add the following secrets to your repository's settings (`Settings > Secrets and variables > Actions`):
- `COPILOT_TOKEN`
- `GEMINI_API_KEY`

### 3. Create Change Definition

Same as the Kubernetes example.

### 4. Apply

Trigger the workflow for the repositories in your change file.

```bash
export GITHUB_TOKEN=ghp_xxx # Required to call the GitHub API

baca gha apply my-change.yaml --repo your-org/your-repo-with-workflow
```

## Commands

### `k8s`

Manage the Kubernetes backend.

- `baca k8s setup --namespace <ns> [--copilot-token | --gemini-api-key | --gemini-oauth]`: Setup Kubernetes backend with credentials.
- `baca k8s apply <change-file> --namespace <ns> [--wait] [--retries N] [--fork-org ORG]`: Execute code transformations using Kubernetes jobs.

### `gha`

Manage the GitHub Actions backend.

- `baca gha setup [--workflow-path <path>]`: Create the GitHub Actions workflow file locally.
- `baca gha apply <change-file> --repo <owner/repo>`: Execute code transformations by triggering a `workflow_dispatch` event.

## Change Definition

```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Natural language description"                 # REQUIRED
  repos: ["https://github.com/org/repo"]                # REQUIRED: Target repos (BACA auto-forks)
  agent: copilot-cli                                     # REQUIRED: copilot-cli or gemini-cli
  branch: main                                            # optional, default: main
  agentsmd: "https://example.com/agents.md"              # optional
  resources: ["https://example.com/docs.md"]             # optional
  image: ghcr.io/manno/baca-runner:latest                # optional (k8s only)
```

## Architecture

### Kubernetes Backend

```
┌─────────────┐
│   Change    │ User creates YAML definition
└──────┬──────┘
       │
       v
┌─────────────┐
│ baca k8s apply │ Creates Kubernetes Jobs (one per repo)
└──────┬──────┘
       │
       v
┌───────────────────────────────────────────┐
│  Kubernetes Job (per repository)          │
│                                           │
│  Init Container 1: fork-setup             │
│  └─ gh repo fork (create/sync fork)       │
│                                           │
│  Init Container 2: git-clone              │
│  └─ fleet gitcloner (clone fork)          │
│                                           │
│  Main Container: runner                   │
│  ├─ baca execute (run agent on fork)      │
│  ├─ git push (push to fork)               │
│  └─ gh pr create (fork → original repo)   │
└───────────────────────────────────────────┘
```

### GitHub Actions Backend

`baca gha apply` triggers a `workflow_dispatch` event on the repository specified with `--repo`. The workflow then checks out the code, runs `baca execute`, and creates a PR.

## Security Model (Staging Fork Approach)

BACA uses a **staging fork approach** to limit token exposure, especially in the Kubernetes backend.

1. **Fork isolation**: Changes are pushed to a fork in the authenticated user's account, not directly to target repos
2. **Token scope**: `GITHUB_TOKEN` only needs write access to user's forks and PR creation on target repos
3. **Cross-fork PRs**: Pull requests are created from `user-fork:branch` → `original-repo:main`

## Supported Agents

- **copilot-cli**: GitHub Copilot (requires token with Copilot Requests permission)
- **gemini-cli**: Google Gemini (requires API key or OAuth)

Add new agents in `internal/agent/config.go`.

## Development

Build:
```bash
./dev/build-release.sh           # Build CLI binaries
./dev/build-runner-image.sh      # Build Docker image
./dev/import-image-k3d.sh        # Import to k3d cluster
```

Test:
```bash
go test ./internal/...               # Unit tests
ginkgo -v ./tests/...                # Integration tests
```