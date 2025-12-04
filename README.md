# Background Automated Coding Agent (BACA)

BACA is a declarative, prompt-driven code transformation platform that orchestrates AI coding agents (Copilot or Gemini) across multiple repositories simultaneously. Write a natural language prompt, specify your repositories, and BACA creates Kubernetes jobs that clone, transform, and submit pull requests automatically.

**Use cases:**
- Apply security fixes across dozens of microservices
- Refactor common patterns organization-wide
- Update dependencies with code changes
- Migrate APIs across all consuming services

## Quick Start

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

baca setup --namespace baca-jobs
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

### 4. Apply

```bash
baca apply my-change.yaml --namespace baca-jobs
```

Creates one Kubernetes job per repository.

## Prerequisites

- Kubernetes cluster (k3d, minikube, or remote) with kubectl configured
- Go 1.25+ (for building from source)

## Commands

### setup

Setup Kubernetes backend with credentials.

```bash
baca setup --namespace <ns> [--copilot-token | --gemini-api-key | --gemini-oauth]
```

### apply

Execute code transformations.

```bash
baca apply <change-file> --namespace <ns> [--wait]
```

Options:
- `--wait`: Wait for completion (default: true)

## Change Definition

```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Natural language description"       # REQUIRED
  repos: ["https://github.com/org/repo"]      # REQUIRED
  agent: copilot-cli                          # REQUIRED: copilot-cli or gemini-cli
  branch: main                                 # optional, default: main
  agentsmd: "https://example.com/agents.md"   # optional
  resources: ["https://example.com/docs.md"]  # optional
  image: ghcr.io/manno/baca-runner:latest # optional
```

## Architecture

```
┌─────────────┐
│   Change    │ User creates YAML definition
└──────┬──────┘
       │
       v
┌─────────────┐
│ baca apply  │ Creates Kubernetes Jobs (one per repo)
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

### Security Model (Staging Fork Approach)

BACA uses a **staging fork approach** to limit token exposure:

1. **Fork isolation**: Changes are pushed to a fork in the authenticated user's account, not directly to target repos
2. **Token scope**: `GITHUB_TOKEN` only needs write access to user's forks and PR creation on target repos
3. **Cross-fork PRs**: Pull requests are created from `user-fork:branch` → `original-repo:main`

**What this protects against:**
- Malicious prompts cannot directly push to production repos
- Fork serves as isolation boundary for untrusted code execution

**Remaining considerations for shared usage:**
- Tokens can still create PRs (potential for spam)
- Agent API tokens (Copilot/Gemini) are still exposed to job environment
- No isolation between different users' jobs in same namespace

## How It Works

Each repository gets a Kubernetes job with three containers:

1. **Init: fork-setup** - Creates/syncs fork in user's GitHub account
2. **Init: git-clone** - Clones fork to shared `/workspace` volume
3. **Main: runner** - Runs AI agent, commits changes, pushes to fork, creates PR

Configuration passed as JSON via environment variable. Jobs auto-cleanup after 5 minutes. No retries by default (configurable with `--retries`).

## Supported Agents

- **copilot-cli**: GitHub Copilot (requires token with Copilot Requests permission)
- **gemini-cli**: Google Gemini (requires API key or OAuth)

Add new agents in `internal/agent/config.go`.

## Troubleshooting

**Jobs fail with authentication errors:**
```bash
kubectl get secret baca-credentials -n <namespace> -o yaml
kubectl logs -n <namespace> <job-pod> --all-containers
```

**Clean up failed jobs:**
```bash
kubectl delete jobs -n <namespace> --all
```

**Check job status:**
```bash
kubectl get jobs -n <namespace>
kubectl describe job <job-name> -n <namespace>
```

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

**Documentation:**
- `tests/README.md` - Testing documentation
- `dev/README.md` - Development scripts (building, testing, not for releases)
- `AGENTS.md` - AI assistant guide (for AI agents working on this project, see https://agents.md/)

## Files

- `cmd/` - CLI commands (setup, apply, execute)
- `internal/backend/` - Kubernetes job management
- `internal/agent/` - Agent executor and configuration
- `internal/change/` - Change definition parser
- `Dockerfile` - Runner image with tools (gh, fleet, gemini, copilot)
- `tests/` - Integration tests with envtest
