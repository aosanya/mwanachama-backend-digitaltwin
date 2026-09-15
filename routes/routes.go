package routes

import (
	"net/http"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
)

// Route is one HTTP endpoint: a method, a path relative to this package's
// mount point, and the handler. The mounting process wraps Handler with its
// own auth/capability gates and builds the mux itself — see doc.go.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Pattern returns the Go 1.22+ ServeMux pattern for this route under
// prefix, e.g. Pattern("/v1/digitaltwin") on {Method: "GET", Path:
// "/assets/{assetID}"} yields "GET /v1/digitaltwin/assets/{assetID}".
func (rt Route) Pattern(prefix string) string {
	return rt.Method + " " + prefix + rt.Path
}

// Routes returns every route this package defines, over registry and
// telemetry. Concatenates the per-model route lists — see asset.go,
// connection.go, metric.go, and telemetry.go.
func Routes(registry digitaltwin.RegistryRepository, telemetry digitaltwin.TelemetryRepository) []Route {
	var out []Route
	out = append(out, AssetRoutes(registry)...)
	out = append(out, ConnectionRoutes(registry)...)
	out = append(out, MetricDefinitionRoutes(registry)...)
	out = append(out, RetentionPolicyRoutes(registry)...)
	out = append(out, TelemetryRoutes(telemetry)...)
	return out
}
