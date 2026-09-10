# mwanachama-backend-digitaltwin — documentation

## Layout

Four folders, in SDLC order, and everything lives under one of them.

| Folder | What's inside |
|--------|---------------|
| [1. requirements/](1.%20requirements/) | Problem, vision and scope for a standalone infrastructure-asset digital twin. |
| [2. design/](2.%20design/) | The asset registry / topology / telemetry schema, the decisions from the 2026-09-03 dev-research session, and the open questions still unconfirmed. |
| [3. implementation/](3.%20implementation/) | The work: `todo.md` (open board), `todo_done.md` (completed rows + board context). |
| [4. qa/](4.%20qa/) | Test coverage and results. |

## Boards and status

| File | What it holds |
| --- | --- |
| [todo.md](3.%20implementation/todo.md) | Open task board |
| [todo_done.md](3.%20implementation/todo_done.md) | Completed rows + board context |

## What this repo is

A standalone infrastructure-asset digital twin — no tie to Mwanachama's
tier/member/campaign model. Built on
[mwanachama-backend-shared](../mwanachama-backend-shared)'s entity-graph
store for the asset registry and topology, with telemetry readings in a
dedicated, service-owned table, and imported directly by
[mwanachama-backend-api-gateway](../mwanachama-backend-api-gateway) — no
gRPC, no sub-service shape. See [CLAUDE.md](../CLAUDE.md) for the full
scope decisions and open questions.
