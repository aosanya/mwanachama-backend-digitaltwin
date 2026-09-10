# Design

The asset registry / topology / telemetry schema, and how it maps onto
`mwanachama-backend-shared`'s entity-graph store — the same retargeting
`mwanachama-backend-taskmanager` and `mwanachama-backend-accounting`
already did for their own domains, **with one deliberate fork**: telemetry
readings are kept out of that shared store entirely.

This file records the decisions from the dev-research session of
2026-09-03 ([DSN-1701](../../../developer/documentation/2.%20design/todo.md))
and the open questions it did not settle, so neither has to be re-derived
from a chat transcript.

## Why telemetry readings are not entity-graph vertices

`mwanachama-backend-shared`'s entity-graph store is a **single shared
Postgres `entities` table**, used by every service in this family —
`taskmanager`'s Tasks, `accounting`'s ledger (once wired), and this repo's
own asset registry all live in it side by side. That's fine at
registry-write volumes: an asset is created, updated, or has its status
changed occasionally.

Telemetry is a different order of magnitude. The research session
described ingestion as "very high frequency" via API push, with retention
customer-defined rather than platform-fixed. Writing every reading as an
entity-graph vertex would make `entities` the hottest table in the whole
platform — for every unrelated service, not just this one. So:

- **Registry** (`Asset`, `Connection`, `MetricDefinition`,
  `RetentionPolicy`) → `mwanachama-backend-shared`'s entity-graph store,
  via `RegistryRepository` (once W4 wires it — currently backed only by
  `MemoryRegistryRepository`).
- **Telemetry** (`TelemetryReading`) → a dedicated, service-owned table,
  via `TelemetryRepository` (currently only `MemoryTelemetryRepository`).

This is the one place this repo's architecture is *not* a straight port of
`taskmanager`'s pattern — everywhere else, it is.

## Asset registry

**Six v1 asset types** (`AssetType`, `vocabulary.go`): `pipeline`,
`transmission_line`, `valve`, `station`, `substation`, `sensor`. `station`
additionally carries a `StationType` (`compressor` | `pump`) — the only
asset type with a required sub-kind.

**Lifecycle status** (`AssetStatus`), a hand-rolled state machine mirroring
`taskmanager.TaskStatus.CanTransitionTo`:

```
planned        → operational, decommissioned
operational    → maintenance, fault, decommissioned
maintenance    → operational, decommissioned
fault          → maintenance, operational, decommissioned
decommissioned → (none — terminal)
```

**No fixed per-type attribute schema.** Pipeline diameter/material,
TransmissionLine voltage rating, and every other type-specific property
were never specified in the research session. Rather than invent field
names with no grounding, `Asset.Attributes` is a flexible
`map[string]string` — the same "we only supply the tool" reasoning DSN-1701
decision 6 established for telemetry, extended to the registry. **This is
a real open question, not a permanent stance** — if a deployment needs
strongly-typed, validated attributes per asset type, that's a schema this
repo doesn't have yet.

## Topology

`Connection` is a directed edge between two assets — `FromAssetID`,
`ToAssetID`, `Kind`. Confirmed wanted in the research session; the edge
vocabulary was not:

| Kind | Meaning | Status |
| --- | --- | --- |
| `connects_to` | generic directed link — source feeds/is upstream of target | working v1 |
| `part_of` | source is a physical component of target (e.g. Valve part_of Pipeline) | working v1 |
| `monitors` | Sensor → asset it measures (optional — see below) | working v1 |

**A `Sensor` is not a mandatory intermediary for a reading.** DSN-1701
decision 6: a `TelemetryReading` may name any asset directly; `monitors`
documents instrumentation topology when it exists, it isn't a precondition
for readings to exist.

## Telemetry

`TelemetryReading` is generic long-format —
`(asset_id, asset_type, metric_name, value, unit, recorded_at)` — rather
than fixed columns like `pressure`/`voltage` per asset type, because the
metric vocabulary itself is expert-configured (decision 6). `MetricName`/
`Unit` are expected to match a registered `MetricDefinition`, but this is
**not enforced at ingest time** — see "Open questions" below.

`RecordedAt` (when the physical measurement happened) is kept distinct
from `IngestedAt` (when this service received it) — a high-frequency push
source is not guaranteed to deliver in order or without delay.

**Reads:** `Query` (time-range, the historical record); `Latest` /
`LatestByAsset` (current-state snapshot — the read a digital twin exists to
answer, "what is this asset doing right now").

