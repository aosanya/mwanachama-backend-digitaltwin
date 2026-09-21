# mwanachama-backend-digitaltwin (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in
[todo_done.md](todo_done.md).

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| W9 | Wire into `mwanachama-backend-api-gateway`: new `internal/domain/digitaltwin` + `internal/api/http/digitaltwin_handlers*.go` consuming this repo's `RegistryRepository`/`TelemetryRepository`, registered in `stores.go`/`router.go` | ⏸️ Blocked | **Blocked on a capability decision, not on code.** Every route in that gateway is wrapped `d.auth(d.operator(CapXxx, handler))` (see `taskmanager_handlers_task.go`) — wiring routes with no `CapDigitalTwinRead`/`CapDigitalTwinWrite`-shaped capability would either leave a live, unauthenticated write surface on a gateway serving a real deployment, or require inventing an authorization model with no grounding (exactly what W7 flagged as undesigned). Needs an explicit decision — who may register/read assets and post telemetry — before any route lands. Not attempted without that sign-off. |
