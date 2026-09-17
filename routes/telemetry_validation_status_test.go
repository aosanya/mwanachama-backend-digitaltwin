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

// TestValidatingTelemetry_UnknownMetricRejectionSurfacesAs500 pins board row
// W12: routes/errors.go's digitaltwinStatusFor switch enumerates every
// sentinel error in errors.go, but ValidatingTelemetryRepository's
// ErrUnknownMetric (telemetry_validating.go) lives outside that file and was
// never added to the switch. The validation logic itself is correct — an
// unregistered metric name, a unit mismatch, or a reading for the wrong
// asset type are all genuinely refused — but because ErrUnknownMetric
// matches none of digitaltwinStatusFor's errors.Is cases, the refusal falls
// through to the default arm and comes back as a 500 "internal error"
// instead of a 400 naming the real problem, exactly like every other
// caller-input validation failure in this package does.
//
// This test asserts TODAY'S (broken) behavior — 500, body "internal
// error" — so it pins the defect: once W12 is fixed (ErrUnknownMetric added
// to digitaltwinStatusFor's ErrInvalid arm, or wrapped in ErrInvalid at the
// source), this assertion must be updated to expect 400 with a body naming
// the unmatched metric, and this test will go red as the signal that it
// needs updating.
func TestValidatingTelemetry_UnknownMetricRejectionSurfacesAs500(t *testing.T) {
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

	// PINS THE BUG: today this is 500 "internal error", not 400 naming the
	// unmatched metric. A fix for W12 makes this assertion wrong on
	// purpose — see this test's doc comment.
	if code != http.StatusInternalServerError {
		t.Fatalf("pinned-defect assertion failed (bug may be fixed — update this test per W12): got status %d, want %d (500)", code, http.StatusInternalServerError)
	}
	if body != `{"error":"internal error"}`+"\n" {
		t.Fatalf("pinned-defect assertion failed (bug may be fixed — update this test per W12): got body %q", body)
	}

	// Same gap on the batch path.
	code, body = post(t, "/telemetry/readings/batch", []digitaltwin.TelemetryReading{
		{AssetID: pipeline.ID, AssetType: digitaltwin.AssetTypePipeline, MetricName: "pressure_psi", Value: 1, Unit: "psi", RecordedAt: recordedAt},
		{AssetID: pipeline.ID, AssetType: digitaltwin.AssetTypePipeline, MetricName: "another_unregistered_metric", Value: 1, Unit: "x", RecordedAt: recordedAt},
	})
	if code != http.StatusInternalServerError {
		t.Fatalf("pinned-defect assertion failed on batch path (bug may be fixed — update this test per W12): got status %d, want %d (500)", code, http.StatusInternalServerError)
	}
}
