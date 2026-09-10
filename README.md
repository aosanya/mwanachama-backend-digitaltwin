# mwanachama-backend-digitaltwin

A standalone infrastructure-asset digital twin — pipelines, transmission
lines, and their component assets (valves, compressor/pump stations,
substations, sensors/meters) — each with a registry entry and historical
telemetry, not just current-state attributes.

No tie to Mwanachama's tier/member/campaign model. No gRPC, no sub-service
shape. Built on [mwanachama-backend-shared](../mwanachama-backend-shared)'s
Postgres entity-graph store for the asset **registry** and topology, with
telemetry readings kept in a **dedicated, service-owned table** — see
[CLAUDE.md](CLAUDE.md) for why. Imported directly by
[mwanachama-backend-api-gateway](../mwanachama-backend-api-gateway), the
same architecture as
[mwanachama-backend-taskmanager](../mwanachama-backend-taskmanager) and
[mwanachama-backend-accounting](../mwanachama-backend-accounting).

See [documentation/](documentation/) for design and task board.
