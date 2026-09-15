// telemetry.go — HTTP routes over digitaltwin.TelemetryRepository:
// RecordReading, RecordReadingBatch, QueryReadings, and one combined latest
// read (GetLatestReading), selected by whether ?metric_name= is present —
// the same "narrow read via query param, list via its absence" shape
// taskmanager's ListProjects/GetProjectByName use. See doc.go.
package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
)

func TelemetryRoutes(telemetry digitaltwin.TelemetryRepository) []Route {
	return []Route{
		{Method: "POST", Path: "/telemetry/readings", Handler: RecordReading(telemetry)},
		{Method: "POST", Path: "/telemetry/readings/batch", Handler: RecordReadingBatch(telemetry)},
		{Method: "GET", Path: "/telemetry/readings", Handler: QueryReadings(telemetry)},
		{Method: "GET", Path: "/assets/{assetID}/telemetry/latest", Handler: GetLatestReading(telemetry)},
	}
}

func RecordReading(telemetry digitaltwin.TelemetryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reading digitaltwin.TelemetryReading
		if err := readJSON(r, &reading); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		out, err := telemetry.Record(r.Context(), reading)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// RecordReadingBatch handles POST /telemetry/readings/batch — the expected
// path for a high-frequency source. All-or-nothing: if any reading fails
// Validate, none are stored.
func RecordReadingBatch(telemetry digitaltwin.TelemetryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var readings []digitaltwin.TelemetryReading
		if err := readJSON(r, &readings); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		out, err := telemetry.RecordBatch(r.Context(), readings)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// QueryReadings handles GET
// /telemetry/readings?asset_id=&metric_name=&from=&to=&limit= — From/To are
// RFC 3339 timestamps; either omitted (or unparseable) leaves that side
// unbounded.
func QueryReadings(telemetry digitaltwin.TelemetryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := digitaltwin.TelemetryFilter{
			AssetID:    q.Get("asset_id"),
			MetricName: q.Get("metric_name"),
		}
		if from := q.Get("from"); from != "" {
			if t, err := time.Parse(time.RFC3339, from); err == nil {
				filter.From = t
			}
		}
		if to := q.Get("to"); to != "" {
			if t, err := time.Parse(time.RFC3339, to); err == nil {
				filter.To = t
			}
		}
		if l := q.Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil {
				filter.Limit = n
			}
		}
		out, err := telemetry.Query(r.Context(), filter)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// GetLatestReading handles GET /assets/{assetID}/telemetry/latest?metric_name=.
// With metric_name, resolves Latest — one reading. Without, resolves
// LatestByAsset — the asset's full current-state snapshot across every
// metric recorded against it.
func GetLatestReading(telemetry digitaltwin.TelemetryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetID := r.PathValue("assetID")
		if metricName := r.URL.Query().Get("metric_name"); metricName != "" {
			out, err := telemetry.Latest(r.Context(), assetID, metricName)
			if err != nil {
				writeDigitaltwinErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, out)
			return
		}
		out, err := telemetry.LatestByAsset(r.Context(), assetID)
		if err != nil {
			writeDigitaltwinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
