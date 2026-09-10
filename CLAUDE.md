# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-digitaltwin

A standalone infrastructure-asset digital twin — pipelines, transmission
lines, and their component assets (valves, compressor/pump stations,
substations, sensors/meters), each with historical telemetry, not just
current-state attributes. Module path
`github.com/aosanya/mwanachama-backend-digitaltwin`. Decided by the
dev-research session recorded at
[DSN-1701](../developer/documentation/2.%20design/todo.md).

## Not part of the electorate platform

Confirmed in the research session: no tie to Mwanachama's tier/member/
campaign model. This is a domain-agnostic backend service in the same
family as `mwanachama-backend-taskmanager` and
`mwanachama-backend-accounting` — a tool, not an extension of the
tier-mobilization product.

Architected exactly like
[mwanachama-backend-taskmanager](../mwanachama-backend-taskmanager): a Go
package on `mwanachama-backend-shared`'s Postgres entity-graph store,
imported directly by `mwanachama-backend-api-gateway`. No gRPC, no
standalone service.

## Scope decisions (session of 2026-09-03)

- **Digital twin, not a static registry.** Historical telemetry is kept
  per asset, not just current attributes — this is what makes it a twin
  rather than a CRUD asset registry like `mwanachama-backend-assetmanager`.
- **Telemetry readings live in a dedicated table, never in the shared
  `entities` table.** `mwanachama-backend-shared`'s entity-graph store is a
  single shared Postgres table used across this whole service family;
  routing "very high frequency" API-pushed readings through it would make
  it the hottest table in the platform for every unrelated service. Only
  the asset **registry** (Pipeline/TransmissionLine/etc. and their static
  attributes, plus topology) goes through the shared store — see
  `RegistryRepository` (`registry_repository.go`) vs. `TelemetryRepository`
  (`telemetry_repository.go`).
- **v1 asset types**: `Pipeline`, `TransmissionLine`, `Valve`, `Station`
  (compressor/pump), `Substation`, `Sensor`/`Meter` — see `AssetType` in
  `vocabulary.go`.
- **Assets connect as a directed graph** (`Connection`, `connection.go`) —
  topology is wanted; the edge vocabulary (`connects_to`/`part_of`/
  `monitors`) is a working v1 set, not a finalized decision.
- **Reading/metric shape is customer/expert-configured, not hardcoded per
  asset type** — "these are configurable once we have an expert, we only
  supply the tool." `TelemetryReading` is a generic long-format record
  (`asset_id, asset_type, metric_name, value, unit, recorded_at`) rather
  than fixed columns like `pressure`/`voltage`, backed by
  `MetricDefinition` (`metric.go`) as the expert-configured vocabulary.
  The same reasoning extends to `Asset.Attributes` — there is no fixed
  per-type schema for Pipeline diameter, TransmissionLine voltage rating,
  etc.; those were never specified and are left as a flexible
  `map[string]string` rather than invented.

## Status (2026-09-03)

W1–W8 done — registry and telemetry both work against real Postgres
(`EntitygraphRegistryRepository`, `PostgresTelemetryRepository`), retention
enforcement (`Sweep`) and opt-in ingest validation
(`ValidatingTelemetryRepository`) are built and tested. **W9 (gateway
wiring) is ⏸️ blocked**, deliberately — see
[documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md):
every route in `mwanachama-backend-api-gateway` is capability-gated
(`d.auth(d.operator(CapXxx, handler))`), and no `CapDigitalTwinRead`/
`CapDigitalTwinWrite`-shaped decision has been made. Wiring routes without
one would either leave a live, unauthenticated write surface on a real
deployment or require inventing an authorization model with no grounding —
neither was done unilaterally.

## Open questions (flagged, not silently decided)

- **Ingestion frequency ceiling.** "Very high" was not quantified —
  `telemetry_postgres.go` ships as a single indexed table (W5); partition
  by `recorded_at` if/when real volume shows it's needed.
- **Connection edge vocabulary.** `connects_to`/`part_of`/`monitors` is a
  working set, wired into the schema (W4), not confirmed as complete or
  final.
- **Access control for W9.** Resolved that no capability model belongs
  *inside* this package (see status above and design doc open question 5)
  — but the actual gateway-side capability decision still has to be made
  before any route wires in.

Retention (was: no mechanism) and ingest-time metric validation (was:
unresolved) are both now resolved — see
[documentation/2. design/README.md](documentation/2.%20design/README.md)
open questions 1 and 4.

## Conventions

- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- Before wiring into `mwanachama-backend-api-gateway`, check for
  naming/scope overlap the same way `taskmanager`/`accounting` do.
