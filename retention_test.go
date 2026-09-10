package digitaltwin

import (
	"context"
	"testing"
	"time"
)

func TestSweepPurgesOnlyWhatAPolicyCovers(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	telemetry := NewMemoryTelemetryRepository()

	// A policy scoped to pipeline/pressure_psi, 30-day retention.
	if _, err := registry.SetRetentionPolicy(ctx, RetentionPolicy{
		AssetType: AssetTypePipeline, MetricName: "pressure_psi", RetentionDays: 30,
	}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -40)   // older than 30 days — covered, purged
	recent := now.AddDate(0, 0, -5) // within 30 days — survives
	base := TelemetryReading{AssetID: "pipeline-1", AssetType: AssetTypePipeline, Unit: "psi"}

	if _, err := telemetry.RecordBatch(ctx, []TelemetryReading{
		{AssetID: base.AssetID, AssetType: base.AssetType, MetricName: "pressure_psi", Value: 1, RecordedAt: old},
		{AssetID: base.AssetID, AssetType: base.AssetType, MetricName: "pressure_psi", Value: 2, RecordedAt: recent},
		// Different metric, same age as the purged one — no policy covers
		// it, so it must survive even though it's just as old.
		{AssetID: base.AssetID, AssetType: base.AssetType, MetricName: "flow_m3h", Value: 3, RecordedAt: old},
	}); err != nil {
		t.Fatal(err)
	}

	purged, err := Sweep(ctx, registry, telemetry, now)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if purged != 1 {
		t.Fatalf("purged = %d, want 1 (only the old pressure_psi reading)", purged)
	}

	remaining, err := telemetry.Query(ctx, TelemetryFilter{AssetID: base.AssetID})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Fatalf("want 2 readings left (recent pressure_psi + old flow_m3h, uncovered by any policy), got %d: %+v", len(remaining), remaining)
	}
}

func TestSweepWithNoPoliciesPurgesNothing(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	telemetry := NewMemoryTelemetryRepository()

	if _, err := telemetry.Record(ctx, TelemetryReading{
		AssetID: "pipeline-1", AssetType: AssetTypePipeline,
		MetricName: "pressure_psi", RecordedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	purged, err := Sweep(ctx, registry, telemetry, time.Now())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if purged != 0 {
		t.Fatalf("purged = %d, want 0 — no policy means keep forever", purged)
	}
}
