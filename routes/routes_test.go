package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/routes"
)

// fixedTime is a stand-in RecordedAt — TelemetryReading.Validate refuses a
// zero time, and the exact instant is irrelevant to what these tests check.
func fixedTime() time.Time {
	return time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
}

func patterns(rts []routes.Route, prefix string) []string {
	out := make([]string, len(rts))
	for i, rt := range rts {
		out[i] = rt.Pattern(prefix)
	}
	return out
}

func assertPatterns(t *testing.T, got []routes.Route, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d routes, want %d: %v", len(got), len(want), patterns(got, ""))
	}
	for i, p := range patterns(got, "") {
		if p != want[i] {
			t.Fatalf("route %d: got %q, want %q", i, p, want[i])
		}
	}
}

func TestAssetRoutes(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	rts := routes.AssetRoutes(registry)
	assertPatterns(t, rts, []string{
		"POST /assets",
		"GET /assets",
		"GET /assets/{assetID}",
		"PUT /assets/{assetID}",
		"POST /assets/{assetID}/status",
	})
}

func TestConnectionRoutes(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	rts := routes.ConnectionRoutes(registry)
	assertPatterns(t, rts, []string{
		"POST /connections",
		"DELETE /connections/{kind}/{fromAssetID}/{toAssetID}",
		"GET /assets/{assetID}/connections",
	})
}

func TestMetricDefinitionRoutes(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	rts := routes.MetricDefinitionRoutes(registry)
	assertPatterns(t, rts, []string{"POST /metric-definitions", "GET /metric-definitions"})
}

func TestRetentionPolicyRoutes(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	rts := routes.RetentionPolicyRoutes(registry)
	assertPatterns(t, rts, []string{"POST /retention-policies", "GET /retention-policies"})
}

func TestTelemetryRoutes(t *testing.T) {
	telemetry := digitaltwin.NewMemoryTelemetryRepository()
	rts := routes.TelemetryRoutes(telemetry)
	assertPatterns(t, rts, []string{
		"POST /telemetry/readings",
		"POST /telemetry/readings/batch",
		"GET /telemetry/readings",
		"GET /assets/{assetID}/telemetry/latest",
	})
}

func TestRoutes_ConcatenatesAll(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	telemetry := digitaltwin.NewMemoryTelemetryRepository()
	all := routes.Routes(registry, telemetry)
	want := 5 + 3 + 2 + 2 + 4 // Asset + Connection + MetricDefinition + RetentionPolicy + Telemetry
	if len(all) != want {
		t.Fatalf("got %d routes, want %d: %v", len(all), want, patterns(all, ""))
	}
}

func TestRoutes_NoDuplicatePatterns(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	telemetry := digitaltwin.NewMemoryTelemetryRepository()
	seen := make(map[string]bool)
	for _, p := range patterns(routes.Routes(registry, telemetry), "") {
		if seen[p] {
			t.Errorf("duplicate route pattern: %s", p)
		}
		seen[p] = true
	}
}

func TestRoute_PatternWithPrefix(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	rts := routes.AssetRoutes(registry)
	if got := rts[0].Pattern("/v1/digitaltwin"); got != "POST /v1/digitaltwin/assets" {
		t.Fatalf("got %q", got)
	}
}

// mux builds a real net/http.ServeMux from routes.Routes, the same way a
// mounting process would — proves PathValue extraction and JSON wire shape
// work against an actual HTTP request, not just a direct handler call.
func mux(registry digitaltwin.RegistryRepository, telemetry digitaltwin.TelemetryRepository) *http.ServeMux {
	m := http.NewServeMux()
	for _, rt := range routes.Routes(registry, telemetry) {
		m.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return m
}

func doJSON(t *testing.T, m *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)
	return rec
}

func TestEndToEnd_AssetConnectionTelemetry(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	telemetry := digitaltwin.NewMemoryTelemetryRepository()
	m := mux(registry, telemetry)

	pipeRec := doJSON(t, m, "POST", "/assets", digitaltwin.Asset{
		Name: "Line 4", AssetType: digitaltwin.AssetTypePipeline,
	})
	if pipeRec.Code != http.StatusCreated {
		t.Fatalf("create pipeline: got %d, body %s", pipeRec.Code, pipeRec.Body.String())
	}
	var pipeline digitaltwin.Asset
	if err := json.Unmarshal(pipeRec.Body.Bytes(), &pipeline); err != nil {
		t.Fatalf("decode pipeline: %v", err)
	}
	if pipeline.ID == "" || pipeline.Status != digitaltwin.AssetStatusPlanned {
		t.Fatalf("unexpected pipeline: %+v", pipeline)
	}

	valveRec := doJSON(t, m, "POST", "/assets", digitaltwin.Asset{
		Name: "Valve 1", AssetType: digitaltwin.AssetTypeValve,
	})
	var valve digitaltwin.Asset
	if err := json.Unmarshal(valveRec.Body.Bytes(), &valve); err != nil {
		t.Fatalf("decode valve: %v", err)
	}

	connRec := doJSON(t, m, "POST", "/connections", digitaltwin.Connection{
		Kind: digitaltwin.ConnectionPartOf, FromAssetID: valve.ID, ToAssetID: pipeline.ID,
	})
	if connRec.Code != http.StatusCreated {
		t.Fatalf("create connection: got %d, body %s", connRec.Code, connRec.Body.String())
	}

	listConnRec := doJSON(t, m, "GET", "/assets/"+valve.ID+"/connections", nil)
	var conns []digitaltwin.Connection
	if err := json.Unmarshal(listConnRec.Body.Bytes(), &conns); err != nil {
		t.Fatalf("decode connections: %v", err)
	}
	if len(conns) != 1 || conns[0].ToAssetID != pipeline.ID {
		t.Fatalf("got connections %+v, want one to pipeline", conns)
	}

	statusRec := doJSON(t, m, "POST", "/assets/"+pipeline.ID+"/status", map[string]string{"status": string(digitaltwin.AssetStatusOperational)})
	if statusRec.Code != http.StatusOK {
		t.Fatalf("transition status: got %d, body %s", statusRec.Code, statusRec.Body.String())
	}

	readingRec := doJSON(t, m, "POST", "/telemetry/readings", digitaltwin.TelemetryReading{
		AssetID: pipeline.ID, AssetType: digitaltwin.AssetTypePipeline,
		MetricName: "pressure_psi", Value: 42.5, RecordedAt: fixedTime(),
	})
	if readingRec.Code != http.StatusCreated {
		t.Fatalf("record reading: got %d, body %s", readingRec.Code, readingRec.Body.String())
	}

	latestRec := doJSON(t, m, "GET", "/assets/"+pipeline.ID+"/telemetry/latest?metric_name=pressure_psi", nil)
	if latestRec.Code != http.StatusOK {
		t.Fatalf("get latest reading: got %d, body %s", latestRec.Code, latestRec.Body.String())
	}
	var latest digitaltwin.TelemetryReading
	if err := json.Unmarshal(latestRec.Body.Bytes(), &latest); err != nil {
		t.Fatalf("decode latest reading: %v", err)
	}
	if latest.Value != 42.5 {
		t.Fatalf("got value %v, want 42.5", latest.Value)
	}

	notFoundRec := doJSON(t, m, "GET", "/assets/does-not-exist", nil)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("get unknown asset: got %d, want %d", notFoundRec.Code, http.StatusNotFound)
	}
}
