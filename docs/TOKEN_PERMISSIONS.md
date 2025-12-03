# GitHub Token Permissions for BCA

## GITHUB_TOKEN (Required)

Used for: Git operations and Pull Request creation

### Fine-Grained Personal Access Token (Recommended)

**Generate at:** https://github.com/settings/personal-access-tokens/new

**Required Permissions:**
1. **Contents**: Read and write
   - Allows: Clone repositories, push branches
   - Used by: `fleet gitcloner`, `git push`

2. **Pull requests**: Read and write
   - Allows: Create pull requests, read PR details
   - Used by: `gh pr create`

3. **Metadata**: Read (automatically included)
   - Allows: Access basic repository metadata
   - Always included with fine-grained tokens

**Repository Access:**
- Select specific repositories you want to transform
- Or grant access to all repositories in your organization

### Classic Personal Access Token

**Generate at:** https://github.com/settings/tokens/new

**Required Scopes:**
- `repo` - Full control of private repositories
- `read:org` - Read org and team membership, read org projects

**Note:** Classic tokens have broader access and cannot be scoped to specific repositories.

## COPILOT_TOKEN (Optional - if using copilot-cli agent)

Used for: GitHub Copilot CLI authentication

**Authentication:** Copilot reads tokens from `GH_TOKEN` or `GITHUB_TOKEN` environment variables (in order of precedence).

### Fine-Grained Personal Access Token (Recommended)

**Generate at:** https://github.com/settings/personal-access-tokens/new

**Required Permissions:**
1. **Copilot Requests**: Read and write
   - Allows: Access GitHub Copilot API
   - Used by: `copilot` CLI

**Instructions from GitHub Copilot:**
1. Visit https://github.com/settings/personal-access-tokens/new
2. Under "Permissions," click "add permissions" and select "Copilot Requests"
3. Generate your token
4. Add the token to your environment via `GH_TOKEN` or `GITHUB_TOKEN`

### Classic Personal Access Token

**Generate at:** https://github.com/settings/tokens/new

**Required Scopes:**
- `repo` - Full control of private repositories
- `read:org` - Read org and team membership, read org projects

**Note:** Classic tokens work for both gh CLI and Copilot CLI with the same scopes.

## Token Usage in BCA

### How Tokens Are Used

The job script handles token routing:

```bash
# Save original GITHUB_TOKEN for gh CLI
SAVED_GITHUB_TOKEN="${GITHUB_TOKEN}"

# Use COPILOT_TOKEN for copilot if available
export GITHUB_TOKEN="${COPILOT_TOKEN:-$GITHUB_TOKEN}"
bca execute --config "$CONFIG" --work-dir /workspace/repo

# Restore original GITHUB_TOKEN for gh pr create
export GITHUB_TOKEN="${SAVED_GITHUB_TOKEN}"
gh pr create --fill
```

This ensures:
- Copilot uses `COPILOT_TOKEN` (if provided) or falls back to `GITHUB_TOKEN`
- gh CLI uses the original `GITHUB_TOKEN` for git operations and PR creation

## Setup Options

### Option 1: Fine-Grained Tokens (Most Secure)
```bash
# Token 1: For git/PR operations (Contents + Pull requests)
export GITHUB_TOKEN=github_pat_abc123...

# Token 2: For Copilot (Copilot Requests)
export COPILOT_TOKEN=github_pat_xyz789...

./bca setup --namespace bca-jobs
```

**Benefits:**
- Minimal permissions per token
- Can scope to specific repositories
- Better audit trail
- More secure

### Option 2: Classic Token (Simpler)
```bash
# One classic token with repo + read:org scopes
export GITHUB_TOKEN=ghp_classic123...

./bca setup --namespace bca-jobs
# Copilot will automatically use GITHUB_TOKEN
```

**Benefits:**
- Single token to manage
- Works for everything
- Simpler setup

**Drawbacks:**
- Broader access than needed
- Cannot scope to specific repos
- Less granular audit trail

### Option 3: Mixed (Fine-Grained + Classic)
```bash
# Fine-grained for git/PR (scoped to repos)
export GITHUB_TOKEN=github_pat_finegrained...

# Classic for Copilot (needs broader access)
export COPILOT_TOKEN=ghp_classic...

./bca setup --namespace bca-jobs
```

## Verifying Token Permissions

```bash
# Test GITHUB_TOKEN
export GITHUB_TOKEN=ghp_your_token
gh auth status
gh repo list --limit 1  # Should work

# Test push access (requires write)
git clone https://github.com/your-org/test-repo
cd test-repo
git checkout -b test-branch
echo "test" > test.txt
git add test.txt
git commit -m "Test"
git push origin test-branch  # Should succeed

# Test PR creation
gh pr create --title "Test" --body "Test"  # Should succeed

# Test COPILOT_TOKEN (if using copilot)
export GITHUB_TOKEN=$COPILOT_TOKEN
copilot --help  # Should not error about authentication
```

## Common Issues

### "Permission denied" when pushing
- Your token needs `Contents: write` permission
- Verify: `gh auth status` shows "write" access

### "Resource not accessible by integration" when creating PR
- Your token needs `Pull requests: write` permission
- Verify: `gh pr list` works

### Copilot authentication fails
- If using separate COPILOT_TOKEN, ensure it has `Copilot Requests: write`
- Verify: Token shows "Copilot" scope in GitHub settings

## Security Best Practices

1. **Use Fine-Grained Tokens**: Scope to specific repositories
2. **Set Expiration**: Max 1 year, rotate regularly
3. **Never Commit**: Use environment variables or secret managers
4. **Separate Tokens**: Use different tokens for different purposes when possible
5. **Audit Access**: Review token usage in GitHub settings regularly
