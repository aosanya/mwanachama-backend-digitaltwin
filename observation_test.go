package digitaltwin_test

import (
	"errors"
	"testing"
	"time"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
)

func TestUpdateLeavesTheKindAndTheStatusAlone(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")

	out, err := tm.UpdateNode(ctx, digitaltwin.Node{
		ID: n.ID, Name: "Line 4 renamed", Kind: "valve", Status: digitaltwin.StatusFault,
		Doc: map[string]any{"station_type": "compressor"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if out.Kind != "pipeline" {
		t.Fatalf("kind = %q, want it unchanged at pipeline", out.Kind)
	}
	if out.Status != digitaltwin.StatusPlanned {
		t.Fatalf("status = %q, want it unchanged at planned", out.Status)
	}
	if out.Name != "Line 4 renamed" || out.Doc["station_type"] != "compressor" {
		t.Fatalf("update dropped a value: %+v", out)
	}
}

func TestAnObservationRoundTripsAndIsStamped(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")

	o, err := tm.RecordObservation(ctx, digitaltwin.Observation{
		NodeID: n.ID, NodeKind: n.Kind, MetricName: "pressure_psi",
		Value: 41.5, Unit: "psi", RecordedAt: "2026-09-30T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if o.ID == "" || o.IngestedAt == "" {
		t.Fatalf("the store assigns an id and an ingested_at, got %+v", o)
	}

	got, err := tm.QueryObservations(ctx, digitaltwin.ObservationFilter{NodeID: n.ID})
	if err != nil || len(got) != 1 {
		t.Fatalf("query = %d rows, err %v", len(got), err)
	}
	if got[0].Value != 41.5 || got[0].Unit != "psi" {
		t.Fatalf("round trip lost the value: %+v", got[0])
	}
}

func TestLatestAnswersOneMetricOrEveryMetric(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")

	for _, o := range []digitaltwin.Observation{
		{NodeID: n.ID, MetricName: "pressure_psi", Value: 1, RecordedAt: "2026-09-30T10:00:00Z"},
		{NodeID: n.ID, MetricName: "pressure_psi", Value: 2, RecordedAt: "2026-09-30T11:00:00Z"},
		{NodeID: n.ID, MetricName: "flow_rate", Value: 9, RecordedAt: "2026-09-30T10:30:00Z"},
	} {
		if _, err := tm.RecordObservation(ctx, o); err != nil {
			t.Fatalf("record: %v", err)
		}
	}

	one, err := tm.Latest(ctx, n.ID, "pressure_psi")
	if err != nil {
		t.Fatalf("latest one: %v", err)
	}
	if got := one.(digitaltwin.Observation); got.Value != 2 {
		t.Fatalf("latest pressure = %v, want the 11:00 reading", got.Value)
	}

	all, err := tm.Latest(ctx, n.ID, "")
	if err != nil {
		t.Fatalf("latest all: %v", err)
	}
	if got := all.([]digitaltwin.Observation); len(got) != 2 {
		t.Fatalf("latest of each = %d, want 2", len(got))
	}
}

func TestRecordingIsAllOrNothing(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")

	_, err := tm.RecordObservations(ctx, []digitaltwin.Observation{
		{NodeID: n.ID, MetricName: "pressure_psi", Value: 1, RecordedAt: "2026-09-30T10:00:00Z"},
		{NodeID: n.ID, MetricName: "pressure_psi", Value: 2, RecordedAt: "not a time"},
	})
	if !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	got, err := tm.QueryObservations(ctx, digitaltwin.ObservationFilter{NodeID: n.ID})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a refused batch stored %d rows, want none", len(got))
	}
}

func TestAMetricUpsertReplacesRatherThanDuplicates(t *testing.T) {
	tm, ctx := newManager(t)

	first, err := tm.UpsertMetric(ctx, digitaltwin.Metric{Name: "pressure_psi", Unit: "psi", NodeKind: "pipeline"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	again, err := tm.UpsertMetric(ctx, digitaltwin.Metric{Name: "pressure_psi", Unit: "bar", NodeKind: "pipeline"})
	if err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	if first.ID != again.ID {
		t.Fatalf("upsert made a second row: %s then %s", first.ID, again.ID)
	}
	all, err := tm.ListMetrics(ctx, "")
	if err != nil || len(all) != 1 {
		t.Fatalf("list = %d rows, err %v", len(all), err)
	}
	if all[0].Unit != "bar" {
		t.Fatalf("unit = %q, want the replacement", all[0].Unit)
	}
}

func TestAnUndeclaredMetricIsRefusedOnlyWhenAskedFor(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")
	reading := digitaltwin.Observation{
		NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi",
		Value: 1, Unit: "psi", RecordedAt: "2026-09-30T10:00:00Z",
	}

	if _, err := tm.RecordObservation(ctx, reading); err != nil {
		t.Fatalf("the plain manager checks no vocabulary: %v", err)
	}

	strict := digitaltwin.NewValidatingTwinManager(tm)
	if _, err := strict.RecordObservation(ctx, reading); !errors.Is(err, digitaltwin.ErrUnknownMetric) {
		t.Fatalf("err = %v, want ErrUnknownMetric", err)
	}

	if _, err := tm.UpsertMetric(ctx, digitaltwin.Metric{Name: "pressure_psi", Unit: "psi", NodeKind: "pipeline"}); err != nil {
		t.Fatalf("declare: %v", err)
	}
	if _, err := strict.RecordObservation(ctx, reading); err != nil {
		t.Fatalf("a declared metric is accepted: %v", err)
	}
}

func TestARetentionRuleNamingAMetricMustNameAKind(t *testing.T) {
	tm, ctx := newManager(t)
	_, err := tm.SetRetention(ctx, digitaltwin.Retention{MetricName: "pressure_psi", RetentionDays: 10})
	if !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if _, err := tm.SetRetention(ctx, digitaltwin.Retention{RetentionDays: 0}); !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("zero days: err = %v, want ErrInvalid", err)
	}
}

func TestSweepPurgesWhatItsRuleCovers(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	old := now.AddDate(0, 0, -40).Format(time.RFC3339)
	fresh := now.AddDate(0, 0, -1).Format(time.RFC3339)
	for _, at := range []string{old, fresh} {
		if _, err := tm.RecordObservation(ctx, digitaltwin.Observation{
			NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi",
			Value: 1, RecordedAt: at,
		}); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	if _, err := tm.SetRetention(ctx, digitaltwin.Retention{
		NodeKind: "pipeline", MetricName: "pressure_psi", RetentionDays: 10,
	}); err != nil {
		t.Fatalf("rule: %v", err)
	}

	purged, err := digitaltwin.Sweep(ctx, tm, now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if purged != 1 {
		t.Fatalf("purged %d, want 1", purged)
	}
	left, err := tm.QueryObservations(ctx, digitaltwin.ObservationFilter{NodeID: n.ID})
	if err != nil || len(left) != 1 {
		t.Fatalf("left = %d rows, err %v", len(left), err)
	}
}
