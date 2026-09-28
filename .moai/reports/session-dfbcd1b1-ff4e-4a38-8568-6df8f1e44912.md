# Session Summary: dfbcd1b1-ff4e-4a38-8568-6df8f1e44912

**Total Hook Invocations:** 378

**Session Duration:** 2h11m51.161s

## Event Breakdown

- **FileChanged**: 2
- **InstructionsLoaded**: 60
- **PermissionRequest**: 9
- **PostToolUse**: 61
- **PostToolUseFailure**: 3
- **PreToolUse**: 131
- **SessionStart**: 4
- **Stop**: 11
- **SubagentStart**: 10
- **SubagentStop**: 74
- **TaskCompleted**: 2
- **UserPromptSubmit**: 11

## Decision Breakdown

- **allow**: 131

## Top 5 Slowest Hook Executions

| # | Event | Handler | Tool | Duration (ms) |
|---|-------|---------|------|---------------|
| 1 | SessionStart | *hook.autoUpdateHandler |  | 151 |
| 2 | SessionStart | *hook.sessionStartHandler |  | 12 |
| 3 | PreToolUse | *hook.preToolHandler | Write | 2 |
| 4 | PreToolUse | *hook.preToolHandler | Write | 2 |
| 5 | PreToolUse | *hook.preToolHandler | Write | 1 |

## Errors (0)

_No errors recorded._

