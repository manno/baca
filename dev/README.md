# BCA Development Guide

Documentation for developers.

## ⚠️  Security

**CRITICAL: Never commit tokens to source control!**

Tokens have sensitive permissions (repo read/write, PR creation, Copilot/Gemini access).

**Don't:**
- ❌ Commit tokens (even in examples/tests/docs)
- ❌ Log tokens in output
- ❌ Use production tokens for testing

**Do:**
- ✅ Use environment variables
- ✅ Store in Kubernetes secrets
- ✅ Rotate tokens regularly

## Architecture

```
CLI → Change YAML → Kubernetes Jobs → Init Container (clone) + Main Container (execute + PR)
```

**Init Container:** `fleet gitcloner` clones repo to `/workspace/repo`
**Main Container:** `bca execute --config <json>` runs agent, then `gh pr create`
**Shared Volume:** EmptyDir at `/workspace` passes repo between containers

## Project Structure

```
cmd/              - CLI commands (setup, apply, execute)
internal/
  agent/          - Agent executor and config (gemini-cli, copilot-cli)
  backend/        - Kubernetes job management
  change/         - Change definition parser
Dockerfile        - Runner image (gh, fleet, gemini, copilot, node v20)
tests/            - Integration tests (Ginkgo + envtest)
dev/              - Build scripts
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/execute.go` | Accepts `--config` JSON, runs agent |
| `internal/agent/executor.go` | Agent-specific execution logic |
| `internal/agent/config.go` | Agent name → command mapping |
| `internal/backend/apply.go` | Creates K8s jobs with init containers |
| `Dockerfile` | Runner image with tools |

## Development Workflow

### Build

```bash
go build -o bca .                    # Build CLI
./dev/build-release.sh           # Multi-arch binaries
./dev/build-runner-image.sh      # Docker image
./dev/import-image-k3d.sh        # Load into k3d
```

### Test

```bash
go test ./internal/...               # Unit tests
ginkgo -v ./tests/...                # Integration tests (uses envtest)
```

**Integration test modes:**
- Default: envtest (fast, ephemeral API server)
- `CI_USE_EXISTING_CLUSTER=true`: Use k3d cluster (for debugging)

### Code Style

- Run `goimports -w .` for formatting
- Run `go vet ./...` to catch issues
- Minimal comments (only for clarification)
- Use structured logging: `logger.Info("msg", "key", value)`

## Change Definition

```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Natural language task"
  repos: ["https://github.com/org/repo"]
  agent: copilot-cli  # or gemini-cli
  branch: main        # optional, default: main
  agentsmd: "https://..." # optional
  resources: ["https://..."] # optional
  image: ghcr.io/manno/background-coder:latest # optional
```

## Agent Configuration

**`internal/agent/config.go`:**

```go
"gemini-cli": {
    Name: "gemini-cli",
    Command: "gemini",
    Credentials: []string{"GEMINI_API_KEY"},
}
"copilot-cli": {
    Name: "copilot-cli",
    Command: "copilot",
    Credentials: []string{"COPILOT_TOKEN"},
}
```

**`internal/agent/executor.go`:**

- **Copilot:** `copilot --add-dir /workspace --add-dir /tmp -p "$PROMPT" --allow-all-tools`
- **Gemini:** `gemini "$PROMPT"`

## Job Structure

```yaml
Job:
  initContainers:
  - name: git-clone
    image: ghcr.io/manno/background-coder:latest
    command: fleet gitcloner --branch main $REPO_URL /workspace/repo
    volumeMounts:
    - name: workspace
      mountPath: /workspace
    envFrom:
    - secretRef: bca-credentials

  containers:
  - name: runner
    image: ghcr.io/manno/background-coder:latest
    command: |
      cd /workspace/repo
      git config --global user.name "BCA Bot"
      bca execute --config "$CONFIG" --work-dir /workspace/repo
      gh pr create --fill
    env:
    - name: CONFIG
      value: '{"agent":"copilot-cli","prompt":"...","agentsmd":"...","resources":[...]}'
    volumeMounts:
    - name: workspace
      mountPath: /workspace
    envFrom:
    - secretRef: bca-credentials

  volumes:
  - name: workspace
    emptyDir: {}
```

**$CONFIG JSON:** Protects special characters in prompts/resources.

## Adding a New Agent

1. **Update `internal/agent/config.go`:**
   ```go
   "my-agent": {
       Name: "my-agent",
       Command: "my-command",
       Credentials: []string{"MY_API_KEY"},
   }
   ```

2. **Add to `internal/agent/executor.go`:**
   ```go
   case "my-agent":
       cmd = exec.CommandContext(ctx, agentCommand, args...)
   ```

3. **Update setup** to handle credentials

4. **Test** with `agent: my-agent` in Change YAML

## Testing

### Unit Tests

```bash
go test ./internal/backend/
```

Tests for job name generation, uniqueness, etc.

### Integration Tests

```bash
# Fast (envtest)
ginkgo -v ./tests/backend/

# With k3d cluster (for debugging)
export CI_USE_EXISTING_CLUSTER=true
ginkgo -v ./tests/backend/

# Inspect jobs
kubectl get jobs -n test
kubectl logs -n test job/bca-xxx
```

**Test Structure:**
```go
var _ = Describe("Backend", func() {
    var namespace string

    BeforeEach(func() {
        namespace, _ = utils.NewNamespaceName()
        // Create namespace
        DeferCleanup(func() {
            // Delete namespace
        })
    })

    It("creates resources", func() {
        // Test with real K8s API server
    })
})
```

## Docker Image

**Base:** `catthehacker/ubuntu:act-latest`
**Includes:**
- Node.js v20 (for gemini-cli, copilot)
- `gh` CLI v2.63.1
- `fleet` v0.14.0
- `gemini` (npm package)
- `copilot` (npm package)
- `bca` binary

**Build:** Multi-arch (amd64, arm64)

## Common Tasks

### Update Agent Logic

Edit `internal/agent/executor.go` → Rebuild → Test

### Change Job Structure

Edit `internal/backend/apply.go` → Rebuild → Test integration

### Add New CLI Command

1. Create `cmd/newcommand.go`
2. Define `cobra.Command`
3. Register in `init()`: `rootCmd.AddCommand(newCmd)`
4. Implement logic in `internal/` package

## Troubleshooting

**Build fails:**
```bash
go mod tidy
go build .
```

**Tests fail:**
```bash
# Check envtest
./dev/setup-envtest.sh

# Use k3d for debugging
export CI_USE_EXISTING_CLUSTER=true
kubectl get pods -n test --watch
```

**Image not found:**
```bash
./dev/build-runner-image.sh
./dev/import-image-k3d.sh
```

## YAGNI Principle

Start simple, iterate. Don't add features until needed.

- Separate `cmd/` (CLI) from `internal/` (logic)
- Minimal comments
- Write integration tests for K8s interactions
- Use structured logging

## Resources

- **SPEC01.md**: Original specification
- **README.md**: User documentation
- **tests/README.md**: Testing guide
- **Fleet**: `github.com/rancher/fleet` (git cloning)
- **Controller-Runtime**: K8s client library
- **Cobra**: CLI framework
- **Ginkgo**: BDD testing framework
