# Orchschema

A local-first event schema, JSONL event log, and replay projector designed for a deterministic agent orchestrator core.

`orchschema` provides the foundational abstractions for building robust, observable, and auditable AI agent systems. It treats agent runs as a deterministic stream of events, allowing you to record, replay, and visualize the entire lifecycle of an AI workflow.

## Features

- **Typed Event Schemas**: Strictly defined contracts in `internal/schema` for all aspects of an agent run (planning, tool calls, model requests, approvals, etc.).
- **Append-Only Event Log**: A high-performance JSONL event log writer and reader in `internal/log`.
- **In-Memory Projector**: A state builder in `internal/projector` that reconstructs the current run state purely from the event stream.
- **TUI Dashboard**: A live-updating terminal UI dashboard (`cmd/dashboard`) to monitor agent runs in real-time.
- **Simulators**: Included demo commands to generate and replay strictly-compliant event logs.

## Quick Start

### 1. Run the Demo Orchestrator

This command writes a deterministic mock agent run to an event log and replays it using the projector:

```bash
go run ./cmd/orchschema
```

*Expected output:*

```text
run_id=run_demo_000001 tasks=1 agents=1 artifacts=1 final_status=completed
log=/tmp/orchschema_demo_run.jsonl
```

### 2. View the Live Dashboard

You can visualize the generated event log using the provided TUI dashboard:

```bash
go run ./cmd/dashboard --log=/tmp/orchschema_demo_run.jsonl
```

*(You can also run a slow-writer simulation to see the dashboard update in real time: `go run ./cmd/slowwrite`)*

## Testing

Run the test suite:

```bash
go test ./...
```

## Project Structure

- `internal/schema/`: The core domain model, event types, and payloads.
- `internal/log/`: JSONL serialization and file I/O for the event stream.
- `internal/projector/`: The state machine that reduces the event log into a current snapshot.
- `internal/tui/`: Bubble Tea models and logic for the terminal dashboard.
- `cmd/orchschema/`: A demo writer and projector.
- `cmd/dashboard/`: The live TUI dashboard viewer.
- `cmd/slowwrite/`: A simulation tool that writes events slowly for dashboard testing.
