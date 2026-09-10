# Requirements

## Problem

Physical infrastructure — pipelines, transmission lines, and the valves,
stations, substations and sensors attached to them — has no home in this
platform family. `mwanachama-backend-assetmanager` tracks generic
current-state assets; nothing tracks a physical asset's *history* the way
a digital twin needs to: what a pipeline's pressure was an hour ago, not
just what it is now.

## Vision

A standalone digital twin: a registry of physical assets and how they
connect to each other, plus a historical record of what each one measured
over time — high-frequency, API-pushed telemetry, retained on a
customer-defined policy rather than a platform-wide default.

## Scope (session of 2026-09-03 — see [DSN-1701](../../../developer/documentation/2.%20design/todo.md))

**In scope:**
- Asset registry: `Pipeline`, `TransmissionLine`, `Valve`, `Station`
  (compressor/pump), `Substation`, `Sensor`/`Meter` — each with a lifecycle
  status (`planned → operational → maintenance/fault → decommissioned`)
  and expert-configured static attributes.
- Directed-graph topology between assets (`Connection`).
- A metric vocabulary (`MetricDefinition`) that a deployment's expert
  configures — this platform supplies the tool, not the specific metrics.
- High-frequency telemetry ingestion (`TelemetryReading`), queryable by
  asset/metric/time-range, with a "current state" read per asset
  (`Latest`/`LatestByAsset`).
- Retention as customer-defined configuration (`RetentionPolicy`) — the
  vocabulary, not yet the enforcement.

**Explicitly out of scope:**
- Any tie to Mwanachama's tier/member/campaign model — this is a
  standalone tool, confirmed in the research session.
- Alerting/thresholds beyond the optional `MinValue`/`MaxValue` bounds
  already carried on `MetricDefinition` — no alerting mechanism is
  designed.
- A generic asset registry for non-infrastructure assets — that's
  `mwanachama-backend-assetmanager`'s job, not this repo's.

**Not yet decided, flagged rather than assumed (see
[2. design/README.md](../2.%20design/README.md)):**
- The exact retention default/enforcement mechanism.
- The ingestion frequency ceiling and whether it requires time-based
  partitioning from day one.
- Whether the `connects_to`/`part_of`/`monitors` edge vocabulary is
  complete.
- Whether ingest should validate a reading's `MetricName`/`Unit` against a
  registered `MetricDefinition`.
