# End-to-End Testing Guide

This document provides a step-by-step guide for testing the complete BCA workflow from setup to pull request creation.

## Prerequisites

- Kubernetes cluster (k3d or production)
- Valid tokens:
  - GITHUB_TOKEN with repo and PR permissions
  - COPILOT_TOKEN (optional, for copilot-cli)
  - GEMINI_API_KEY (optional, for gemini-cli)
- Built bca binary and runner image
- Test repository you have write access to

## Setup Steps

### 1. Build the Project

```bash
# Build CLI binary
go build -o bca .

# Build and load runner image (for k3d)
./scripts/build-release.sh
./scripts/build-runner-image.sh
./scripts/import-runner-image.sh  # Only for k3d
```

### 2. Prepare Test Repository

Create or use an existing test repository:
- Must be accessible via HTTPS
- You must have write permission
- Should have a simple structure for testing

Example: `https://github.com/your-username/test-repo`

### 3. Setup Backend

```bash
# Set environment variables
export GITHUB_TOKEN="ghp_your_token_here"
export GEMINI_API_KEY="AIza_your_key_here"  # If using gemini-cli
export COPILOT_TOKEN="github_pat_your_token"  # If using copilot-cli

# Create namespace and credentials
./bca setup --namespace bca-test
```

Verify setup:
```bash
kubectl get secret -n bca-test bca-credentials
kubectl describe secret -n bca-test bca-credentials
```

### 4. Create Test Change Definition

Create `test-change.yaml`:
```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Add a simple README.md file with project description and license information"
  repos:
  - https://github.com/your-username/test-repo
  agent: gemini-cli  # or copilot-cli
```

### 5. Apply Change

```bash
# Apply and wait for completion
./bca apply test-change.yaml --namespace bca-test --wait

# Or apply without waiting
./bca apply test-change.yaml --namespace bca-test --no-wait

# Or without comiling first
go run ./main.go apply change.yaml
```

### 6. Monitor Job Execution

```bash
# List jobs
kubectl get jobs -n bca-test

# Check job status
kubectl describe job -n bca-test bca-test-repo

# View logs
kubectl logs -n bca-test -l job-name=bca-test-repo --follow

# Check pod status if job fails
kubectl get pods -n bca-test
kubectl logs -n bca-test <pod-name>
```

## Expected Workflow

1. **Job Creation**: BCA creates Kubernetes job
2. **Clone**: Job clones repository using fleet gitcloner
3. **Download**: Downloads agents.md and resources (if specified)
4. **Execute**: Runs coding agent with prompt
5. **PR Creation**: Creates pull request with changes
6. **Cleanup**: Job completes and TTL cleanup after 1 hour

## Troubleshooting

### Job Fails to Start

```bash
# Check job events
kubectl describe job -n bca-test <job-name>

# Check pod events
kubectl get pods -n bca-test
kubectl describe pod -n bca-test <pod-name>
```

Common issues:
- Image pull errors (ImagePullBackOff)
- Missing credentials (check secret)
- Insufficient permissions

### Clone Fails

Check logs for fleet gitcloner errors:
```bash
kubectl logs -n bca-test -l job-name=<job-name> | grep -A5 "fleet gitcloner"
```

Common issues:
- Invalid GITHUB_TOKEN
- Repository doesn't exist
- No read permission on repo

### Agent Execution Fails

Check logs for agent errors:
```bash
kubectl logs -n bca-test -l job-name=<job-name> | grep -A10 "gemini\|copilot"
```

Common issues:
- Invalid API key (gemini-cli)
- Invalid token (copilot-cli)
- Node.js version incompatibility
- Rate limiting

### PR Creation Fails

Check logs for gh CLI errors:
```bash
kubectl logs -n bca-test -l job-name=<job-name> | grep -A5 "gh pr create"
```

Common issues:
- No changes to commit
- Branch protection rules
- Missing PR permissions in token
- Not authenticated with gh

## Validation

After successful execution:

1. **Check Pull Request**:
   - Visit repository on GitHub
   - Look for new PR from BCA Bot
   - Review changes made by agent

2. **Verify Changes**:
   - PR should contain modifications requested in prompt
   - Changes should be reasonable and complete
   - No unintended modifications

3. **Check Job Cleanup**:
   ```bash
   # Jobs should be cleaned up after 1 hour
   kubectl get jobs -n bca-test
   ```

## Multi-Repository Testing

Test with multiple repositories:

```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Update copyright year to 2025 in all source files"
  repos:
  - https://github.com/your-username/repo1
  - https://github.com/your-username/repo2
  - https://github.com/your-username/repo3
  agent: gemini-cli
```

Expected behavior:
- One job created per repository
- Jobs run in parallel
- Each creates its own PR
- All jobs monitored until completion

## Cleanup

```bash
# Delete namespace and all resources
kubectl delete namespace bca-test

# Or just delete jobs
kubectl delete jobs -n bca-test --all
```

## Next Steps

Once end-to-end testing passes:
- Test with different agents (gemini-cli, copilot-cli)
- Test with complex prompts
- Test error handling (invalid repos, failed transformations)
- Test with private repositories
- Performance testing with many repositories
