# Background Coding Agent (BCA)

A platform that allows engineers to execute complex code transformations across multiple repositories using natural language prompts.

## Quick Start

### 1. Build the CLI

```bash
go build -o bca .
```

### 2. Setup the Backend

First, configure your GitHub token to allow cloning repos and creating PRs:

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

Then setup the Kubernetes backend:

```bash
./bca setup --namespace bca-jobs
```

This will:
- Create the `bca-jobs` namespace in your Kubernetes cluster
- Create a secret `bca-credentials` with your GitHub token
- Configure the environment for running coding agent jobs

### 3. Apply a Change

Create a Change definition (see `example-change.yaml`):

```yaml
kind: Change
apiVersion: v1
spec:
  agentsmd: https://example.com/agents.md
  resources:
  - https://example.com/docs/guide.md
  prompt: "Add error handling to all HTTP handlers"
  repos:
  - https://github.com/example/repo1
  - https://github.com/example/repo2
  agent: gemini-cli
  image: ghcr.io/manno/background-coding-agent:latest
```

Apply the change:

```bash
./bca apply example-change.yaml --namespace bca-jobs
```

This will:
- Create one Kubernetes Job per repository
- Each job will clone the repo, download resources, run the coding agent, and create a PR
- Monitor job status and report when all jobs complete (or use `--no-wait` to return immediately)

## Commands

### setup

Set up the execution backend:

```bash
./bca setup --namespace bca-jobs [--github-token TOKEN]
```

Options:
- `--namespace` - Kubernetes namespace (default: "default")
- `--github-token` - GitHub token (defaults to GITHUB_TOKEN env var)
- `--kubeconfig` - Path to kubeconfig file

### apply

Apply a Change definition:

```bash
./bca apply change.yaml --namespace bca-jobs
```

Options:
- `--namespace` - Kubernetes namespace (default: "default")
- `--kubeconfig` - Path to kubeconfig file
- `--wait` - Wait for jobs to complete (default: true)

By default, the command monitors job status and reports when all jobs are done. Use `--no-wait` to create jobs and return immediately.

### clone

Clone a git repository (used internally by jobs):

```bash
./bca clone https://github.com/example/repo --output ./repo
```

Options:
- `--output` - Output directory (default: current directory)
- `--branch` - Git branch (default: "main")

### execute

Execute a coding agent (used internally by jobs):

```bash
./bca execute change.yaml --work-dir ./repo
```

Options:
- `--work-dir` - Working directory (default: current directory)

## Change Definition

A Change defines a code transformation task:

```yaml
kind: Change
apiVersion: v1
spec:
  agentsmd: <url>           # URL to agent instructions (markdown)
  resources:                 # Additional documentation URLs
  - <url>
  prompt: <string>          # Natural language prompt describing the task
  repos:                     # List of repository URLs to transform
  - <repo-url>
  agent: <agent-name>       # Coding agent to use (e.g., gemini-cli, copilot-cli)
  image: <image>            # Optional: container image to run (default: ghcr.io/manno/background-coding-agent:latest)
```

## Architecture

```
┌─────────────┐
│  CLI (bca)  │  - User interface
└──────┬──────┘
       │
       v
┌─────────────────────┐
│  Change Definition  │  - YAML manifest describing transformation
└─────────┬───────────┘
          │
          v
┌──────────────────────┐
│ Kubernetes Backend   │  - Creates jobs per repository
└─────────┬────────────┘
          │
          v
┌──────────────────────┐
│  Execution Runner    │  - Runs in K8s job
│  - Clone repo        │
│  - Download resources│
│  - Run coding agent  │
│  - Create PR         │
└──────────────────────┘
```

## Development

### Prerequisites

- Go 1.25+
- Docker (for building runner image)
- Kubernetes cluster (for integration tests: envtest)
- GitHub token

### Build

Local development:
```bash
go build -o bca .
```

Release binaries (static, with debug symbols):
```bash
./scripts/build-release.sh                    # Builds for linux/amd64 (default)
GOARCH=arm64 ./scripts/build-release.sh       # Builds for linux/arm64
GOOS=darwin GOARCH=arm64 ./scripts/build-release.sh  # Builds for macOS arm64
```

Binaries are output to `dist/bca-$GOOS-$GOARCH`

Docker multi-arch image:
```bash
# Step 1: Build binaries for all target architectures
./scripts/build-release.sh              # linux/amd64
GOARCH=arm64 ./scripts/build-release.sh # linux/arm64

# Step 2: Build and push multi-arch image
./scripts/build-runner-image.sh         # Uses buildx for linux/amd64,linux/arm64
```

### Test

Unit tests:
```bash
go test ./internal/...
```

Integration tests (requires KUBEBUILDER_ASSETS):
```bash
# Setup envtest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
setup-envtest use 1.34.1
export KUBEBUILDER_ASSETS=$(setup-envtest use -p path 1.34.1)

# Run tests
go install github.com/onsi/ginkgo/v2/ginkgo@latest
ginkgo -v ./tests/...
```

### Code Quality

```bash
# Format
goimports -w .

# Lint
go vet ./...
golangci-lint run --fix
```

## Documentation

- [SPEC01.md](SPEC01.md) - Original specification
- [PROGRESS.md](PROGRESS.md) - Implementation progress
- [AGENTS.md](AGENTS.md) - AI assistant guide
- [tests/README.md](tests/README.md) - Testing documentation

## License

TBD
