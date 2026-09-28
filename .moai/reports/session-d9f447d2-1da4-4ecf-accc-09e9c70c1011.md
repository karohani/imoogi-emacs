# Session Summary: d9f447d2-1da4-4ecf-accc-09e9c70c1011

**Total Hook Invocations:** 108

**Session Duration:** 5h38m47.64s

## Event Breakdown

- **ConfigChange**: 3
- **CwdChanged**: 2
- **InstructionsLoaded**: 29
- **PermissionRequest**: 1
- **PostToolUse**: 4
- **PostToolUseFailure**: 1
- **PreToolUse**: 25
- **SessionStart**: 4
- **Stop**: 4
- **StopFailure**: 14
- **SubagentStop**: 1
- **UserPromptSubmit**: 20

## Decision Breakdown

- **allow**: 25

## Top 5 Slowest Hook Executions

| # | Event | Handler | Tool | Duration (ms) |
|---|-------|---------|------|---------------|
| 1 | SessionStart | *hook.autoUpdateHandler |  | 645 |
| 2 | SessionStart | *hook.sessionStartHandler |  | 15 |
| 3 | PreToolUse | *hook.preToolHandler | Write | 2 |
| 4 | PreToolUse | *hook.preToolHandler | Edit | 1 |
| 5 | Stop | *hook.stopHandler |  | 1 |

## Errors (0)

_No errors recorded._

