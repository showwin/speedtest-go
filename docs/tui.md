# Terminal UI

This document describes the terminal UI used by the `speedtest-go` command.

## Goals

- Keep progress output stable during latency, download, upload, and packet-loss
  measurements.
- Ensure that only one goroutine writes interactive output to the terminal.
- Keep task callbacks independent from terminal cursor management.
- Preserve machine-readable output for `--json` and `--jsonl`.
- Preserve line-oriented output for `--unix` and redirected output.

## Output Modes

| Mode | Renderer | Output contract |
| --- | --- | --- |
| Default TTY | Interactive | Animated task progress and committed final results |
| Default non-TTY | Plain | One final line per task, without ANSI sequences |
| `--unix` | Plain | Line-oriented text without ANSI sequences |
| `--json` | Silent | One JSON document without task output |
| `--jsonl` | Silent | One JSON object per line without task output |

Errors in a silent mode are written to standard error so standard output stays
machine-readable.

## Architecture

The task layer and terminal layer communicate through two small interfaces:

```go
type taskRenderer interface {
    NewTask(message string) taskHandle
    Println(message string)
    BlankLine()
    Stop()
}

type taskHandle interface {
    Update(message string)
    Complete()
    Error()
}
```

`TaskManager` creates handles and never performs cursor movement itself. Task
callbacks only update in-memory task state.

Three renderer implementations provide the output modes:

- `interactiveRenderer` owns all ANSI output and runs one render goroutine.
- `plainRenderer` writes one final line for each completed or failed task.
- `silentRenderer` discards task output for JSON and JSONL modes.

All terminal output owned by the TUI, including separator blank lines, goes
through the renderer. A direct stdout write while the interactive renderer is
active invalidates its cursor position and can leave a stale spinner frame on
screen.

## Interactive Rendering

The interactive renderer maintains only active tasks in its dynamic region.
When a task completes, its final line is committed to the terminal and removed
from that region. This avoids redrawing the full command history on every
animation frame.

The render loop performs these operations:

1. Snapshot task state under a mutex.
2. Clear only the lines drawn by the previous frame.
3. Print newly completed task lines once.
4. Draw the current active task lines.
5. Wait for the next frame or shutdown request.

The frame cadence has a lower bound of 120 milliseconds. Measurement callbacks
may update task state more frequently, but only the newest value present at the
next frame is rendered. Messages are converted to one line and truncated to the
current terminal width so wrapping cannot corrupt cursor positioning.

The running and completed symbols use ysmrr's default bright green. Error
symbols use bright red. Messages remain in the terminal's default color. Width
calculation and truncation happen before ANSI styles are applied.

Spinner frames come from `ysmrr v0.6.0`. The project does not use ysmrr's
spinner-manager lifecycle because its `Stop()` method does not provide a render
goroutine completion barrier.

## Lifecycle

- One interactive renderer is created for the command.
- `Stop()` is idempotent.
- Normal shutdown waits for asynchronous tasks before stopping the renderer.
- Interactive `Stop()` waits for the final frame and cursor restoration.
- No new renderer is created between discovery, measurement, or server phases.
- The server list stops the renderer before switching to direct text output.

## Tests

`tui_test.go` covers:

- Coalescing high-frequency task updates into the latest rendered value.
- Serializing separator blank lines with task state changes.
- Applying status colors without affecting terminal-width calculations.
- Idempotent renderer shutdown.
- Message sanitization and terminal-width truncation.
- Single-emission behavior for the plain renderer.
- The silent renderer contract.
- Shutdown waiting for asynchronous task completion.

Runtime verification should cover Windows Terminal, PowerShell, a Unix terminal,
redirected standard output, every output mode, and multi-server measurements.

## Maintenance Rules

- Do not write ANSI sequences outside `tui.go`.
- Do not write directly to stdout while the interactive renderer is active.
- Do not let measurement callbacks write directly to the terminal.
- Do not start more than one interactive renderer.
- Keep JSON and JSONL diagnostics on standard error.
- Add a renderer test when changing task state or output lifecycle behavior.
