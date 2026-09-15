// Package routes is mwanachama-backend-digitaltwin's own HTTP surface:
// decode a request, call one [digitaltwin.RegistryRepository] or
// [digitaltwin.TelemetryRepository] method, encode the response — the same
// shape the root package's Go callers already get, just reachable from an
// HTTP mux. It exists so a route's request/response shape and its domain
// logic are authored and reviewed together, in the package that owns the
// domain, rather than reimplemented a second time in whichever process
// happens to mount this package.
//
// Every route here is a single repository call with no policy layered on
// top and no capability gate — access control is still an open question
// for W9 (CLAUDE.md's "Open questions"), not a decision made here. [Routes]
// returns every route as one list; [AssetRoutes]/[ConnectionRoutes]/
// [MetricDefinitionRoutes]/[RetentionPolicyRoutes]/[TelemetryRoutes] return
// one model's routes at a time, for a mounting process that wraps
// different groups in different policy.
//
// Deliberately NOT here: [digitaltwin.Sweep] and the TelemetryRepository
// method it drives, PurgeBefore. Sweep needs both repositories plus a
// caller-supplied "now", is not a plain per-row CRUD operation, and — per
// retention.go's own doc comment — invoking it is a deployment decision
// ("a cron, a periodic goroutine...") left to whoever wires this in, not
// something this package should expose as an ad hoc HTTP-triggerable bulk
// delete.
//
// A route built from this package still needs a caller-identity/capability
// gate wrapped around it before it is safe to serve — this package answers
// "what happens once that gate has passed", never "who may pass it". The
// mounting process supplies that gate by wrapping the http.HandlerFunc this
// package returns, not by this package reaching for a session or a
// capability itself.
package routes
