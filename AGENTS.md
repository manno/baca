# Background Automated Coding Agent (BACA) - AI Assistant Guide

This document provides comprehensive guidance for AI coding assistants working on the Background Automated Coding Agent project.

## ⚠️  SECURITY WARNING

**CRITICAL: Handle all API tokens and credentials with extreme care!**

The tokens used by BACA have **very sensitive permissions**:
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

Background Automated Coding Agent (BACA) is a declarative, prompt-driven code transformation platform that allows engineers to execute complex code transformations across multiple repositories using natural language prompts. It orchestrates coding agents (gemini-cli or copilot-cli) through Kubernetes jobs.

**Key Concept:** User creates a Change YAML → BACA creates K8s Jobs (one per repo) → Each job clones repo, runs AI agent, creates PR.

## Current Architecture (December 2025)

### Job Structure

```yaml
Pod:
  initContainers:
  - name: fork-setup
    image: ghcr.io/manno/baca-runner:latest
    command: gh repo fork $ORIGINAL_REPO_URL
    volumeMounts:
    - name: workspace
      mountPath: /workspace
    envFrom:
    - secretRef: baca-credentials

  - name: git-clone
    image: ghcr.io/manno/baca-runner:latest
    command: fleet gitcloner --branch main $FORK_URL /workspace/repo
    volumeMounts:
    - name: workspace
      mountPath: /workspace
    envFrom:
    - secretRef: baca-credentials

  containers:
  - name: runner
    image: ghcr.io/manno/baca-runner:latest
    command: |
      cd /workspace/repo
      git config --global user.name "BACA Bot"
      git remote add upstream $ORIGINAL_REPO_URL
      baca execute --config "$CONFIG" --work-dir /workspace/repo
      git push origin $BRANCH_NAME
      gh pr create --repo $ORIGINAL_REPO --head $FORK_OWNER:$BRANCH_NAME
    env:
    - name: CONFIG
      value: '{"agent":"copilot-cli","prompt":"...","agentsmd":"...","resources":[...]}'
    - name: ORIGINAL_REPO_URL
      value: https://github.com/org/repo
    volumeMounts:
    - name: workspace
      mountPath: /workspace
    envFrom:
    - secretRef: baca-credentials

  volumes:
  - name: workspace
    emptyDir: {}
```

**Why Init Containers?**
- **fork-setup**: Creates/syncs fork in authenticated user's account
- **git-clone**: Clones fork (not original repo) for isolation
- Init containers share /workspace volume with main container
- Standard Kubernetes pattern for setup tasks

**Why JSON Config?**
- Protects special characters (quotes, $, newlines) in prompts and resources
- Single environment variable instead of multiple
- No shell escaping issues
- Easy to serialize/deserialize

**Security Model (Staging Fork Approach):**
- Changes pushed to user's fork, not directly to target repos
- Cross-fork PRs created from `user-fork:branch` → `original-repo:main`
- Limits blast radius of compromised prompts
- Incremental security improvement (not a complete solution for shared usage)

### Execution Flow

1. **User** creates Change YAML with target repos (original repos, not forks)
2. **baca apply** creates one Kubernetes Job per repository (optionally with `--fork-org`)
3. **fork-setup init container** runs `gh repo fork <target-repo>` to create/sync fork
   - Uses `--fork-org` if provided, otherwise authenticated user's account
   - If fork exists, verifies it's actually a fork (fails with error if repo exists but is NOT a fork)
   - If fork doesn't exist, creates fork from target repo
4. **git-clone init container** runs `fleet gitcloner --branch main <fork-url> /workspace/repo`
5. **Main Container** receives JSON config via `$CONFIG` environment variable
6. **baca execute** parses JSON, downloads resources (agentsmd, resources), runs agent
7. **Agent** (copilot or gemini) executes transformation on fork
8. **git push** pushes changes to fork
9. **gh pr create** creates cross-fork pull request from fork to target repo
10. **Job cleanup** happens automatically after 5 minutes (TTL)

### Key Design Decisions

