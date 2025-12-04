# Background Coding Agent (BCA)

Execute AI-driven code transformations across multiple repositories using natural language prompts.

## ⚠️  Security Notice

**Never commit API tokens to source control.** Use environment variables or secret managers. See [AGENTS.md](AGENTS.md#security-best-practices).

## Prerequisites

- **Kubernetes cluster** (k3d, minikube, or remote)
- **kubectl** configured
- **GitHub Token** with repository access:
  - Fine-grained PAT: `Contents` read/write, `Pull requests` read/write, `Metadata` read
  - Classic PAT: `repo`, `read:org` scopes
- **Agent credentials:**
  - Copilot: Fine-grained PAT with `Copilot Requests` read/write OR Classic PAT with `repo`, `read:org`
  - Gemini: API key from https://aistudio.google.com/apikey

## Quick Start

### 1. Build BCA

```bash
git clone https://github.com/manno/background-coding-agent
cd background-coding-agent
go build -o bca .
```

### 2. Setup Backend

```bash
export GITHUB_TOKEN=ghp_xxx        # For git/PR operations
export COPILOT_TOKEN=github_pat_xxx # OR
export GEMINI_API_KEY=xxx           # Choose your agent

bca setup --namespace bca-jobs
```

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
bca apply my-change.yaml --namespace bca-jobs
```

Creates one job per repository: clone → transform → create PR.

## Commands

### setup

Setup Kubernetes backend with credentials.

```bash
bca setup --namespace <ns> [--copilot-token | --gemini-api-key | --gemini-oauth]
```

### apply

Execute code transformations.

```bash
bca apply <change-file> --namespace <ns> [--wait]
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
  image: ghcr.io/manno/background-coder:latest # optional
```

## Architecture

```
┌─────────────┐
│   Change    │ User creates YAML definition
└──────┬──────┘
       │
       v
┌─────────────┐
│  bca apply  │ Creates Kubernetes Jobs (one per repo)
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
│  ├─ bca execute (run agent on fork)       │
│  ├─ git push (push to fork)               │
│  └─ gh pr create (fork → original repo)   │
└───────────────────────────────────────────┘
```

**Init Container 1:** Creates/syncs fork in user's account using `gh repo fork`  
**Init Container 2:** Clones fork using fleet gitcloner to shared volume  
**Main Container:** Runs `bca execute --config <json>`, pushes to fork, creates cross-fork PR  
**Shared Volume:** EmptyDir at `/workspace` for passing data between containers

### Security Model

BCA uses a **staging fork approach** to limit token exposure:

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

## Job Execution Flow

1. **fork-setup init container** creates or syncs fork: `gh repo fork owner/repo`
2. **git-clone init container** clones fork with `fleet gitcloner --branch main <fork-url> /workspace/repo`
3. **Main container** receives JSON config via `$CONFIG` environment variable:
   ```json
   {
     "agent": "copilot-cli",
     "prompt": "Fix bugs",
     "agentsmd": "https://...",
     "resources": ["https://..."]
   }
   ```
4. **bca execute** downloads resources and runs agent on forked repo:
   - Copilot: `copilot --add-dir /workspace --add-dir /tmp -p "$PROMPT" --allow-all-tools`
   - Gemini: `gemini "$PROMPT"`
5. **git push** pushes changes to fork
6. **gh pr create** creates pull request from fork to original repo
7. **Auto-cleanup** after 5 minutes (TTL), max 1 retry

## Agents

| Agent | Command | Requirements |
|-------|---------|--------------|
| copilot-cli | `copilot` | GitHub token with Copilot Requests permission |
| gemini-cli | `gemini` | Gemini API key OR OAuth authentication |

Agent configuration in `internal/agent/config.go`.

## Troubleshooting

**Jobs fail with authentication errors:**
```bash
kubectl get secret bca-credentials -n <namespace> -o yaml
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

See [AGENTS.md](AGENTS.md) for detailed development guide.

## Files

- `cmd/` - CLI commands (setup, apply, execute)
- `internal/backend/` - Kubernetes job management
- `internal/agent/` - Agent executor and configuration
- `internal/change/` - Change definition parser
- `Dockerfile` - Runner image with tools (gh, fleet, gemini, copilot)
- `tests/` - Integration tests with envtest

## License

See LICENSE file.
