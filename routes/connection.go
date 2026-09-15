// connection.go — HTTP routes over digitaltwin.RegistryRepository's
// Connection (directed graph edge) operations: CreateConnection,
// DeleteConnection, ListConnections. See doc.go.
package routes

import (
	"net/http"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
)

func ConnectionRoutes(registry digitaltwin.RegistryRepository) []Route {
	return []Route{
		{Method: "POST", Path: "/connections", Handler: CreateConnection(registry)},
		{Method: "DELETE", Path: "/connections/{kind}/{fromAssetID}/{toAssetID}", Handler: DeleteConnection(registry)},
		{Method: "GET", Path: "/assets/{assetID}/connections", Handler: ListConnections(registry)},
	}
}

// CreateConnection handles POST /connections. Re-creating an existing edge
// is idempotent — see CreateConnection's own doc comment — so this always
// answers 201, never a conflict.
func CreateConnection(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c digitaltwin.Connection
		if err := readJSON(r, &c); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		out, err := registry.CreateConnection(r.Context(), c)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// DeleteConnection handles DELETE /connections/{kind}/{fromAssetID}/{toAssetID} —
// the same (Kind, FromAssetID, ToAssetID) triple that identifies one edge.
func DeleteConnection(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind := digitaltwin.ConnectionKind(r.PathValue("kind"))
		err := registry.DeleteConnection(r.Context(), r.PathValue("fromAssetID"), r.PathValue("toAssetID"), kind)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListConnections handles GET
// /assets/{assetID}/connections?kind=&direction=outbound|inbound — empty
// kind matches every kind; direction defaults to outbound.
func ListConnections(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kind := digitaltwin.ConnectionKind(q.Get("kind"))
		dir := digitaltwin.DirectionOutbound
		if q.Get("direction") == "inbound" {
			dir = digitaltwin.DirectionInbound
		}
		out, err := registry.ListConnections(r.Context(), r.PathValue("assetID"), kind, dir)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
