# The objects are declared

The tables come from [`digitaltwin.blueprint.json`](../../digitaltwin.blueprint.json),
not from Go row structs. The engine is `mwanachama-backend-shared`'s `spec`,
`specstore` and `dispatch`; the org-wide strategy is
`developer/documentation/2. design/architecture-spec-driven-modules.md`.

Converted 2026-09-30. This page records the decisions the conversion turned
on; the audit that preceded it found the repo failing every gate but three.

## This was also the move off entitygraph

Before the conversion the module had two storage halves and neither was
GORM:

- the registry — `Asset`, `Connection`, `MetricDefinition`,
  `RetentionPolicy` — went through `mwanachama-backend-shared/entitygraph`
  into the shared `entities` table, via hand-written
  `assetToEntity`/`assetFromEntity` property-bag converters;
- telemetry went into its own `digitaltwin_telemetry_readings` table through
  raw `database/sql` and seven hand-written statements.

The standard normally wants entitygraph retired first and the declared-domain
move second. Here the two collapse into one step, because the declared store
*is* the GORM-backed store: converting onto `specstore` is the migration.
That was only safe because there was nothing to orphan — W9 was still
blocked, the module was never in `mwanachama-backend-api-gateway`'s `go.mod`,
and no deployment held a table of its own. A repo with live rows under the
old readable names would have needed the adoption step first.

**DSN-1701 decision 3 survives intact.** Readings were kept out of the shared
`entities` table because routing high-frequency writes through a table every
other service shares would make it the hottest table in the platform. Under
the spec engine every declared object gets its own table
(`<instance>_<hashOf(mount)>_<hashOf(module_object)>`), so `observation`
lands in a table of its own and the concern is answered by construction
rather than by a second interface.

## The module names no domain

A word that means something in one domain and nothing in another does not
belong here. The module was written for utility infrastructure and has to run
a vehicle fleet, a factory line or a building without a line changing.

| Was | Why it failed | Where it went |
| --- | --- | --- |
| `Asset` | serviceable, but paired with `AssetType`'s values it read as one domain's word, and `mwanachama-backend-assetmanager` already owns "asset" as a general concept | role `node` |
| `Connection` | fine, but the pair reads better as a graph | role `link` |
| `TelemetryReading` | "telemetry" is instrumentation's word for it | role `observation` |
| `AssetType` values — `pipeline`, `transmission_line`, `valve`, `station`, `substation`, `sensor` | six stored enum values, every one of them a utility word | out of the module entirely — `node.kind` is a declared string the domain fills |
| `StationType` — `compressor`, `pump` | a sub-kind of one domain's one kind | inside `node.doc`, the way catalog moved `Sector` |

The third row is the one that mattered. **A stored enum value outlives a
rename**, so the six asset types were the worst thing in the module.

### Why `kind` is a string and not an enum

A domain spec may set a **default and nothing else** — `spec`'s
`Blueprint.extend` refuses a domain that sets a type, a description or a
value set on a field the module declares. So "the module declares `kind` as
an enum and each domain supplies its own values" was never available.

The choice was therefore between putting the classification inside `doc`, as
catalog did with `Sector`, and keeping it as a column the module does not
interpret. It stayed a column because it does structural work: nodes are
filtered by it, observations denormalise it, and both `metric` and
`retention` scope to it. A `doc` path could not serve those without the
module learning a domain's word.

**This gives up a check the module used to perform.** `IsAssetType` refused
an unknown asset type at write time; nothing refuses an unknown kind now,
because the module has no idea what the domain's kinds are. What survives is
`required` — a node must be classified — and `matches: "name"`, which holds
the value to the strict lowercase-underscore alphabet. Whether the closed set
should come back as something a domain can declare is filed as **W14**; it is
a gap in the standard, not something to paper over here.

## What stays in Go

Only what a spec cannot say, and each piece lives with the type it is about:

- `Status.CanTransitionTo` — a state machine, not a value set. The five
  states *are* declared, and `TestVocabularyMatchesTheBlueprint` holds
  `models/vocabulary.go` and the blueprint to each other in both directions.
- `Link.EndsApart` — a field whose validity depends on another field.
- `Retention.KeepsForAPositivePeriod` and `Retention.ScopeIsOrdered` — a
  positive count, and a scope that must name a kind before it names a metric.
- `Metric.BoundsOrdered` — a cross-field comparison.

`validate.go` reads `required`, an enum's `values` and `matches` off the spec
and applies them on the way in. `patterns.go` maps the one pattern name this
module supplies, `name`, to `spec.NamePattern`. A spec naming a pattern
nobody supplies is an error, not a rule that quietly never runs.

`Check(spec, role, value)` is the same validation without a database, for a
bulk import validating what it has read before it opens a connection.

## Timestamps are strings

`spec.TypeTimestamp` is stored as text so every dialect compares it the same
way, and `specstore`'s codec calls `reflect.Value.String()` on the field. So
`RecordedAt`, `IngestedAt`, `CreatedAt`, `UpdatedAt` and `CommissionedAt` are
RFC 3339 `string`s, where the old types used `time.Time`. The JSON wire shape
is unchanged — `time.Time` already marshalled to an RFC 3339 string — but Go
callers see strings. `Sweep` still takes a `time.Time` and formats at the
boundary, because computing a cutoff is arithmetic and belongs in a time
type.

Observations validate that `recorded_at` parses as RFC 3339 before being
stored, which is what the old `RecordedAt.IsZero()` check stood for.

## Two domains, and one of them is unprovisioned

`spec/examples/utility.digitaltwin.json` is the original domain — assets,
connections, readings, with a `doc` path index on `station_type` so the
domain's own vocabulary is fast without the module learning the word.

`spec/examples/fleet.digitaltwin.json` is a vehicle fleet nobody has
provisioned, which is the point: the shipped example must not also be
production config (catalog's CAT6). It names its objects differently
(`vehicle`, `coupling`, `channel`, `sample`, `retention_rule`), sets a
`default` on `retention_days`, indexes two of its own document paths, and
declares an object of its own — `driver` — with no role, to prove a domain
can add what the module does not have.

`TestTwoDomainsCoexist` migrates both into one database and writes through
both managers; `TestEveryExampleFitsTheTypes` builds a manager over *every*
shipped spec, which is what catches drift under the domain nobody loaded.

## The codec, and two things a rewrite gets wrong

- **A column is found by field name, never by json tag.** `NodeID` is
  `node_id`. A tag-reading codec would silently stop storing any `json:"-"`
  field.
- **Every declared column is written on every write.** A map missing a key
  means "leave it alone" to an update, so omitting empty values would make
  clearing a field impossible — a node would keep the `notes` that an update
  is meant to drop. `TestPG_EveryDeclaredColumnIsWrittenOnEveryWrite` covers
  it.

The spec and the types are checked against each other when the manager is
built: a declared column with no field, or a field with no column, fails in
`NewTwinManager` rather than dropping a value on every write.
