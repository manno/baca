#!/bin/bash
set -e

# Extract owner and repo from URL
REPO_PATH=$(echo "$ORIGINAL_REPO_URL" | sed -e 's|^https://github.com/||' -e 's|^git@github.com:||' -e 's|\.git$||')
echo "Original repo: $REPO_PATH"

# Determine fork owner (override or authenticated user)
if [ -n "$FORK_ORG" ]; then
  FORK_OWNER="$FORK_ORG"
  echo "Using specified fork organization: $FORK_OWNER"
else
  FORK_OWNER=$(gh api user --jq .login)
  echo "Using authenticated user as fork owner: $FORK_OWNER"
fi

# Check if fork already exists
REPO_NAME=$(echo "$REPO_PATH" | cut -d'/' -f2)
if gh repo view "$FORK_OWNER/$REPO_NAME" >/dev/null 2>&1; then
  echo "Fork already exists: $FORK_OWNER/$REPO_NAME"
  
  # Verify it's actually a fork
  IS_FORK=$(gh api "repos/$FORK_OWNER/$REPO_NAME" --jq .fork)
  if [ "$IS_FORK" != "true" ]; then
    echo "ERROR: Repository $FORK_OWNER/$REPO_NAME exists but is NOT a fork!"
    echo "BACA requires repos to be forks. Please delete $FORK_OWNER/$REPO_NAME or use a different repository."
    exit 1
  fi
  
  # Try to sync fork with upstream (non-fatal if it fails due to conflicts/divergence)
  echo "Attempting to sync fork with upstream..."
  if gh repo sync "$FORK_OWNER/$REPO_NAME" --branch main; then
    echo "Fork synced successfully"
  else
    echo "Warning: Fork sync failed (possibly due to conflicts or divergent history)"
    echo "This is usually fine - BACA will work from the fork's current state"
  fi
else
  echo "Creating fork: $FORK_OWNER/$REPO_NAME"
  if [ -n "$FORK_ORG" ]; then
    gh repo fork "$REPO_PATH" --clone=false --org "$FORK_ORG"
  else
    gh repo fork "$REPO_PATH" --clone=false
  fi
  # Wait a moment for fork to be created
  sleep 2
fi

# Write fork URL to file for next container
echo "https://github.com/$FORK_OWNER/$REPO_NAME" > /workspace/fork-url.txt
echo "Fork URL: $(cat /workspace/fork-url.txt)"
