# Feature: GitHub Actions Execution Mode

**Status:** Not part of MVP - Future Enhancement

## Overview

This feature would enable BCA to execute code transformations using GitHub Actions workflows instead of Kubernetes jobs. This provides an alternative execution backend that doesn't require a Kubernetes cluster.

## Motivation

- **Lower barrier to entry**: Users without Kubernetes access can still use BCA
- **GitHub-native**: Leverages existing GitHub infrastructure and secrets
- **Simpler setup**: No need for cluster management or credential sync
- **Cost-effective**: Uses GitHub Actions minutes instead of compute infrastructure

## Architecture Changes

### Command Structure

The existing commands would be namespaced under execution backends:

**Current:**
```bash
bca setup --github-token <token>
bca apply change.yaml
bca execute --config <json>
```

**New Structure:**
```bash
# Kubernetes backend (existing)
bca k8s setup --github-token <token>
bca k8s apply change.yaml
bca execute --config <json>  # unchanged, runs in job/workflow

# GitHub Actions backend (new)
bca gha setup --repo <org/repo>
bca gha apply change.yaml --repo <org/repo>
```

### GitHub Actions Workflow

`bca gha setup` would create a reusable workflow file in the target repository:

**.github/workflows/bca-execute.yml:**
```yaml
name: BCA Execute

on:
  workflow_dispatch:
    inputs:
      agent:
        description: 'Agent to use (copilot-cli or gemini-cli)'
        required: true
        type: string
      prompt:
        description: 'Transformation prompt'
        required: true
        type: string
      branch:
        description: 'Base branch'
        required: false
        default: 'main'
        type: string
      agentsmd:
        description: 'URL to agents.md file'
        required: false
        type: string
      resources:
        description: 'Comma-separated resource URLs'
        required: false
        type: string

jobs:
  transform:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout repository
      uses: actions/checkout@v4
      with:
        ref: ${{ inputs.branch }}
        token: ${{ secrets.GITHUB_TOKEN }}

    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '20'

    - name: Install BCA and agents
      run: |
        # Install BCA CLI
        curl -L https://github.com/manno/background-coding-agent/releases/latest/download/bca-linux-amd64 -o /usr/local/bin/bca
        chmod +x /usr/local/bin/bca

        # Install agents based on input
        if [ "${{ inputs.agent }}" == "copilot-cli" ]; then
          npm install -g @github/copilot
        elif [ "${{ inputs.agent }}" == "gemini-cli" ]; then
          npm install -g @google/gemini-cli
        fi

    - name: Create config JSON
      id: config
      run: |
        CONFIG=$(jq -n \
          --arg agent "${{ inputs.agent }}" \
          --arg prompt "${{ inputs.prompt }}" \
          --arg agentsmd "${{ inputs.agentsmd }}" \
          --arg resources "${{ inputs.resources }}" \
          '{
            agent: $agent,
            prompt: $prompt,
            agentsmd: ($agentsmd | if . == "" then null else . end),
            resources: ($resources | if . == "" then [] else split(",") end)
          }')
        echo "json=$CONFIG" >> $GITHUB_OUTPUT

    - name: Execute transformation
      env:
        CONFIG: ${{ steps.config.outputs.json }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        COPILOT_TOKEN: ${{ secrets.COPILOT_TOKEN }}
        GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
      run: |
        git config --global user.name "BCA Bot"
        git config --global user.email "bca-bot@users.noreply.github.com"
        bca execute --config "$CONFIG" --work-dir .

    - name: Create Pull Request
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: |
        gh pr create --fill || echo "PR creation failed or no changes"
```

## Command Details

### `bca gha setup`

**Purpose:** Install the BCA workflow file in a GitHub repository

**Usage:**
```bash
bca gha setup --repo owner/repo [--workflow-path .github/workflows/bca-execute.yml]
```

**Implementation:**
1. Authenticate with GitHub (using GITHUB_TOKEN)
2. Check if repository exists and user has write access
3. Create/update `.github/workflows/bca-execute.yml` via GitHub API
4. Verify required secrets exist (GITHUB_TOKEN, COPILOT_TOKEN or GEMINI_API_KEY)
5. Print setup instructions for missing secrets

**Required Secrets:**
- `GITHUB_TOKEN`: Automatically available in GitHub Actions
- `COPILOT_TOKEN`: User must add to repository secrets
- `GEMINI_API_KEY`: User must add to repository secrets

Alternatively, this command would just create the workflow file and inform the user to add the necessary secrets via GitHub UI.

### `bca gha apply`

**Purpose:** Trigger workflow runs for each repository in Change definition

