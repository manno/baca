# Workflow Agent Feature

The Workflow Agent provides an interactive session to help refine your code transformation tasks before execution. It asks clarifying questions, gathers context, and produces a refined prompt that is passed to the coding agent.

## Overview

Traditional approach (apply):
```
User → Prompt → Coding Agent → Code Changes
```

Workflow approach:
```
User → Workflow Agent (interactive) → Refined Prompt → Coding Agent → Code Changes
```

## Why Use Workflow Mode?

1. **Better Results**: More detailed prompts lead to better code transformations
2. **Clarification**: Ask questions before execution, not after
3. **Context Gathering**: Include relevant issues, PRs, documentation
4. **Validation**: Review the refined prompt before execution
5. **Iterative**: Refine your task description through conversation

## Usage

### Basic Interactive Workflow

```bash
baca workflow --change my-change.yaml --namespace baca-jobs
```

The workflow agent will:
1. Parse your initial Change definition
2. Ask clarifying questions
3. Gather additional context
4. Generate a refined prompt
5. Ask for confirmation
6. Apply the change to all repositories

### With MCP Context Gathering (NEW!)

Automatically gather context from external sources:

```bash
# Gather from GitHub (issues and PRs)
baca workflow --change my-change.yaml --mcp github

# Multiple sources (when available)
baca workflow --change my-change.yaml --mcp github,slack
```

**What MCP Does:**
- Searches GitHub issues in your target repositories
- Finds relevant pull requests
- Uses your prompt keywords as search terms
- Includes top 5 most relevant items
- Adds full content to refined prompt with source URLs

**Requirements:**
- GitHub: `gh` CLI installed and authenticated
- Slack: Coming in future release

### Skip Interactive Mode

If you want to use the apply flow directly (no refinement):

```bash
baca workflow --change my-change.yaml --skip-interactive
```

This is equivalent to `baca apply` but provides a consistent interface.

### Wait for Completion

```bash
baca workflow --change my-change.yaml --wait
```

Waits for all jobs to complete before returning.

## Interactive Session Flow

### 1. Initial Prompt

Start with a Change YAML:

```yaml
kind: Change
apiVersion: v1
spec:
  prompt: "Add error handling to HTTP handlers"
  repos:
  - https://github.com/myorg/api-server
  agent: copilot-cli
```

### 2. Clarifying Questions

The workflow agent asks questions to refine your task:

```
=== BACA Workflow Agent ===
I'll help you refine your task before executing it.
Answer the questions below, or type 'skip' to skip a question.
Type 'done' when you're ready to proceed.

Q1: Which specific files or directories should be modified?
> src/handlers/*.go

Q2: Are there any related issues, PRs, or documentation to reference?
> Issue #123 and PR #456 have context on error handling requirements

Q3: What is the expected behavior after the changes?
> All handlers should return JSON error responses with proper status codes

Q4: Are there any constraints or requirements to consider?
> skip

Do you have any additional context or requirements? (Enter to skip)
> Follow the error handling pattern from the users handler
```

### 3. Refined Prompt

The agent generates a structured prompt:

```
=== Refined Prompt ===
# Task Overview
Add error handling to HTTP handlers

# Clarifications from Discussion
- src/handlers/*.go
- Issue #123 and PR #456 have context on error handling requirements
- All handlers should return JSON error responses with proper status codes

# Additional Context
## additional_context (manual)
Follow the error handling pattern from the users handler

======================

Proceed with this refined prompt? (y/n): y
```

### 4. Execution

Once confirmed, the refined prompt is passed to the coding agent and jobs are created.

## Prompt Refinement Structure

The refined prompt is structured with sections:

### Task Overview
The original prompt from your Change definition.

### Clarifications from Discussion
User responses to clarifying questions, formatted as bullet points.

### Additional Context
Any additional context gathered, including:
- Manual input from the user
- MCP-gathered context (future: GitHub issues, Slack messages, etc.)

## Tips for Best Results

