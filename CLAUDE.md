# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-digitaltwin

A standalone digital twin: things with a service life and a place, the
directed graph between them, and a historical record of what each one
measured over time — not just current-state attributes. Module path
`github.com/aosanya/mwanachama-backend-digitaltwin`. Decided by the
dev-research session recorded at
[DSN-1701](../developer/documentation/2.%20design/todo.md), which was about
utility infrastructure — pipelines, transmission lines, valves, stations,
substations and sensors. **That is a domain this module runs, not what this
module is**; it ships as `spec/examples/utility.digitaltwin.json`.

## Not part of the electorate platform

Confirmed in the research session: no tie to Mwanachama's tier/member/
campaign model. This is a domain-agnostic backend service in the same
family as `mwanachama-backend-taskmanager` and
`mwanachama-backend-accounting` — a tool, not an extension of the
tier-mobilization product.

Domain logic AND storage both live in this package, imported directly by
whatever mounts it — no separate service, no gRPC, no proto — the same shape
`mwanachama-backend-catalog` and `mwanachama-backend-actor` already took.

## Objects are declared, not written

Converted to the declared-domain standard on 2026-09-30. The tables come
from a JSON spec, not from Go row structs — see
[documentation/2. design/declared-domain.md](documentation/2.%20design/declared-domain.md)
and `developer/documentation/2. design/architecture-spec-driven-modules.md`.

- `digitaltwin.blueprint.json` — **the module's five objects, declared
  once**: `node`, `link`, `metric`, `observation`, `retention`. Embedded into
  the root package and reached through `Blueprint()`, `LoadSpec(path)` and
  `ParseSpec(raw)`. Load a domain spec through those, never through
  `spec.Load`, or its roled objects arrive with no fields.
- `mwanachama-backend-shared/spec`, `specstore` and `dispatch` are the
  engine. `spec.Migrate` creates a table per declared object and its indexes;
  `Provision(db, s)` is this module's name for it, and `cmd/ddl` prints the
  statements so a spec can be reviewed as SQL before it is trusted.
- `spec/examples/utility.digitaltwin.json` and `fleet.digitaltwin.json` — the
  same module under two domains. The fleet is deliberately a domain nobody
  has provisioned, so the shipped example is not also production config.

**A domain names objects; it does not re-declare them.** Its spec supplies
`instance`, the name and table each role lands in, its own indexes, and a
**default** on a declared field — nothing else. A domain needing a field of
its own declares an object of its own, with no role, the way the fleet spec
declares `driver`.

## The rule that matters most: this module names no domain

**A word that means something in one domain and nothing in another does not
belong in this module.** It was written for pipelines and transmission lines
and must run a vehicle fleet or a factory line without a line changing.

The six asset types — `pipeline`, `transmission_line`, `valve`, `station`,
`substation`, `sensor` — were **stored enum values**, which is the worst
version of this failure because a stored value outlives a rename. They are
gone: `node.kind` is a declared string the domain fills, and the module never
interprets it. `station_type` went inside `node.doc`. The full table of what
moved and why is in
[documentation/2. design/declared-domain.md](documentation/2.%20design/declared-domain.md).

A domain spec may set **a default and nothing else**, so a domain-supplied
value set was never available — which is why `kind` is a string rather than
an enum, and why nothing now refuses an unknown kind. That is a real check
the module gave up; it is filed as **W14**, not papered over.

`domain_agnostic_test.go` enforces the rule over four surfaces: identifiers
in `models/`, `routes/` and the root package; every stored enum value; every
blueprint role, field, value and index; and every path and action in the
declared route table. Domain words stay allowed in a domain's own spec, which
is exactly where they belong.

## Rules are declared too, where they can be

`validate.go` reads `required`, an enum's `values` and `matches` off the spec
and applies them on the way in. `patterns.go` maps the one pattern name this
module supplies — `name` — to `spec.NamePattern`; a spec naming a pattern
nobody supplies is an error, not a rule that quietly never runs.

**What stays in Go is what a spec cannot say**, each piece with the type it
is about: `Status.CanTransitionTo` (a state machine), `Link.EndsApart` (a
field whose validity depends on another), `Retention.KeepsForAPositivePeriod`
and `ScopeIsOrdered`, and `Metric.BoundsOrdered`.

`TestVocabularyMatchesTheBlueprint` holds `models/vocabulary.go` and the
blueprint's declared `values` to each other in **both** directions.

`Check(spec, role, value)` is the same validation without a database, for a
bulk import validating what it read before it opens a connection.

## The route table is declared

`digitaltwin.operations.json` declares all sixteen addresses, and
`routes/routes.go` is a forty-line adapter over
`mwanachama-backend-shared/dispatch`. There are no hand-written handlers and
no route builders to name — adding an address is an edit to that file plus a
method on `TwinManager`. See
[documentation/2. design/routes.md](documentation/2.%20design/routes.md).

`AnonymousActions` is empty: this module publishes nothing anonymously, and
an operation added later and not named there arrives gated. The module still
has **no auth model of its own** — a mount supplies the `dispatch.Authorizer`.
W9's blocking question is unchanged by the conversion.

## Storage invariants