- **Staging fork**: Security isolation - changes pushed to fork, not target repo
- **Fork org override**: `--fork-org` flag allows specifying where to create forks
- **No ConfigMap**: Pass config as JSON env var (simpler, no extra resource)
- **fleet gitcloner**: Handles git auth automatically from GITHUB_TOKEN
- **Single JSON**: Avoids shell escaping nightmare with multiple env vars
- **Execute command**: Knows how to run each agent (copilot vs gemini)
- **Two init containers**: Fork setup, then clone fork

## Project Structure

```
.
├── cmd/                    # CLI commands
│   ├── root.go            # Root command, config, logging setup
│   ├── setup.go           # Backend setup (namespace, secrets)
│   ├── apply.go           # Apply Change definitions (create jobs)
│   └── execute.go         # Execute coding agents (runs in job)
├── internal/              # Internal packages
│   ├── agent/             # Agent configuration and execution
│   │   ├── config.go      # Agent registry (name → command mapping)
│   │   └── executor.go    # Downloads resources, runs agent
│   ├── backend/           # Backend implementations
│   │   └── k8s/           # Kubernetes backend
│   │       ├── client.go      # K8s client setup
│   │       ├── kubernetes.go  # Backend implementation
│   │       ├── apply.go       # Job creation logic
│   │       ├── setup.go       # Backend setup
│   │       └── scripts/       # Embedded bash scripts
│   │           ├── fork-setup.sh   # Fork creation/sync
│   │           └── job-runner.sh   # Agent execution & PR
│   └── change/            # Change definition
│       ├── types.go       # Change struct
│       └── parser.go      # YAML parser & validation
├── tests/                 # Integration tests
│   ├── utils/             # Test utilities (envtest setup)
│   └── backend/           # Backend integration tests
│       ├── suite_test.go  # Ginkgo suite setup
│       ├── setup_test.go  # Setup command tests
│       └── apply_test.go  # Apply command tests
├── Dockerfile             # Multi-arch runner image
├── dev/                   # Build and utility scripts
├── main.go                # CLI entrypoint
├── SPEC01.md             # Original specification
├── README.md             # User documentation
└── STATUS_SUMMARY.md     # Current status and progress
```

## Key Technologies

- **Language**: Go 1.25+
- **CLI Framework**: Cobra + Viper
- **Logging**: slog (structured JSON logging)
- **Kubernetes**: controller-runtime client
- **Testing**: Ginkgo v2 + Gomega + envtest
- **Docker**: Multi-arch builds (amd64, arm64)

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
- Run `dev/build.sh` to verify compilation after code changes
- Minimal comments - only for clarification
- Write integration tests for K8s interactions (see tests/README.md)
- **When testing changes locally**: Use `go run ./main.go apply change.yaml` to avoid stale binary issues

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
  prompt: "Your transformation task"        # REQUIRED: Natural language prompt
  repos:                                    # REQUIRED: Target repositories
  - https://github.com/org/repo1
  - https://github.com/org/repo2
  agent: copilot-cli                        # REQUIRED: gemini-cli or copilot-cli
  branch: main                              # OPTIONAL: default is "main"
  agentsmd: https://example.com/agents.md   # OPTIONAL: Agent instructions
  resources:                                # OPTIONAL: Additional documentation
  - https://example.com/docs/guide.md
  image: ghcr.io/manno/baca-runner:latest  # OPTIONAL: Custom runner image
