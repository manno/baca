# Background Coding Agent (BCA)

Execute AI-driven code transformations across multiple repositories using natural language prompts.

## ⚠️  Security Notice

**Never commit API tokens to source control.** Use environment variables or secret managers. See [AGENTS.md](AGENTS.md#security-best-practices).

## Prerequisites

- **Kubernetes cluster** (k3d, minikube, or remote)
- **kubectl** configured
- **GitHub Token** with `Contents` + `Pull requests` permissions
- **Agent credentials:**
  - Copilot: Fine-grained PAT with `Copilot Requests` permission
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
┌───────────────────────────────────┐
│  Kubernetes Job (per repository)  │
│                                   │
│  Init Container:                  │
│  └─ fleet gitcloner (clone repo)  │
│                                   │
│  Main Container:                  │
│  ├─ bca execute (run agent)       │
│  └─ gh pr create (create PR)      │
└───────────────────────────────────┘
```

**Init Container:** Clones repository using fleet gitcloner to shared volume  
**Main Container:** Runs `bca execute --config <json>` with agent-specific logic  
**Shared Volume:** EmptyDir at `/workspace` for passing repository between containers

## Job Execution Flow

1. **Init container** clones repo with `fleet gitcloner --branch main <repo> /workspace/repo`
2. **Main container** receives JSON config via `$CONFIG` environment variable:
   ```json
   {
     "agent": "copilot-cli",
     "prompt": "Fix bugs",
     "agentsmd": "https://...",
     "resources": ["https://..."]
   }
   ```
3. **bca execute** downloads resources and runs agent:
   - Copilot: `copilot --add-dir /workspace --add-dir /tmp -p "$PROMPT" --allow-all-tools`
   - Gemini: `gemini "$PROMPT"`
4. **gh pr create** creates pull request with changes
5. **Auto-cleanup** after 1 hour (TTL), max 3 retries

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
./scripts/build-release.sh           # Build CLI binaries
./scripts/build-runner-image.sh      # Build Docker image
./scripts/import-image-k3d.sh        # Import to k3d cluster
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
