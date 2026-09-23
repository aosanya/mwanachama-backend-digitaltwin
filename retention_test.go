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

// TestW13_BroadPolicyPurgesWhatASpecificPolicyShouldHaveProtected pins a
// known defect (board row W13): RetentionPolicy.AssetType's own doc comment
// says an empty AssetType "applies to every asset type not covered by a
// more specific policy," but Sweep/PurgeBefore implement no such
// precedence — each policy is applied independently, so a broad/catch-all
// policy (empty AssetType, or empty MetricName) purges readings a more
// specific, longer-retention policy exists to protect. This test must go
// RED once the fix lands (Sweep must skip a reading, under a broad policy's
// cutoff, that a more specific policy also covers and would not yet purge).
func TestW13_BroadPolicyPurgesWhatASpecificPolicyShouldHaveProtected(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	telemetry := NewMemoryTelemetryRepository()

	// A specific policy: keep pipeline pressure_psi readings for a year.
	if _, err := registry.SetRetentionPolicy(ctx, RetentionPolicy{
		AssetType: AssetTypePipeline, MetricName: "pressure_psi", RetentionDays: 365,
	}); err != nil {
		t.Fatal(err)
	}
	// A catch-all policy: everything not covered by a more specific policy
	// gets only 10 days — per RetentionPolicy.AssetType's own doc comment,
	// this should NOT apply to pipeline/pressure_psi, since the policy
	// above is more specific.
	if _, err := registry.SetRetentionPolicy(ctx, RetentionPolicy{
		AssetType: "", MetricName: "", RetentionDays: 10,
	}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	// 40 days old: past the catch-all's 10 days, well within the specific
	// policy's 365 days — the doc comment says this reading must survive.
	old := now.AddDate(0, 0, -40)

	if _, err := telemetry.Record(ctx, TelemetryReading{
		AssetID: "pipeline-1", AssetType: AssetTypePipeline,
		MetricName: "pressure_psi", Value: 42, RecordedAt: old,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := Sweep(ctx, registry, telemetry, now); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	remaining, err := telemetry.Query(ctx, TelemetryFilter{AssetID: "pipeline-1"})
	if err != nil {
		t.Fatal(err)
	}

	// KNOWN DEFECT (W13): the catch-all policy purges the reading anyway,
	// even though a more specific policy names a 365-day retention for it.
	if len(remaining) != 0 {
		t.Fatalf("pin: current (buggy) behaviour purges the reading regardless of the more specific policy; got %d remaining instead, meaning W13 may already be fixed: %+v", len(remaining), remaining)
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
