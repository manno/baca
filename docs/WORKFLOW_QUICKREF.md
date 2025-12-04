# Workflow Agent - Quick Reference

## Command Syntax

```bash
# Interactive workflow (default)
baca workflow --change <file> --namespace <ns>

# Skip interactive mode
baca workflow --change <file> --skip-interactive

# Wait for completion
baca workflow --change <file> --wait
```

## Interactive Commands

During the workflow session:

| Input | Action |
|-------|--------|
| `<text>` | Answer the question |
| `skip` | Skip current question |
| `done` | Skip remaining questions and proceed |
| `y` or `yes` | Confirm refined prompt |
| `n` or `no` | Cancel workflow |

## Workflow Steps

1. **Parse** - Read Change YAML
2. **Question** - Agent asks clarifying questions (4 default)
3. **Context** - Optional additional context input
4. **Refine** - Generate structured prompt
5. **Confirm** - Show refined prompt, get confirmation
6. **Execute** - Apply to backend (create K8s jobs)

## Refined Prompt Format

```markdown
# Task Overview
[Your original prompt]

# Clarifications from Discussion
- [Answer 1]
- [Answer 2]
- [Answer 3]

# Additional Context
## [type] ([source])
[Context content]
```

## Example Session

```bash
$ baca workflow --change fix-bug.yaml --namespace baca-jobs

=== BACA Workflow Agent ===
Q1: Which specific files or directories should be modified?
> src/api/users.go

Q2: Are there any related issues, PRs, or documentation to reference?
> Issue #789

Q3: What is the expected behavior after the changes?
> No null pointer crashes

Q4: Are there any constraints?
> skip

Additional context? (Enter to skip)
> [Enter]

=== Refined Prompt ===
# Task Overview
Fix null pointer crash in users API

# Clarifications from Discussion
- src/api/users.go
- Issue #789
- No null pointer crashes
======================

Proceed? (y/n): y

✓ Jobs created
```

## Tips

✅ **Be Specific**: "src/handlers/*.go" not "handler files"  
✅ **Reference Issues**: "Issue #123" not "there's an issue"  
✅ **Describe Behavior**: "Return 400 with JSON" not "handle errors"  
✅ **Use 'done'**: Skip remaining questions if you have enough info  
✅ **Use 'skip'**: Skip irrelevant questions  

## Comparison with Apply

| Feature | `workflow` | `apply` |
|---------|-----------|---------|
| Interactive | Yes (default) | No |
| Prompt refinement | Yes | No |
| Clarifying questions | Yes | No |
| Context gathering | Yes | No |
| Direct execution | With `--skip-interactive` | Yes |
| Use case | Complex tasks | Simple tasks |

## When to Use Workflow

Use `workflow` when:
- Task is complex or ambiguous
- You want to add more context
- Multiple people need to clarify requirements
- You want structured output

Use `apply` when:
- Prompt is already detailed
- Task is straightforward
- No additional context needed
- Automation/scripting

## File Structure

```
internal/workflow/
├── types.go      # Session, Message, Context types
├── session.go    # Session management
├── refiner.go    # Prompt refinement logic
└── agent.go      # Workflow orchestration

cmd/
└── workflow.go   # CLI command

tests/workflow/
└── workflow_test.go  # 8 tests

docs/
├── FEATURE_WORKFLOW.md           # Full documentation
└── WORKFLOW_IMPLEMENTATION.md    # Implementation details
```

## Testing

```bash
# Run workflow tests
ginkgo -v ./tests/workflow/

# Run all tests
ginkgo -v ./tests/...

# Build
go build .
```

## Related Documentation

- [FEATURE_WORKFLOW.md](FEATURE_WORKFLOW.md) - Complete feature guide
- [WORKFLOW_IMPLEMENTATION.md](WORKFLOW_IMPLEMENTATION.md) - Implementation details
- [SPEC02.md](../SPEC02.md) - Original specification
- [README.md](../README.md) - Main documentation
