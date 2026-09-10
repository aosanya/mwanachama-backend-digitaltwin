package digitaltwin

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryTelemetryRepository is a reference TelemetryRepository, in-process
// and unpersisted. Stands in for the dedicated Postgres telemetry_readings
// table (W4 on the task board) — see telemetry.go's doc comment for why
// that table is separate from the registry's entity-graph store. Safe for
// concurrent use.
type MemoryTelemetryRepository struct {
	mu       sync.Mutex
	readings map[string]TelemetryReading
}

// NewMemoryTelemetryRepository returns an empty repository.
func NewMemoryTelemetryRepository() *MemoryTelemetryRepository {
	return &MemoryTelemetryRepository{readings: make(map[string]TelemetryReading)}
}

// Record implements TelemetryRepository.
func (r *MemoryTelemetryRepository) Record(_ context.Context, reading TelemetryReading) (TelemetryReading, error) {
	if err := reading.Validate(); err != nil {
		return TelemetryReading{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	reading.ID = uuid.NewString()
	reading.IngestedAt = time.Now().UTC()
	r.readings[reading.ID] = reading
	return reading, nil
}

// RecordBatch implements TelemetryRepository. All-or-nothing: every reading
// is validated before any is stored.
func (r *MemoryTelemetryRepository) RecordBatch(_ context.Context, readings []TelemetryReading) ([]TelemetryReading, error) {
	for _, reading := range readings {
		if err := reading.Validate(); err != nil {
			return nil, err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	out := make([]TelemetryReading, 0, len(readings))
	for _, reading := range readings {
		reading.ID = uuid.NewString()
		reading.IngestedAt = now
		r.readings[reading.ID] = reading
		out = append(out, reading)
	}
	return out, nil
}

// Query implements TelemetryRepository.
func (r *MemoryTelemetryRepository) Query(_ context.Context, filter TelemetryFilter) ([]TelemetryReading, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultPage
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]TelemetryReading, 0)
	for _, reading := range r.readings {
		if filter.AssetID != "" && reading.AssetID != filter.AssetID {
			continue
		}
		if filter.MetricName != "" && reading.MetricName != filter.MetricName {
			continue
		}
		if !filter.From.IsZero() && reading.RecordedAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && reading.RecordedAt.After(filter.To) {
			continue
		}
		out = append(out, reading)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RecordedAt.Before(out[j].RecordedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Latest implements TelemetryRepository.
func (r *MemoryTelemetryRepository) Latest(_ context.Context, assetID, metricName string) (TelemetryReading, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var latest TelemetryReading
	found := false
	for _, reading := range r.readings {
		if reading.AssetID != assetID || reading.MetricName != metricName {
			continue
		}
		if !found || reading.RecordedAt.After(latest.RecordedAt) {
			latest = reading
			found = true
		}
	}
	if !found {
		return TelemetryReading{}, ErrReadingNotFound
	}
	return latest, nil
}

// LatestByAsset implements TelemetryRepository.
func (r *MemoryTelemetryRepository) LatestByAsset(_ context.Context, assetID string) ([]TelemetryReading, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	latestByMetric := make(map[string]TelemetryReading)
	for _, reading := range r.readings {
		if reading.AssetID != assetID {
			continue
		}
		existing, ok := latestByMetric[reading.MetricName]
		if !ok || reading.RecordedAt.After(existing.RecordedAt) {
			latestByMetric[reading.MetricName] = reading
		}
	}
	out := make([]TelemetryReading, 0, len(latestByMetric))
	for _, reading := range latestByMetric {
		out = append(out, reading)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MetricName < out[j].MetricName })
	return out, nil
}

// PurgeBefore implements TelemetryRepository.
func (r *MemoryTelemetryRepository) PurgeBefore(_ context.Context, assetType AssetType, metricName string, cutoff time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	purged := 0
	for id, reading := range r.readings {
		if assetType != "" && reading.AssetType != assetType {
			continue
		}
		if metricName != "" && reading.MetricName != metricName {
			continue
		}
		if !reading.RecordedAt.Before(cutoff) {
			continue
		}
		delete(r.readings, id)
		purged++
	}
	return purged, nil
}
