package digitaltwin_test

import (
	"testing"
	"time"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
)

func TestSweepPurgesOnlyWhatARuleCovers(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -40).Format(time.RFC3339)
	recent := now.AddDate(0, 0, -5).Format(time.RFC3339)

	if _, err := tm.SetRetention(ctx, digitaltwin.Retention{
		NodeKind: "pipeline", MetricName: "pressure_psi", RetentionDays: 30,
	}); err != nil {
		t.Fatalf("rule: %v", err)
	}
	if _, err := tm.RecordObservations(ctx, []digitaltwin.Observation{
		{NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi", Value: 1, RecordedAt: old},
		{NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi", Value: 2, RecordedAt: recent},
		{NodeID: n.ID, NodeKind: "pipeline", MetricName: "flow_m3h", Value: 3, RecordedAt: old},
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	purged, err := digitaltwin.Sweep(ctx, tm, now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if purged != 1 {
		t.Fatalf("purged = %d, want 1 — only the old pressure_psi observation", purged)
	}

	left, err := tm.QueryObservations(ctx, digitaltwin.ObservationFilter{NodeID: n.ID})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(left) != 2 {
		t.Fatalf("left = %d, want 2 — the recent pressure and the old flow no rule covers", len(left))
	}
}

func TestW13_BroadRulePurgesWhatASpecificRuleShouldHaveProtected(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)

	if _, err := tm.SetRetention(ctx, digitaltwin.Retention{
		NodeKind: "pipeline", MetricName: "pressure_psi", RetentionDays: 365,
	}); err != nil {
		t.Fatalf("specific rule: %v", err)
	}
	if _, err := tm.SetRetention(ctx, digitaltwin.Retention{RetentionDays: 10}); err != nil {
		t.Fatalf("catch-all rule: %v", err)
	}

	if _, err := tm.RecordObservation(ctx, digitaltwin.Observation{
		NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi",
		Value: 42, RecordedAt: now.AddDate(0, 0, -40).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	if _, err := digitaltwin.Sweep(ctx, tm, now); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	left, err := tm.QueryObservations(ctx, digitaltwin.ObservationFilter{NodeID: n.ID})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("pin: the catch-all rule currently purges the observation regardless of the more specific rule; %d left instead, meaning W13 may already be fixed: %+v", len(left), left)
	}
}

func TestSweepWithNoRulesPurgesNothing(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")

	if _, err := tm.RecordObservation(ctx, digitaltwin.Observation{
		NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi",
		RecordedAt: "2000-01-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	purged, err := digitaltwin.Sweep(ctx, tm, time.Now())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if purged != 0 {
		t.Fatalf("purged = %d, want 0 — no rule means keep indefinitely", purged)
	}
}
