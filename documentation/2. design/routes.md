# The route table is declared

Every address this module answers is an entry in
[`digitaltwin.operations.json`](../../digitaltwin.operations.json). There are
no hand-written handlers: `routes/routes.go` is an adapter of about forty
lines that hands the operations file, the sentinel map and the anonymous
allowlist to `mwanachama-backend-shared/dispatch`, which builds the
`http.HandlerFunc` for each entry by reflecting on `TwinManager`.

Adding an address is an edit to the JSON file plus a method on the manager.
If the two disagree — a wrong arity, a missing method, a sentinel the spec
maps but nobody supplies — `dispatch.Dispatch` refuses to build the table at
all, so the failure arrives at startup rather than on the first request.

## The sixteen operations

| Method | Path | Manager call | Action | Status |
|---|---|---|---|---|
| POST | `/nodes` | `CreateNode` | `digitaltwin.node.create` | 201 |
| GET | `/nodes` | `ListNodes` | `digitaltwin.node.list` | 200 |
| GET | `/nodes/{nodeID}` | `GetNode` | `digitaltwin.node.read` | 200 |
| PUT | `/nodes/{nodeID}` | `UpdateNode` | `digitaltwin.node.update` | 200 |
| POST | `/nodes/{nodeID}/status` | `SetNodeStatus` | `digitaltwin.node.set_status` | 200 |
| POST | `/links` | `CreateLink` | `digitaltwin.link.create` | 201 |
| DELETE | `/links/{relation}/{fromNodeID}/{toNodeID}` | `DeleteLink` | `digitaltwin.link.delete` | 204 |
| GET | `/nodes/{nodeID}/links` | `ListLinks` | `digitaltwin.link.list` | 200 |
| POST | `/metrics` | `UpsertMetric` | `digitaltwin.metric.upsert` | 200 |
| GET | `/metrics` | `ListMetrics` | `digitaltwin.metric.list` | 200 |
| POST | `/retentions` | `SetRetention` | `digitaltwin.retention.set` | 200 |
| GET | `/retentions` | `ListRetentions` | `digitaltwin.retention.list` | 200 |
| POST | `/observations` | `RecordObservation` | `digitaltwin.observation.record` | 201 |
| POST | `/observations/batch` | `RecordObservations` | `digitaltwin.observation.record_batch` | 201 |
| GET | `/observations` | `QueryObservations` | `digitaltwin.observation.query` | 200 |
| GET | `/nodes/{nodeID}/observations/latest` | `Latest` | `digitaltwin.observation.latest` | 200 |

The addresses changed with the conversion — `/assets` became `/nodes`,
`/telemetry/readings` became `/observations`, and so on. Nothing consumed
this module at the time (W9 was still blocked and the gateway never had it
in its `go.mod`), so no caller had to be migrated. A domain word in an
address outlives a rename exactly the way a stored enum value does, which is
why `TestNoDomainWordsInTheDeclaredRouteTable` reads the paths and actions
back and fails on one.

## The gate is data

`AnonymousActions` is an allowlist of action ids and it is **empty**. This
module publishes nothing anonymously: every one of the sixteen operations
arrives gated, and `TestEveryRouteIsGatedUntilNamedAnonymous` asserts that
`PublicRoutes` is empty and all sixteen are in the gated half.

The allowlist names what is public, never what is protected. An operation
added to the file later and not named there arrives gated, so the failure
direction is a 401 rather than an unpublished surface on the open internet.

This module still has **no auth model of its own** — the same position
`mwanachama-backend-catalog` and `mwanachama-backend-assetmanager` take. A
mount supplies a `dispatch.Authorizer` through `routes.Mount`, and a refused
action answers 403 without reaching the manager. W9's blocking question —
which capability guards these routes — is unchanged by the conversion; what
changed is that the answer is now a list of action ids rather than a wrapper
around each Go function.

## The address outranks the body

`update_node` binds `{"from": "path", "as": "nodeID", "into": "ID"}`, applied
after the body is decoded. Without it a caller could `PUT /nodes/a` with a
body claiming to be `b` and rewrite a different node.
`TestTheAddressOutranksTheBody` sends exactly that request and asserts the
addressed node is the one updated and the other is untouched.

## Errors

The `errors` block maps every exported sentinel to a status. An unmapped
sentinel is redacted to a 500 with `"internal error"` — catalog's CAT7, a
400-shaped refusal arriving as an unexplained 500 — so
`TestEverySentinelIsMappedToAStatus` calls
`dispatch.UnmappedSentinels` and fails if a sentinel is exported by the
package, or supplied to the dispatcher, without a status.

| Sentinel | Status |
|---|---|
| `ErrInvalid` | 400 |
| `ErrInvalidLink` | 400 |
| `ErrUnknownMetric` | 400 |
| `ErrNodeNotFound` | 404 |
| `ErrLinkNotFound` | 404 |
| `ErrMetricNotFound` | 404 |
| `ErrRetentionNotFound` | 404 |
| `ErrObservationNotFound` | 404 |

These are the statuses the deleted `routes/errors.go` produced, transcribed
rather than reconsidered. `ErrUnknownMetric` at 400 is W12's fix and is
carried across unchanged.

## One read, two shapes

`GET /nodes/{nodeID}/observations/latest` answers one observation when
`metric_name` is given and a list of the latest per metric when it is not —
the behaviour the deleted `GetLatestReading` handler had. The dispatcher
routes one address to one manager method, so `Latest` returns `any` and
makes the choice itself. `TestLatestSwitchesOnTheMetricName` covers both
shapes.
