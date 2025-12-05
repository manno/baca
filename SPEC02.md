This file contains several possible next steps.


## Hydrate the prompt with a workflow agent

The current execute command takes a prompt and executes it directly, this is the coding agent. Currently it's a wrapper around copilot-cli or gemini-cli.

* Multi-layered approach, Workflow Agents trigger Coding Agents:
  * Workflow Agent: Gather information about the task interactively from the user.
  * Coding Agent: Once the interactive agent has refined the task into a clear prompt, it hands this prompt off to the "coding agent."

* Use MCP:
  * Workflow Agent gathers context information from Slack, Github, etc.

## Do not push into the repo directly

The problem is that a user could build a prompt that exposes the tokens used by the execute command to push into the repo. This is a security risk.
It should use a staging repo to create the PRs for the real repo.
That way the tokens would only allow access to the staging repo and copilot requests.

## Extend the execute command for Claude CLI

Claude can also be used interactively via the claude CLI. The execute command could be extended to also wrap claude CLI.

## Create an API Server for Apply

The API server would accept requests from multiple clients, including a web UI, MCP, Slack bot, or other interfaces.
It would have the same effect as the Apply command.

## Support GHA as a backend

This feature would enable BACA to execute code transformations using GitHub Actions workflows instead of Kubernetes jobs.
This provides an alternative execution backend that doesn't require a Kubernetes cluster.

See @docs/FEATURE_GHA.md
