// asset.go — HTTP routes over digitaltwin.RegistryRepository's Asset
// operations: CreateAsset, GetAsset, ListAssets, UpdateAsset,
// UpdateAssetStatus. See doc.go.
package routes

import (
	"net/http"
	"strconv"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
)

// AssetRoutes returns the plain Asset CRUD plus the dedicated status-change
// route (UpdateAssetStatus enforces AssetStatus.CanTransitionTo; general
// UpdateAsset never touches Status — see registry_repository.go).
func AssetRoutes(registry digitaltwin.RegistryRepository) []Route {
	return []Route{
		{Method: "POST", Path: "/assets", Handler: CreateAsset(registry)},
		{Method: "GET", Path: "/assets", Handler: ListAssets(registry)},
		{Method: "GET", Path: "/assets/{assetID}", Handler: GetAsset(registry)},
		{Method: "PUT", Path: "/assets/{assetID}", Handler: UpdateAsset(registry)},
		{Method: "POST", Path: "/assets/{assetID}/status", Handler: UpdateAssetStatus(registry)},
	}
}

func CreateAsset(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var a digitaltwin.Asset
		if err := readJSON(r, &a); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		out, err := registry.CreateAsset(r.Context(), a)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func GetAsset(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := registry.GetAsset(r.Context(), r.PathValue("assetID"))
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListAssets handles GET /assets?asset_type=&status=&limit=.
func ListAssets(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := digitaltwin.AssetFilter{
			AssetType: digitaltwin.AssetType(q.Get("asset_type")),
			Status:    digitaltwin.AssetStatus(q.Get("status")),
		}
		limit := 0
		if l := q.Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil {
				limit = n
			}
		}
		out, err := registry.ListAssets(r.Context(), filter, limit)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateAsset handles PUT /assets/{assetID} — Name/Location/InstalledAt/
// Notes/Attributes only; AssetType, StationType, and Status are immutable
// through this route (see UpdateAssetStatus for Status).
func UpdateAsset(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var a digitaltwin.Asset
		if err := readJSON(r, &a); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		a.ID = r.PathValue("assetID")
		out, err := registry.UpdateAsset(r.Context(), a)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateAssetStatus handles POST /assets/{assetID}/status.
func UpdateAssetStatus(registry digitaltwin.RegistryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Status digitaltwin.AssetStatus `json:"status"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		out, err := registry.UpdateAssetStatus(r.Context(), r.PathValue("assetID"), body.Status)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
