package digitaltwin

import (
	"context"
	"errors"
	"testing"
	"time"
)

// RunTelemetryConformance exercises the rules that matter against any
// TelemetryRepository implementation — memory or the dedicated Postgres
// table (once W4 lands). Both backends must agree.
func RunTelemetryConformance(t *testing.T, newRepo func() TelemetryRepository) {
	t.Helper()
	ctx := context.Background()
	const asset = "pipeline-1"

	reading := func(metric string, value float64, at time.Time) TelemetryReading {
		return TelemetryReading{
			AssetID: asset, AssetType: AssetTypePipeline,
			MetricName: metric, Value: value, Unit: "psi", RecordedAt: at,
		}
	}

	t.Run("RecordSetsIDAndIngestedAt", func(t *testing.T) {
		r := newRepo()
		out, err := r.Record(ctx, reading("pressure_psi", 42.0, time.Now().UTC()))
		if err != nil {
			t.Fatalf("Record: %v", err)
		}
		if out.ID == "" || out.IngestedAt.IsZero() {
			t.Fatalf("Record did not fill in ID/IngestedAt: %+v", out)
		}
	})

	t.Run("RecordRejectsMissingRecordedAt", func(t *testing.T) {
		r := newRepo()
		_, err := r.Record(ctx, reading("pressure_psi", 42.0, time.Time{}))
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("want ErrInvalid for a zero recorded_at, got %v", err)
		}
	})

	t.Run("RecordBatchIsAllOrNothing", func(t *testing.T) {
		r := newRepo()
		now := time.Now().UTC()
		_, err := r.RecordBatch(ctx, []TelemetryReading{
			reading("pressure_psi", 40.0, now),
			reading("pressure_psi", 41.0, time.Time{}), // invalid: zero RecordedAt
		})
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("want ErrInvalid, got %v", err)
		}

		got, err := r.Query(ctx, TelemetryFilter{AssetID: asset})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("a failed batch must store nothing, got %d readings", len(got))
		}
	})

	t.Run("QueryFiltersByAssetMetricAndTimeRange", func(t *testing.T) {
		r := newRepo()
		base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		if _, err := r.RecordBatch(ctx, []TelemetryReading{
			reading("pressure_psi", 10, base),
			reading("pressure_psi", 20, base.Add(time.Hour)),
			reading("pressure_psi", 30, base.Add(2*time.Hour)),
			reading("flow_m3h", 5, base.Add(time.Hour)),
		}); err != nil {
			t.Fatal(err)
		}

		got, err := r.Query(ctx, TelemetryFilter{
			AssetID: asset, MetricName: "pressure_psi",
			From: base.Add(30 * time.Minute), To: base.Add(90 * time.Minute),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Value != 20 {
			t.Fatalf("want exactly the 20-value reading, got %+v", got)
		}
	})

	t.Run("LatestReturnsMostRecent", func(t *testing.T) {
		r := newRepo()
		base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		if _, err := r.RecordBatch(ctx, []TelemetryReading{
			reading("pressure_psi", 10, base),
			reading("pressure_psi", 20, base.Add(time.Hour)),
		}); err != nil {
			t.Fatal(err)
		}
		latest, err := r.Latest(ctx, asset, "pressure_psi")
		if err != nil {
			t.Fatal(err)
		}
		if latest.Value != 20 {
			t.Fatalf("Latest value = %v, want 20", latest.Value)
		}
	})

	t.Run("LatestReturnsErrReadingNotFoundWhenEmpty", func(t *testing.T) {
		r := newRepo()
		if _, err := r.Latest(ctx, asset, "pressure_psi"); !errors.Is(err, ErrReadingNotFound) {
			t.Fatalf("want ErrReadingNotFound, got %v", err)
		}
	})

	t.Run("LatestByAssetReturnsOnePerMetric", func(t *testing.T) {
		r := newRepo()
		base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		if _, err := r.RecordBatch(ctx, []TelemetryReading{
			reading("pressure_psi", 10, base),
			reading("pressure_psi", 20, base.Add(time.Hour)),
			reading("flow_m3h", 5, base.Add(time.Hour)),
		}); err != nil {
			t.Fatal(err)
		}
		latest, err := r.LatestByAsset(ctx, asset)
		if err != nil {
			t.Fatal(err)
		}
		if len(latest) != 2 {
			t.Fatalf("want one reading per metric (2), got %d: %+v", len(latest), latest)
		}
		for _, l := range latest {
			if l.MetricName == "pressure_psi" && l.Value != 20 {
				t.Fatalf("pressure_psi latest = %v, want 20", l.Value)
			}
		}
	})

	t.Run("PurgeBeforeDeletesOnlyMatchingOlderReadings", func(t *testing.T) {
		r := newRepo()
		base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		if _, err := r.RecordBatch(ctx, []TelemetryReading{
			reading("pressure_psi", 1, base),                       // old, matches
			reading("pressure_psi", 2, base.Add(400*24*time.Hour)), // recent, must survive
			reading("flow_m3h", 3, base),                           // old, different metric — must survive
		}); err != nil {
			t.Fatal(err)
		}

		cutoff := base.Add(24 * time.Hour)
		purged, err := r.PurgeBefore(ctx, AssetTypePipeline, "pressure_psi", cutoff)
		if err != nil {
			t.Fatal(err)
		}
		if purged != 1 {
			t.Fatalf("purged = %d, want 1", purged)
		}

		remaining, err := r.Query(ctx, TelemetryFilter{AssetID: asset})
		if err != nil {
			t.Fatal(err)
		}
		if len(remaining) != 2 {
			t.Fatalf("want 2 readings left (the recent pressure_psi + the old flow_m3h), got %d: %+v", len(remaining), remaining)
		}
	})
}