**Append-only.** No `Update`, no `Delete`, the same discipline
`mwanachama-backend-accounting`'s `LedgerRepository` documents for its own
entries — a wrong reading is corrected by recording a fresh one, not by
editing history.

## Fields (first cut — `Asset`)

| Field | Type | Notes |
| --- | --- | --- |
| `ID` | uuid | |
| `Name` | string | required |
| `AssetType` | enum | one of six v1 types; immutable after creation |
| `StationType` | enum, nullable | required iff `AssetType == station` |
| `Status` | enum | defaults to `planned` on create |
| `Location` | `{lat, lng}`, nullable | |
| `InstalledAt` | timestamp, nullable | |
| `Notes` | text | |
| `Attributes` | `map[string]string` | expert-configured, no fixed schema — see above |
| `CreatedAt` / `UpdatedAt` | timestamp | |

## Fields (first cut — `TelemetryReading`)

| Field | Type | Notes |
| --- | --- | --- |
| `ID` | uuid | |
| `AssetID` / `AssetType` | uuid / enum | `AssetType` denormalised to avoid a registry join per read |
| `MetricName` / `Unit` | string | expected to match a `MetricDefinition`, not enforced |
| `Value` | float | |
| `RecordedAt` | timestamp | physical measurement time — the ordering/query key |
| `IngestedAt` | timestamp | server receipt time |
| `Quality` | string, optional | free-form, not a validated vocabulary |

No balance-style derived column anywhere — `Latest`/`LatestByAsset` fold
over stored readings on read, the same "no stored derived value" rule
`accounting`'s ledger applies to balances.

## Open questions — flagged, not silently decided

**1. Retention default/mechanism.** ~~Confirmed customer-defined in the
research session; no default period or enforcement job specified.~~
**Resolved (W6).** `Sweep` (`retention.go`) reads every `RetentionPolicy`
and calls `TelemetryRepository.PurgeBefore` for each — proven against both
backends. Still true as designed: no policy means keep forever, and
**nothing invokes `Sweep` on a schedule** — that's a deployment decision,
left for W9.

**2. Ingestion frequency ceiling.** "Very high" was never quantified.
Decides whether the dedicated telemetry table needs time-based
partitioning from day one or can start as a single indexed table and be
partitioned later once real volume is measured. **Still open** —
`telemetry_postgres.go` ships as a single indexed table (W5); partition by
`recorded_at` if/when real volume shows it's needed.

**3. Connection edge vocabulary completeness.** `connects_to`/`part_of`/
`monitors` is a working v1 set built from the confirmed examples
(Pipeline↔Valve, TransmissionLine↔Substation, Sensor↔asset), now wired
into the schema (W4: one `Asset` `TypeDefinition`, self-referencing
relationships for all three kinds plus their inverses). Real topology
modeling may need more (e.g. bidirectional peering, redundancy/backup
links) — **still open**, not designed.

**4. Ingest-time metric validation.** ~~Whether `Record`/`RecordBatch`
should reject a `MetricName`/`Unit` that doesn't match a registered
`MetricDefinition`.~~ **Resolved (W8), as an opt-in, not a default.** The
plain `TelemetryRepository` never does a registry lookup, keeping a
high-frequency write cheap. `ValidatingTelemetryRepository`
(`telemetry_validating.go`) wraps any `TelemetryRepository` +
`RegistryRepository` pair and rejects a reading whose `(AssetType,
MetricName)` — and `Unit`, when set — doesn't match a registered
`MetricDefinition` (type-scoped or any-type). A caller who wants
enforcement opts in; nothing changes for one who doesn't.

**5. Access control.** ~~No capability/authorization model has been
designed.~~ **Resolved by precedent — none belongs here.**
`mwanachama-backend-taskmanager` has no authorization logic in its own
package at all; every route that calls it is capability-gated at
`mwanachama-backend-api-gateway`'s HTTP layer instead
(`d.auth(d.operator(CapTaskManagerWrite, handler))`). This repo follows
the same split — `RegistryRepository`/`TelemetryRepository` carry no auth
concept of their own. **The actual decision —
who may register assets, post telemetry, read what — still has to be made
before W9 wires any route**, but it's a gateway-layer decision (a new
`CapDigitalTwinRead`/`CapDigitalTwinWrite` pair, or something narrower),
not something this package needs to carry.
