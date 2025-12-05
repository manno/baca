package gha

import (
	"fmt"
	"os"
	"path/filepath"
)

const workflowYAML = `name: BACA Execute

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

    - name: Install BACA and agents
      run: |
        # Install BACA CLI
        curl -L https://github.com/manno/baca/releases/latest/download/baca-linux-amd64 -o /usr/local/bin/baca
        chmod +x /usr/local/bin/baca

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
          '{'
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
        git config --global user.name "BACA Bot"
        git config --global user.email "baca-bot@users.noreply.github.com"
        baca execute --config "$CONFIG" --work-dir .

    - name: Create Pull Request
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: |
        gh pr create --fill || echo "PR creation failed or no changes"
`

// WriteWorkflowFile creates the GitHub Actions workflow file.
func WriteWorkflowFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	if err := os.WriteFile(path, []byte(workflowYAML), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}