```

**Field Details:**
- `prompt`: Natural language description - can contain quotes, special chars (protected by JSON)
- `repos`: List of full GitHub URLs to **target repositories** (BACA auto-forks to your account)
- `agent`: Logical agent name (maps to command via `internal/agent/config.go`)
- `branch`: Git branch to check out (defaults to "main")
- `agentsmd`: URL downloaded to `agents.md` in repo root
- `resources`: URLs downloaded to `resource-0.md`, `resource-1.md`, etc.
- `image`: Override default runner image

### Agent Configuration

**Location:** `internal/agent/config.go`

Maps logical agent names to actual commands and required credentials:

```go
var agentConfig = map[string]AgentConfig{
    "gemini-cli": {
        Name:        "gemini-cli",
        Command:     "gemini",
        Credentials: []string{"GEMINI_API_KEY"},
    },
    "copilot-cli": {
        Name:        "copilot-cli",
        Command:     "copilot",
        Credentials: []string{"COPILOT_TOKEN"},
    },
}
```

**Agent Execution:** `internal/agent/executor.go`

```go
// Copilot: Combined command with all args
copilot --add-dir /workspace --add-dir /tmp -p "$PROMPT" --allow-all-tools

// Gemini: Simple command
gemini "$PROMPT"
```

**Why separate config?**
- Agents may have different invocation patterns
- Credentials can be different per agent
- Easy to add new agents without touching job creation code

## Common Tasks

### Adding a New Agent

1. **Update `internal/agent/config.go`:**
   ```go
   "my-agent": {
       Name:        "my-agent",
       Command:     "my-command",
       Credentials: []string{"MY_API_KEY"},
   }
   ```

2. **Update `internal/agent/executor.go`:**
   ```go
   case "my-agent":
       cmd = exec.CommandContext(ctx, agentCommand, c.Spec.Prompt)
   ```

3. **Update `cmd/setup.go`** to handle new credentials if needed

4. **Test** with `agent: my-agent` in Change YAML

### Adding a New CLI Command

1. Create `cmd/newcommand.go`
2. Define `cobra.Command` with Use, Short, Long, RunE
3. Add flags with `cmd.Flags().String(...)`
4. Register in `init()` with `rootCmd.AddCommand(newCmd)`
5. Implement business logic in `internal/` package

### Modifying Job Structure

**File:** `internal/backend/apply.go`

Job structure is created in `createJob()` function. Key areas:

- **Init container**: Clone logic
- **Main container**: Execute command
- **Volumes**: Shared workspace, secrets, etc.
- **Environment**: JSON config, credentials
- **Labels**: For querying and management
- **TTL/Backoff**: Cleanup and retry logic

**Important:** When adding volumes, use `append()` not `=` to preserve existing volumes (e.g., gemini OAuth).

### Working with Change Definitions

1. Update types in `internal/change/types.go`
2. Update validation in `internal/change/parser.go`
3. YAML tags should match snake_case field names
4. Add JSON tags for execute command serialization

## Testing

**See `tests/README.md` for detailed testing documentation.**

### Quick Start

**Integration Tests with envtest** (default, fast):
```bash
ginkgo -v ./tests/...
```

**Integration Tests with k3d** (for inspecting actual jobs):
```bash
export CI_USE_EXISTING_CLUSTER=true
ginkgo -v ./tests/...
```

**Unit Tests**:
```bash
go test ./internal/...
```

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

### Important Testing Notes

**⚠️ Docker Image Loading for k3d:**

When you update the Docker runner image, you MUST load it into k3d:

```bash
# Build new image
./dev/build-runner-image.sh