- **A column is found by field name, never by json tag.** `NodeID` is
  `node_id`. A tag-reading codec silently stops storing any `json:"-"` field.
- **Every declared column is written on every write.** A map missing a key
  means "leave it alone" to an update, so omitting empty values would make
  clearing a field impossible.
- **A required field must not carry a default** — the default is exactly what
  lets an omitted value pass unnoticed. `TestRequiredFieldsHaveNoDefault`
  asserts it over the blueprint and every shipped spec.
- **Timestamps are RFC 3339 strings, not `time.Time`**, because the codec
  stores a timestamp as text. `Sweep` still takes a `time.Time` and formats
  at the boundary.
- **A table is `<instance>_hashOf(<mount>)_hashOf(<module>_<object>)`.**
  Assert on `spec.RawNameFor` in tests, never on a physical name literal, and
  exclude `%_spec_table_names` from anything that counts tables.
- **Identifiers reach SQL as text** — `spec.NamePattern` and
  `spec.DocPathPattern` refuse anything outside a strict alphabet. Never
  relax either to "escape it instead".

## Scope decisions (session of 2026-09-03)

- **Digital twin, not a static registry.** Historical observations are kept
  per node, not just current attributes — this is what makes it a twin rather
  than a CRUD asset registry like `mwanachama-backend-assetmanager`.
- **Observations live in a table of their own, never in a table shared with
  unrelated services.** This was DSN-1701 decision 3, and it is now answered
  by construction: the spec engine gives every declared object its own table,
  so the old second repository interface was no longer what enforced it.
- **Node kinds are the domain's, not the module's** — see above. The v1
  utility set (`pipeline`, `transmission_line`, `valve`, `station`,
  `substation`, `sensor`) now lives in `spec/examples/utility.digitaltwin.json`.
- **Nodes connect as a directed graph** (`link`) — the relation vocabulary
  (`connects_to`/`part_of`/`monitors`) is a working v1 set, not a finalised
  decision. These stayed a module-declared enum because they are relations
  between things rather than words about any one subject.
- **Measurement shape is customer/expert-configured, not hardcoded per node
  kind** — `observation` is a generic long-format record backed by `metric`
  as the expert-configured vocabulary, and `node.doc` carries whatever static
  properties a deployment's expert configures.

## Status (2026-09-30)

W1–W8 done. W9 (gateway wiring) is ⏸️ **blocked** — deliberately, on a
capability decision rather than on code; the conversion did not change that,
only the shape of the answer (a list of action ids rather than a wrapper per
Go function). W13 is an open 🐞 and was carried across **unfixed**, still
pinned by `TestW13_BroadRulePurgesWhatASpecificRuleShouldHaveProtected`,
which asserts the current behaviour and fails loudly if the defect is fixed.
W14 is new, from the conversion.

## What is superseded

- **`entitygraph_registry.go` and `entitygraph_converters.go`** — deleted
  2026-09-30 (W15). The property-bag converters are what `spec.Migrate` and
  `specstore`'s codec replace. This was also the repo's move off entitygraph;
  there was nothing to adopt because no deployment ever held a table.
- **`telemetry_postgres.go` and `schema.go`** — deleted 2026-09-30 (W15).
  Seven hand-written SQL statements and a hand-maintained DDL, replaced by
  the declared `observation` object and `cmd/ddl`.
- **`memory_registry.go` and `memory_telemetry.go`** — deleted 2026-09-30
  (W15). The tests run against the shipped `utility` spec on SQLite rather
  than a second hand-written implementation of the same interface.
- **`registry_repository.go` and `telemetry_repository.go`** — deleted
  2026-09-30 (W15). One `TwinManager` replaces the two interfaces, because
  the dispatcher binds one manager; what the split protected is now the
  spec's per-object table.
- **`routes/`'s five hand-written handler files, its own `Route`/`Pattern`
  pair and its `wire.go`** — deleted 2026-09-30 (W16). The table is declared
  in `digitaltwin.operations.json`; `Route` is `httpwire.Route` and the
  builder ladder is `dispatch.Table`, both from shared rather than copied
  here.
- **`Asset.Validate`, `Connection.Validate`, `MetricDefinition.Validate`,
  `TelemetryReading.Validate`, `RetentionPolicy.Validate`, and the four
  `IsX` membership maps** — gone 2026-09-30 (W15). The rules a spec can state
  are read off the spec; what a spec cannot state stays with its type. The
  per-field doc comments went with them: they were a hand-written copy of the
  blueprint's `description`, which is where that prose lives now.

## Conventions

- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- Before wiring into `mwanachama-backend-api-gateway`, check for
  naming/scope overlap the same way `taskmanager`/`accounting` do.
- `go test ./...` (sqlite via `glebarez/sqlite`) is the expected way to
  verify a change here — do not reach for a real Postgres. The `jsonb`
  behaviours SQLite cannot stand in for live in `postgres_integration_test.go`
  (`//go:build integration`, gated on `POSTGRES_URL`, run by `make test-pg`),
  which is **not** part of `go test ./...`.
- No Go file over 300 lines; split by responsibility.
- There are no hand-written route builders left to name: an address is an
  entry in `digitaltwin.operations.json`.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
