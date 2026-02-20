# orchschema

Local-first event schema, JSONL event log, and replay projector for a deterministic agent orchestrator core.

## What is included

- Typed schema contracts in `internal/schema`
- Validation helpers and redaction/truncation utilities
- Append-only JSONL event log writer and reader in `internal/log`
- In-memory projector in `internal/projector` that rebuilds run state from events
- Demo command in `cmd/orchschema` that writes and replays a deterministic run

## Run the demo

```bash
go run ./cmd/orchschema
```

Expected output shape:

```text
run_id=run_demo_000001 tasks=1 agents=1 artifacts=1 final_status=completed
log=/tmp/orchschema_demo_run.jsonl
```

## Run tests

```bash
go test ./...
```