### Be Specific
When answering questions, provide specific details:
- ✅ "src/handlers/*.go and pkg/middleware/"
- ❌ "handler files"

### Reference Issues/PRs
Link to relevant GitHub issues or PRs:
- ✅ "See issue #123 for requirements"
- ❌ "There's an issue about this"

### Describe Expected Behavior
Clearly state what the code should do after changes:
- ✅ "Return 400 Bad Request with JSON error body for invalid input"
- ❌ "Handle errors properly"

### Use 'done' to Skip Remaining Questions
If you've provided enough context:
```
Q2: Are there any related issues?
> done
```

### Use 'skip' for Individual Questions
Skip questions that aren't relevant:
```
Q4: Are there any constraints?
> skip
```

## Architecture

### Components

#### Session (`internal/workflow/session.go`)
Manages the interactive session state:
- Conversation log (questions and answers)
- Gathered context
- Refined prompt

#### Refiner (`internal/workflow/refiner.go`)
Generates refined prompts from session data:
- Template-based prompt construction
- Structured output with sections
- Context integration

#### Agent (`internal/workflow/agent.go`)
Orchestrates the workflow:
- Interactive question loop
- User input handling
- Confirmation flow
- Integration with backend

### Data Flow

```
1. Parse Change YAML
   ↓
2. Create Session
   ↓
3. Generate Questions (Refiner)
   ↓
4. Interactive Loop (Agent)
   - Ask question
   - Read user input
   - Store in session
   ↓
5. Refine Prompt (Refiner)
   ↓
6. Show refined prompt
   ↓
7. Confirm with user
   ↓
8. Create new Change with refined prompt
   ↓
9. Apply to backend (existing flow)
```

## Future Enhancements

### MCP Integration
Planned for future releases:

```bash
baca workflow --change my-change.yaml --mcp github,slack
```

This will:
- Fetch related GitHub issues and PRs
- Pull Slack conversations for context
- Include documentation from wikis
- Gather test results and CI logs

### LLM-Based Refinement
Future versions may use an LLM to:
- Generate context-aware questions
- Synthesize refined prompts intelligently
- Suggest improvements to task descriptions

### Conversation History
Save and resume workflow sessions:

```bash
# Save session
baca workflow --change my-change.yaml --save-session session.json

# Resume later
baca workflow --resume-session session.json
```

## Examples

### Example 1: Bug Fix

```bash
$ baca workflow --change fix-bug.yaml

Q1: Which specific files or directories should be modified?
> src/api/users.go

Q2: Are there any related issues, PRs, or documentation to reference?
> Issue #789 describes the null pointer crash

Q3: What is the expected behavior after the changes?
> No crashes when user data is missing, return 404 instead

Proceed with this refined prompt? (y/n): y
```

### Example 2: Feature Addition

```bash
$ baca workflow --change add-feature.yaml

Q1: Which specific files or directories should be modified?
> src/auth/ and src/middleware/

Q2: Are there any related issues, PRs, or documentation to reference?
> RFC #45 describes the OAuth2 flow requirements

Q3: What is the expected behavior after the changes?
> Support OAuth2 authentication alongside existing token auth

Additional context?
> Keep backward compatibility with existing tokens
```

### Example 3: Refactoring

```bash
$ baca workflow --change refactor.yaml

Q1: Which specific files or directories should be modified?
> all files in src/

Q2: Are there any related issues, PRs, or documentation to reference?
> skip

Q3: What is the expected behavior after the changes?
> Use structured logging throughout, no behavior changes

Q4: Are there any constraints?
> Must not change public APIs
```

## Testing

Run workflow tests:

```bash
ginkgo -v ./tests/workflow/
```

Tests cover:
- Session management
- Message and context tracking
- Prompt refinement
- Non-interactive mode

## Related Documentation

- [SPEC02.md](../SPEC02.md) - Workflow agent specification
- [README.md](../README.md) - Main documentation
- [AGENTS.md](../AGENTS.md) - AI assistant guide
