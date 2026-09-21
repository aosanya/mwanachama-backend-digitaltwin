package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/routes"
)

// TestValidatingTelemetry_UnknownMetricRejectionSurfacesAs400 pins the W12
// fix: ValidatingTelemetryRepository's ErrUnknownMetric maps to 400 with a
// body naming the rejected metric, on both the single and batch paths,
// instead of falling through digitaltwinStatusFor to an opaque 500.
func TestValidatingTelemetry_UnknownMetricRejectionSurfacesAs400(t *testing.T) {
	registry := digitaltwin.NewMemoryRegistryRepository()
	rawTelemetry := digitaltwin.NewMemoryTelemetryRepository()
	validating := digitaltwin.NewValidatingTelemetryRepository(rawTelemetry, registry)

	mux := http.NewServeMux()
	for _, rt := range routes.Routes(registry, validating) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	post := func(t *testing.T, path string, body any) (int, string) {
		t.Helper()
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
		resp, err := http.DefaultClient.Post(srv.URL+path, "application/json", &buf)
		if err != nil {
			t.Fatalf("post %s: %v", path, err)
		}
		defer resp.Body.Close()
		out := new(bytes.Buffer)
		_, _ = out.ReadFrom(resp.Body)
		return resp.StatusCode, out.String()
	}

	// Create a pipeline asset and register a MetricDefinition for it, so
	// there is a real registry to validate against.
	var pipeline digitaltwin.Asset
	code, body := post(t, "/assets", digitaltwin.Asset{Name: "Line 1", AssetType: digitaltwin.AssetTypePipeline})
	if code != http.StatusCreated {
		t.Fatalf("create asset: got %d, body %s", code, body)
	}
	if err := json.Unmarshal([]byte(body), &pipeline); err != nil {
		t.Fatalf("decode asset: %v", err)
	}

	code, body = post(t, "/metric-definitions", digitaltwin.MetricDefinition{
		Name: "pressure_psi", Unit: "psi", AssetType: digitaltwin.AssetTypePipeline,
	})
	if code != http.StatusOK {
		t.Fatalf("register metric definition: got %d, body %s", code, body)
	}

	recordedAt := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	// A reading for a metric name that was never registered — a genuine
	// caller mistake, the class of error every other validation failure in
	// this package answers with 400.
	code, body = post(t, "/telemetry/readings", digitaltwin.TelemetryReading{
		AssetID: pipeline.ID, AssetType: digitaltwin.AssetTypePipeline,
		MetricName: "totally_unregistered_metric", Value: 1, Unit: "widgets", RecordedAt: recordedAt,
	})

	if code != http.StatusBadRequest {
		t.Fatalf("unregistered metric: got status %d (body %s), want 400", code, body)
	}
	if !strings.Contains(body, "totally_unregistered_metric") {
		t.Fatalf("body should name the rejected metric, got %q", body)
	}

	// Batch path.
	code, body = post(t, "/telemetry/readings/batch", []digitaltwin.TelemetryReading{
		{AssetID: pipeline.ID, AssetType: digitaltwin.AssetTypePipeline, MetricName: "pressure_psi", Value: 1, Unit: "psi", RecordedAt: recordedAt},
		{AssetID: pipeline.ID, AssetType: digitaltwin.AssetTypePipeline, MetricName: "another_unregistered_metric", Value: 1, Unit: "x", RecordedAt: recordedAt},
	})
	if code != http.StatusBadRequest {
		t.Fatalf("batch with an unregistered metric: got status %d (body %s), want 400", code, body)
	}
	if !strings.Contains(body, "another_unregistered_metric") {
		t.Fatalf("batch body should name the rejected metric, got %q", body)
	}
}
