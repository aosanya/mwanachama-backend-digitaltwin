package digitaltwin

import (
	"context"
	"time"
)

// TelemetryRepository is the persistence boundary for TelemetryReading —
// deliberately separate from RegistryRepository (DSN-1701 decision 3).
// **There is no Update and no Delete.** A reading is what a sensor said at a
// point in time; a wrong reading is corrected by recording a fresh one, not
// by editing history — the same append-only discipline
// mwanachama-backend-accounting's LedgerRepository documents for its own
// entries, applied here because a digital twin's historical record should
// be as trustworthy as a ledger's.
type TelemetryRepository interface {
	// Record validates and stores one reading. ID/IngestedAt are set by the
	// repository.
	Record(ctx context.Context, r TelemetryReading) (TelemetryReading, error)

	// RecordBatch stores multiple readings in one call — the expected path
	// for a high-frequency source, so it isn't paying one round trip per
	// sample. All-or-nothing: if any reading fails Validate, none are
	// stored.
	RecordBatch(ctx context.Context, readings []TelemetryReading) ([]TelemetryReading, error)

	// Query returns readings matching filter, ordered oldest RecordedAt
	// first. filter.Limit <= 0 means DefaultPage.
	Query(ctx context.Context, filter TelemetryFilter) ([]TelemetryReading, error)

	// Latest returns the most recent reading for one (assetID, metricName)
	// pair — the "what is this asset doing right now" read a digital twin
	// exists to answer. ErrReadingNotFound if nothing has been recorded yet.
	Latest(ctx context.Context, assetID, metricName string) (TelemetryReading, error)

	// LatestByAsset returns the most recent reading for every metric
	// recorded against one asset — the full current-state snapshot of that
	// asset. Empty slice, no error, if nothing has been recorded yet.
	LatestByAsset(ctx context.Context, assetID string) ([]TelemetryReading, error)

	// PurgeBefore deletes every reading matching (assetType, metricName)
	// with RecordedAt strictly before cutoff, and returns the count
	// removed. assetType/metricName empty means "any" — the scoping
	// RetentionPolicy itself uses (empty AssetType = every asset type;
	// empty MetricName = every metric of that asset type). This is the
	// mechanism W6's retention sweep calls; nothing in this package invokes
	// it on a schedule.
	PurgeBefore(ctx context.Context, assetType AssetType, metricName string, cutoff time.Time) (int, error)
}
