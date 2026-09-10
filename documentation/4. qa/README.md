# QA

Test coverage and results for `mwanachama-backend-digitaltwin`.

As of W8 ([todo_done.md](../3.%20implementation/todo_done.md)): 56 test
cases (top-level + subtests), passing both without Postgres (`go test
./...`, the two `Memory*`-backed conformance suites plus `retention_test.go`
and `telemetry_validating_test.go`) and with it (`POSTGRES_URL=... go test
./...`, which additionally runs `TestPostgresRegistryRepository` and
`TestPostgresTelemetryRepository` — the same `RunRegistryConformance`/
`RunTelemetryConformance` suites, against `EntitygraphRegistryRepository`
and `PostgresTelemetryRepository` respectively). `go build ./...`, `go vet
./...`, `gofmt -l .` all clean in both configurations.

Both conformance suites are written to run against any implementation of
their respective interface, per this platform's "test both stores" rule
(see `mwanachama-backend-accounting`'s `conformance_test.go` for the
precedent this repo follows) — memory and Postgres are proven to agree on
every rule the suite checks, not just compile against the same interface.

**Not covered:** anything at the HTTP/gateway layer — W9 (gateway wiring)
is blocked on a capability decision and hasn't started, so there is no
route-level test surface yet. `Sweep`'s scheduling (a cron, a periodic
goroutine, …) is also untested because it doesn't exist yet — only the
purge logic itself (`retention_test.go`) is proven.