**Usage:**
```bash
bca gha apply change.yaml --repo owner/repo [--wait]
```

**Implementation:**
1. Parse Change YAML
2. For each target repository:
   - Extract owner/repo from URL
   - Call GitHub API to trigger workflow_dispatch
   - Pass Change fields as workflow inputs
3. If `--wait`: Poll workflow run status and print logs
4. Report summary of triggered workflows

**GitHub API Call:**
```
POST /repos/{owner}/{repo}/actions/workflows/bca-execute.yml/dispatches
{
  "ref": "main",
  "inputs": {
    "agent": "copilot-cli",
    "prompt": "Fix all typos",
    "branch": "main",
    "agentsmd": "https://...",
    "resources": "https://...,https://..."
  }
}
```

## Comparison with Kubernetes Backend

| Feature | Kubernetes Backend | GitHub Actions Backend |
|---------|-------------------|------------------------|
| **Setup Complexity** | High (cluster required) | Low (GitHub only) |
| **Credentials** | K8s Secret | Repository Secrets |
| **Parallelism** | Job per repo | Workflow run per repo |
| **Monitoring** | `kubectl` or `--wait` | GitHub UI or `--wait` |
| **Cost** | Cluster compute | GitHub Actions minutes |
| **Access Control** | K8s RBAC | GitHub permissions |
| **Logs** | `kubectl logs` | GitHub Actions logs |

## Implementation Phases

### Phase 1: Command Structure Refactoring
- Rename existing commands to `k8s` namespace
- Create backend interface abstraction
- Keep existing functionality unchanged

### Phase 2: GitHub Actions Backend
- Implement `gha setup` command
- Generate workflow file via GitHub API
- Add secret validation

### Phase 3: Workflow Dispatch
- Implement `gha apply` command
- Trigger workflow runs via API
- Add `--wait` support with log streaming

### Phase 4: Enhancements
- Support for custom workflow templates
- Parallel workflow triggers with rate limiting
- Better error handling for API failures

## User Experience

### Setup Flow

```bash
# One-time setup: Install workflow in a repository
$ bca gha setup --repo myorg/myrepo
✓ Workflow file created: .github/workflows/bca-execute.yml
⚠ Required secrets not configured:
  - COPILOT_TOKEN: Add at https://github.com/myorg/myrepo/settings/secrets/actions
  - GEMINI_API_KEY: Add at https://github.com/myorg/myrepo/settings/secrets/actions

# Add secrets via GitHub UI
# Then apply changes
$ bca gha apply change.yaml --repo myorg/myrepo
✓ Triggered workflow run: https://github.com/myorg/myrepo/actions/runs/123456
✓ Triggered workflow run: https://github.com/myorg/another-repo/actions/runs/123457

# Monitor progress
$ bca gha apply change.yaml --repo myorg/myrepo --wait
⏳ Waiting for workflow runs to complete...
✓ myorg/myrepo: Success - PR created #42
✓ myorg/another-repo: Success - PR created #15
```

## Technical Considerations

### Workflow File Management
- **Version control**: Workflow file is committed to repository
- **Updates**: `bca gha setup` can update existing workflow
- **Customization**: Users can modify workflow after creation

### Secret Management
- Secrets must be added via GitHub UI or API
- No way to bulk-add secrets across repositories
- Consider GitHub App for easier credential management

### Rate Limiting
- GitHub API has rate limits for workflow dispatches
- Implement exponential backoff for retries
- Consider batch processing for many repositories

### Security
- Workflow runs with repository secrets
- Need write permissions to create workflow file
- Consider using GitHub App tokens for better security

## Future Enhancements

1. **GitHub App Integration**: Use GitHub App for authentication and secrets
2. **Matrix Strategy**: Run transformations for multiple repos in single workflow
3. **Artifact Storage**: Save transformation logs as workflow artifacts
4. **Status Checks**: Integration with PR status checks
5. **Self-hosted Runners**: Support for running on self-hosted runners

## Migration Path

Existing Kubernetes users can migrate gradually:

```bash
# Continue using K8s
bca k8s apply change.yaml

# Or switch to GHA
bca gha setup --repo org/repo
bca gha apply change.yaml --repo org/repo
```

Both backends use the same Change YAML format and `execute` command.

## Open Questions

1. **Multi-repo workflows**: Should we support triggering one workflow in a management repo that processes multiple repos, or keep one customized workflow per repo?

## Related Work

- GitHub Actions workflow_dispatch API
- GitHub CLI (`gh`) for workflow triggers
- GitHub Apps for better authentication
