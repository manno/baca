# Background Coding Agent (BCA) - AI Assistant Guide

This document provides guidance for AI coding assistants working on the Background Coding Agent project.

## ⚠️  SECURITY WARNING

**CRITICAL: Handle all API tokens and credentials with extreme care!**

The tokens used by BCA have **very sensitive permissions**:
- **GITHUB_TOKEN**: Access to repositories, ability to create PRs, read/write code
- **COPILOT_TOKEN**: Access to GitHub Copilot services
- **GEMINI_API_KEY**: Access to Google AI services

**Never:**
- ❌ Commit tokens to source control (including examples, tests, docs)
- ❌ Log tokens in application output or error messages
- ❌ Share tokens in screenshots, documentation, or chat
- ❌ Store tokens in plain text files (except encrypted secrets)
- ❌ Use production tokens for testing or development

**Always:**
- ✅ Use environment variables or secure secret management
- ✅ Store in Kubernetes secrets with encryption at rest enabled

---

## Project Overview

Background Coding Agent (BCA) is a platform that allows engineers to execute complex code transformations across multiple repositories using natural language prompts. It orchestrates coding agents (like gemini-cli or copilot-cli) through Kubernetes jobs.

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

## Project Structure

```
.
├── cmd/                    # CLI commands
│   ├── root.go            # Root command, config, logging
│   ├── setup.go           # Backend setup
│   ├── apply.go           # Apply Change definitions
│   ├── clone.go           # Git cloning (uses fleet gitcloner)
│   └── execute.go         # Execute coding agents
├── internal/              # Internal packages
│   ├── agent/             # Agent configuration
│   │   ├── config.go      # Agent registry (name → command mapping)
│   │   └── executor.go    # Download resources, run agent
│   ├── backend/           # Kubernetes backend
│   │   ├── client.go      # K8s client setup
│   │   └── kubernetes.go  # Backend implementation
│   ├── change/            # Change definition
│   │   ├── types.go       # Change struct
│   │   └── parser.go      # YAML parser & validation
│   └── git/               # Git operations
│       └── clone.go       # Clone wrapper (calls fleet)
├── tests/                 # Integration tests
│   ├── utils/             # Test utilities (envtest setup)
│   └── backend/           # Backend integration tests
│       ├── suite_test.go  # Ginkgo suite setup
│       └── backend_test.go # Test specs
├── runner-image/          # Docker runner image
│   └── Dockerfile         # Multi-arch runner image
├── scripts/               # Build and utility scripts
├── main.go                # CLI entrypoint
├── SPEC01.md             # Original specification
└── PROGRESS.md           # Implementation progress
```

## Key Technologies

- **Language**: Go 1.25+
- **CLI Framework**: Cobra + Viper
- **Logging**: slog (structured JSON logging)
- **Kubernetes**: controller-runtime client
- **Testing**: Ginkgo v2 + Gomega + envtest

## Development Principles

### YAGNI (You Ain't Gonna Need It)
Start simple, iterate. Don't add features until they're needed.

### Code Organization
- `cmd/` - CLI interface only, minimal logic
- `internal/` - All business logic
- Separate packages by domain (backend, change, git, agent)

### Code Style
- Use `goimports -w .` for formatting
- Run `go vet ./...` to catch issues
- Run `go build` to verify compilation
- Minimal comments - only for clarification
- Write integration tests for K8s interactions (see tests/README.md)

### Logging
Use structured logging with slog:
```go
logger.Info("message", "key", value)
logger.Error("failed", "error", err)
```

## Change Definition Format

```yaml
kind: Change
apiVersion: v1
spec:
  agentsmd: https://example.com/agents.md  # Agent instructions
  resources:                                # Additional documentation
  - https://example.com/docs/guide.md
  prompt: "Your transformation task"        # Natural language prompt
  repos:                                    # Target repositories
  - https://github.com/org/repo1
  - https://github.com/org/repo2
  agent: gemini-cli                         # Logical agent name
  image: ghcr.io/example/runner:latest     # Optional container image
```

### Agent Configuration

Agents use a configuration system that maps logical names to actual commands:

| Logical Name | Command | Required Credentials |
|-------------|---------|---------------------|
| `gemini-cli` | `gemini` | `GOOGLE_API_KEY` |
| `copilot-cli` | `copilot` | `GITHUB_TOKEN` |

**Configuration:** `internal/agent/config.go`

When you specify `agent: gemini-cli` in the Change definition, the job will execute the `gemini` command with the appropriate credentials injected.

## Common Tasks

### Adding a New Agent

