// metric.go — HTTP routes over digitaltwin.RegistryRepository's
// expert-configured vocabulary: MetricDefinition (UpsertMetricDefinition,
// ListMetricDefinitions) and RetentionPolicy (SetRetentionPolicy,
// ListRetentionPolicies). See doc.go.
package routes

import (
	"net/http"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
)

func MetricDefinitionRoutes(registry digitaltwin.RegistryRepository) []Route {
	return []Route{
		{Method: "POST", Path: "/metric-definitions", Handler: UpsertMetricDefinition(registry)},
		{Method: "GET", Path: "/metric-definitions", Handler: ListMetricDefinitions(registry)},
	}
}

// UpsertMetricDefinition handles POST /metric-definitions — create-or-update
// identified by (AssetType, Name), matching UpsertMetricDefinition's own
// find-or-update contract.
func UpsertMetricDefinition(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m digitaltwin.MetricDefinition
		if err := readJSON(r, &m); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		out, err := registry.UpsertMetricDefinition(r.Context(), m)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListMetricDefinitions handles GET /metric-definitions?asset_type= — empty
// asset_type matches every asset type.
func ListMetricDefinitions(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetType := digitaltwin.AssetType(r.URL.Query().Get("asset_type"))
		out, err := registry.ListMetricDefinitions(r.Context(), assetType)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func RetentionPolicyRoutes(registry digitaltwin.RegistryRepository) []Route {
	return []Route{
		{Method: "POST", Path: "/retention-policies", Handler: SetRetentionPolicy(registry)},
		{Method: "GET", Path: "/retention-policies", Handler: ListRetentionPolicies(registry)},
	}
}

// SetRetentionPolicy handles POST /retention-policies — create-or-update
// identified by (AssetType, MetricName). Configuration only — see
// telemetry.go's RetentionPolicy doc comment for what this does and does
// not enforce.
func SetRetentionPolicy(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p digitaltwin.RetentionPolicy
		if err := readJSON(r, &p); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		out, err := registry.SetRetentionPolicy(r.Context(), p)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func ListRetentionPolicies(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := registry.ListRetentionPolicies(r.Context())
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