# Import into k3d cluster (required!)
./dev/import-image-k3d.sh
```

**Test Modes:**
- **envtest (default)**: Fast, isolated, ephemeral API server. Use for running integration tests.
- **k3d cluster**: Use for inspecting results, debugging actual job execution, testing with real images.

Set `CI_USE_EXISTING_CLUSTER=true` to use k3d cluster instead of envtest.

## Docker Runner Image

**Base Image:** `catthehacker/ubuntu:act-latest`
**Location:** `Dockerfile`
**Registry:** `ghcr.io/manno/baca-runner:latest`

**Includes:**
- Node.js v20.19.6 (upgraded from v18 for gemini/copilot)
- `gh` CLI v2.63.1 (GitHub CLI for PR creation)
- `fleet` v0.14.0 (Fleet gitcloner for repo cloning)
- `@google/gemini-cli` (npm package)
- `@github/copilot` (npm package)
- `baca` binary (copied during build)

**Build Process:**
```bash
./dev/build-release.sh           # Build CLI binaries (amd64 + arm64)
./dev/build-runner-image.sh      # Build multi-arch Docker image
./dev/import-image-k3d.sh        # Import to k3d for local testing
```

**Multi-arch:** Builds for both linux/amd64 and linux/arm64 using buildx.

**Important:** ARG TARGETARCH must be declared before RUN commands that use it (not ONBUILD).

## Kubernetes Backend

### Client Setup

**File:** `internal/backend/client.go`

Creates controller-runtime client with custom scheme including Batch/v1 for Jobs.

### Backend Implementation

**File:** `internal/backend/kubernetes.go`

`KubernetesBackend` struct holds client, namespace, and logger.

**Default Image:** `ghcr.io/manno/baca-runner:latest`

### Apply Logic

**File:** `internal/backend/apply.go`

**Key Functions:**
- `ApplyChange()`: Main entry point, creates jobs for each repo
- `createJob()`: Builds Job spec with init container and main container
- `generateJobName()`: Creates unique job names with random suffix
- `monitorJobs()`: Watches job status when --wait flag is used

**Job Naming:**
- Format: `baca-{sanitized-repo}-{random-8-chars}`
- Random suffix prevents conflicts on retry
- Sanitized repo name from URL (max 63 chars K8s limit)

**Job Configuration:**
- TTLSecondsAfterFinished: 300 (5 minutes cleanup)
- BackoffLimit: 0 (no retries by default, configurable)
- RestartPolicy: Never
- ImagePullPolicy: IfNotPresent (for local testing)

## Credentials Management

### Setup Command

**File:** `cmd/setup.go`

Creates namespace and `baca-credentials` secret with tokens:
- `GITHUB_TOKEN`: For git clone and PR creation
- `COPILOT_TOKEN`: For Copilot CLI (optional, falls back to GITHUB_TOKEN)
- `GEMINI_API_KEY`: For Gemini CLI (optional)
- `GEMINI_*`: OAuth files if using `--gemini-oauth` (optional)

**Secret Structure:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: baca-credentials
stringData:
  GITHUB_TOKEN: "ghp_..."
  COPILOT_TOKEN: "github_pat_..."  # optional
  GEMINI_API_KEY: "AIza..."        # optional
  # OAuth files as base64 if --gemini-oauth used
```

### In Jobs

Credentials are injected via `envFrom`:
```yaml
envFrom:
- secretRef:
    name: baca-credentials
```

## Execute Command

**File:** `cmd/execute.go`

Accepts `--config` flag with JSON string containing:
```json
{
  "agent": "copilot-cli",
  "prompt": "Fix bugs with \"quotes\" and $VARS",
  "agentsmd": "https://example.com/agents.md",
  "resources": ["https://example.com/doc.md"]
}
```

Parses JSON into `change.ChangeSpec`, creates `change.Change`, passes to executor.

**Why JSON?**
- Shell escaping is nightmare with multiple env vars containing special chars
- Single env var = simpler
- Easy to debug (just echo $CONFIG)
- No risk of quote/dollar/backslash issues

## Error Handling Patterns

```go
// Log and return wrapped error
if err != nil {
    logger.Error("operation failed", "error", err)
    return fmt.Errorf("failed to do thing: %w", err)
}

// Create resources
if err := k.client.Create(ctx, obj); err != nil {
    logger.Error("failed to create resource", "kind", obj.GetObjectKind(), "error", err)
    return fmt.Errorf("failed to create kubernetes resource: %w", err)
}
```

## Common Patterns

### Creating K8s Resources
```go
obj := &corev1.Namespace{
    ObjectMeta: metav1.ObjectMeta{
        Name: "example",
    },
}
err := k.client.Create(ctx, obj)
```

### Logging with Context
```go
logger.Info("operation starting",
    "namespace", namespace,
    "resource", resourceName)
```

### Reading Files
```go
data, err := os.ReadFile(path)
if err != nil {
    return fmt.Errorf("failed to read file: %w", err)
}
```

