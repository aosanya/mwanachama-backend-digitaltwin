# mwanachama-backend-digitaltwin

A standalone digital twin: things with a service life and a place, the
directed graph between them, and a historical record of what each one
measured over time — not just current-state attributes.

It names no domain. The module declares five objects — `node`, `link`,
`metric`, `observation`, `retention` — and a domain spec says what they are
called and where they land. It ships two: utility infrastructure (pipelines,
transmission lines, valves, stations, substations, sensors) and a vehicle
fleet, under `spec/examples/`.

No tie to Mwanachama's tier/member/campaign model. No gRPC, no sub-service
shape. Storage and domain logic both live in this package, imported directly
by whatever mounts it.

Objects come from [`digitaltwin.blueprint.json`](digitaltwin.blueprint.json)
and addresses from
[`digitaltwin.operations.json`](digitaltwin.operations.json); the engine is
[mwanachama-backend-shared](../mwanachama-backend-shared)'s `spec`,
`specstore` and `dispatch`. There are no row structs and no hand-written
handlers — see
[documentation/2. design/declared-domain.md](documentation/2.%20design/declared-domain.md)
and [routes.md](documentation/2.%20design/routes.md).

```sh
make build     # compile
make test      # unit tests, sqlite in memory
make test-pg   # the jsonb behaviours, needs POSTGRES_URL
go run ./cmd/ddl spec/examples/utility.digitaltwin.json postgres
```

See [documentation/](documentation/) for design and task board, and
[CLAUDE.md](CLAUDE.md) for the invariants.