1. Update `internal/agent/config.go`:
   ```go
   "my-agent": {
       Name:        "my-agent",
       Command:     "my-command",
       Credentials: []string{"MY_API_KEY"},
   }
   ```
2. Update setup command to handle new credentials
3. Update job creation to inject agent-specific credentials
4. Test with `agent: my-agent` in Change YAML

### Adding a New Command

1. Create `cmd/newcommand.go`
2. Define cobra.Command with Use, Short, Long, RunE
3. Add flags with `cmd.Flags().String(...)`
4. Register in `init()` with `rootCmd.AddCommand(newCmd)`
5. Implement business logic in `internal/` package

### Adding to Kubernetes Backend

1. Update `internal/backend/client.go` scheme if new K8s types needed
2. Add methods to `KubernetesBackend` in `internal/backend/`
3. Use `k.client.Create/Get/List/Update/Delete` for K8s operations
4. Always check errors and log appropriately

### Working with Change Definitions

1. Update types in `internal/change/types.go`
2. Update validation in `internal/change/parser.go`
3. YAML tags should match snake_case field names

## Testing

See [tests/README.md](tests/README.md) for detailed testing documentation.

### Quick Start

**Integration Tests with k3d** (use for inspecting actual job results):
```bash
# Use existing k3d cluster
export CI_USE_EXISTING_CLUSTER=true
ginkgo -v ./tests/...
```

**Unit Tests** (no infrastructure needed):
```bash
go test ./internal/...
```

**Manual Testing with k3d**:
```bash
go build -o bca .

# Setup namespace and credentials
./bca setup --namespace test --github-token ghp_xxx

# Apply a Change (creates jobs)
./bca apply change.yaml --namespace test --wait

# Inspect job results
kubectl get jobs -n test
kubectl logs -n test job/bca-<repo-name>
```

### Important Testing Notes

**⚠️ Docker Image Loading for k3d:**

When you update the Docker runner image, you MUST load it into k3d:

```bash
# Build new image
./scripts/build-release.sh
./scripts/build-runner-image.sh
# Import into k3d cluster (required!)
./scripts/import-runner-image.sh
```

**Test Modes:**
- **envtest (default)**: Use for running integration tests. Fast, isolated, ephemeral API server.
- **k3d cluster**: Use for inspecting results, debugging actual job execution, testing with real images.

Set `CI_USE_EXISTING_CLUSTER=true` to use k3d cluster instead of envtest.

### Test Organization

- **Unit tests**: `internal/*/` - Fast, no K8s cluster
- **Integration tests**: `tests/*/` - Use Ginkgo + envtest with real K8s API server
- **Test utilities**: `tests/utils/` - Shared helpers for envtest setup

### Writing Integration Tests

Use Ginkgo BDD style with envtest:

```go
var _ = Describe("Backend", func() {
    var namespace string

    BeforeEach(func() {
        namespace, _ = utils.NewNamespaceName()
        Expect(k8sClient.Create(ctx, &corev1.Namespace{
            ObjectMeta: metav1.ObjectMeta{Name: namespace},
        })).To(Succeed())

        DeferCleanup(func() {
            k8sClient.Delete(ctx, &corev1.Namespace{
                ObjectMeta: metav1.ObjectMeta{Name: namespace},
            })
        })
    })

    It("creates resources", func() {
        backend, err := backend.NewKubernetesBackend(cfg, namespace, logger)
        Expect(err).NotTo(HaveOccurred())
        // Test with real K8s API server
    })
})
```

## Working with This Codebase

### Making Changes
1. **Read SPEC01.md** - Understand requirements
2. **Check PROGRESS.md** - See what's done/planned
3. **Follow existing patterns** - Consistency is key
4. **Test locally** - Build and run before committing
5. **Update PROGRESS.md** - Document significant changes

### Code Review Checklist
- [ ] Follows existing code structure and patterns
- [ ] Uses structured logging appropriately
- [ ] Handles errors and logs them
- [ ] Formats with `goimports -w .`
- [ ] Passes `go vet ./...`
- [ ] Builds successfully with `go build`
- [ ] Integration tests pass: `ginkgo -v ./tests/...`
- [ ] Minimal, targeted changes (surgical approach)

## Current Status