### HTTP Downloads
```go
req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := http.DefaultClient.Do(req)
defer resp.Body.Close()
io.Copy(out, resp.Body)
```

## Troubleshooting

### Build Issues

```bash
go mod tidy          # Fix module issues
go build .           # Try building
go vet ./...         # Check for issues
goimports -w .       # Fix formatting
```

### Test Issues

```bash
# Check envtest
./dev/setup-envtest.sh

# Use k3d for debugging
export CI_USE_EXISTING_CLUSTER=true
kubectl get pods -n test --watch
kubectl logs -n test <pod-name> -c git-clone
kubectl logs -n test <pod-name> -c runner
```

### Image Issues

```bash
# Rebuild and import
./dev/build-runner-image.sh
./dev/import-image-k3d.sh

# Check image in k3d
docker exec k3d-upstream-server-0 crictl images | grep baca-runner
```

### Job Debugging

```bash
# List jobs
kubectl get jobs -n <namespace>

# Describe job
kubectl describe job <job-name> -n <namespace>

# Get pod
kubectl get pods -n <namespace> -l job-name=<job-name>

# Logs from init container
kubectl logs -n <namespace> <pod-name> -c git-clone

# Logs from main container
kubectl logs -n <namespace> <pod-name> -c runner

# Check config
kubectl get job <job-name> -n <namespace> -o yaml | grep CONFIG -A20
```

## Making Changes

### Workflow
1. **Read SPEC01.md** - Understand requirements
2. **Check STATUS_SUMMARY.md** - See what's done/planned
3. **Follow existing patterns** - Consistency is key
4. **Test locally** - Build and run before committing
5. **Update STATUS_SUMMARY.md** - Document significant changes

### Code Review Checklist
- [ ] Follows existing code structure and patterns
- [ ] Uses structured logging appropriately
- [ ] Handles errors and logs them
- [ ] Formats with `goimports -w .`
- [ ] Passes `go vet ./...`
- [ ] Builds successfully with `go build`
- [ ] Integration tests pass: `ginkgo -v ./tests/...`
- [ ] Minimal, targeted changes (surgical approach)

## Current Status (December 2025)

### ✅ Completed
- CLI framework (setup, apply, execute)
- Change definition parser with validation
- Kubernetes backend with job creation and monitoring
- Agent configuration system (logical name → command mapping)
- Fleet gitcloner integration for git authentication
- Job status monitoring with `--wait` flag
- Namespace and secret creation in `backend.Setup()`
- Docker runner image with multi-arch support
- Integration test infrastructure (envtest + k3d modes)
- ImagePullPolicy support for local testing
- Per-agent credential injection
- Authentication system (all agents)
- Job creation and monitoring
- Secret management
- Git cloning with credentials (init container)
- **Job cleanup** (TTL: 5 minutes, BackoffLimit: configurable, default 0)
- Integration tests (8/8 passing)
- JSON config for execute command
- Init container architecture
- Branch support (default: main)
- Copilot single command invocation
- GitHub CLI in Docker image
- Volume mount fixes for gemini OAuth

### Current Architecture Highlights

- **Init container** pattern for git cloning
- **JSON config** to protect special characters
- **Execute command** knows how to run each agent
- **Shared workspace** via EmptyDir volume
- **Multi-arch** Docker builds (amd64, arm64)
- **8/8 tests passing** (unit + integration)

## Resources

- **SPEC01.md**: Original specification and requirements
- **README.md**: User documentation
- **STATUS_SUMMARY.md**: Current status and progress
- **tests/README.md**: Comprehensive testing documentation
- **Fleet Gitcloner**: `github.com/rancher/fleet` - Git cloning with authentication
- **Controller-Runtime**: `sigs.k8s.io/controller-runtime` - K8s client library
- **Cobra**: `github.com/spf13/cobra` - CLI framework
- **Ginkgo**: Testing framework for integration tests

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
- **Node.js (in Docker)**: v20.19.6
- **gh CLI**: v2.63.1
- **fleet**: v0.14.0