### ✅ Completed
- CLI framework (setup, apply, clone, execute)
- Change definition parser with validation
- Kubernetes backend with job creation and monitoring
- Agent configuration system (logical name → command mapping)
- Fleet gitcloner integration for git authentication
- Job status monitoring with `--wait` flag
- Namespace and secret creation in `backend.Setup()`
- Docker runner image with multi-arch support
- Integration test infrastructure (envtest + k3d modes)
- ImagePullPolicy support for local testing
- Per-agent credential injection (starting with gemini-cli)
- Node.js version update for gemini compatibility
- Authentication system (all agents)
- Job creation and monitoring
- Secret management
- Git cloning with credentials
- Job cleanup (TTL: 1 hour, BackoffLimit: 3)
- Integration tests (7/7 passing)

See PROGRESS.md for detailed next steps and TODO items.

## Docker Image Building

```bash
# Build for current architecture
./scripts/build-runner-image.sh

# Import to k3d for testing
./scripts/import-runner-image.sh
```

### Base Image Details
- Base: `catthehacker/ubuntu:act-latest`
- Upgraded Node.js: `/usr/bin/node` (v20.19.6)

### Common Patterns

#### Creating K8s Resources
```go
obj := &corev1.Namespace{
    ObjectMeta: metav1.ObjectMeta{
        Name: "example",
    },
}
err := k.client.Create(ctx, obj)
```

#### Logging with Context
```go
logger.Info("operation starting",
    "namespace", namespace,
    "resource", resourceName)
```

#### Error Handling
```go
if err != nil {
    logger.Error("operation failed", "error", err)
    return fmt.Errorf("failed to do thing: %w", err)
}
```

## Kubernetes Job Pattern

The execution flow:
1. User runs `bca apply change.yaml` with Change definition
2. Backend creates one K8s Job per repository
3. Each Job runs with runner image containing:
   - `fleet gitcloner` to clone repository (uses GITHUB_TOKEN)
   - `curl` to download agents.md and resources
   - Agent command (`gemini` or `copilot`) to execute transformation
   - `gh` tool to create pull request
4. Jobs are monitored for completion/failure with `--wait` flag
5. Results are reported back to user

**Job Script Example:**
```bash
set -e && \
cd /workspace && \
fleet gitcloner $REPO_URL ./repo && \
cd ./repo && \
curl -L -o agents.md 'https://...' && \
gemini "$PROMPT" && \
gh pr create --fill
```

**Credentials:**
- `GITHUB_TOKEN`: From `bca-credentials` secret (for git clone and gh CLI)
- `GOOGLE_API_KEY`: To be injected for gemini-cli agent
- Agent-specific credentials: Configured in `internal/agent/config.go`

## Resources

- **SPEC01.md**: Original specification and requirements
- **PROGRESS.md**: Detailed implementation progress and recent updates
- **tests/README.md**: Comprehensive testing documentation
- **Fleet Gitcloner**: `github.com/rancher/fleet` - Git cloning with authentication
- **Controller-Runtime**: `sigs.k8s.io/controller-runtime` - K8s client library
- **Cobra**: `github.com/spf13/cobra` - CLI framework
- **Ginkgo**: Testing framework for integration tests

## Security Best Practices

### Token Management

**Storage:**
- ✅ Environment variables (local development)
- ✅ Kubernetes secrets with encryption at rest (production)
- ✅ External secret managers (Vault, AWS Secrets Manager, etc.)
- ❌ Never in source code, even in comments
- ❌ Never in container images
- ❌ Never in logs or error messages

**Usage:**
```bash
# Good: Environment variable
export GITHUB_TOKEN=$(cat ~/.secrets/github-token)
./bca setup

# Bad: Inline token
./bca setup --github-token ghp_actualtoken123  # DON'T DO THIS

# Good: From secret manager
export GITHUB_TOKEN=$(vault kv get -field=token secret/bca/github)
./bca setup
```

### Token Permissions Reference

**GITHUB_TOKEN**

Authenticate with a Personal Access Token (PAT)
The minimum required scopes for the token are: repo, read:org, and gist


**COPILOT_TOKEN:**

Visit https://github.com/settings/personal-access-tokens/new
Under "Permissions," click "add permissions" and select "Copilot Requests"

**GEMINI_API_KEY:**
```
Permissions managed at: https://aistudio.google.com/apikey
- API key provides full access to Gemini API
- Use quotas/rate limits to prevent abuse
- Monitor usage through Google Cloud Console
```

## Questions?

When uncertain:
1. Check SPEC01.md for requirements
2. Look at existing code for patterns
3. Follow YAGNI - start simple
4. Ask for clarification if needed

## Version Info

- **Go Version**: 1.25+
- **Controller-Runtime**: v0.22.4
- **Kubernetes Client**: v0.34.2
- **Cobra**: v1.10.1
