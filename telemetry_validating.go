// telemetry_validating.go — ValidatingTelemetryRepository (W8): resolves
// open question 4 in documentation/2. design/README.md — "should ingest
// reject a MetricName/Unit that doesn't match a registered
// MetricDefinition?" — as an opt-in decorator rather than baking a registry
// lookup into every write.
//
// The default path (plain TelemetryRepository, no decorator) stays cheap:
// a high-frequency write pays no registry round trip. A caller who wants
// enforcement wraps their TelemetryRepository with this type; nothing
// changes for a caller who doesn't.
package digitaltwin

import (
	"context"
	"fmt"
)

// ErrUnknownMetric is returned by ValidatingTelemetryRepository when a
// reading's (AssetType, MetricName) doesn't match any registered
// MetricDefinition — neither one scoped to that AssetType nor one scoped
// to "any type" (MetricDefinition.AssetType == "").
var ErrUnknownMetric = fmt.Errorf("digitaltwin: metric not registered")

// ValidatingTelemetryRepository wraps a TelemetryRepository and rejects
// Record/RecordBatch calls whose (AssetType, MetricName) — and Unit, when
// the reading sets one — don't match a MetricDefinition registered in
// Registry. Every other method delegates straight through via the embedded
// interface.
type ValidatingTelemetryRepository struct {
	TelemetryRepository
	Registry RegistryRepository
}

// NewValidatingTelemetryRepository wraps telemetry with metric validation
// backed by registry.
func NewValidatingTelemetryRepository(telemetry TelemetryRepository, registry RegistryRepository) *ValidatingTelemetryRepository {
	return &ValidatingTelemetryRepository{TelemetryRepository: telemetry, Registry: registry}
}

// matchesDefinition reports whether a registered definition (scoped to
// reading.AssetType or scoped to any type) names the same metric — and, if
// the reading carries a Unit, that the units agree.
func matchesDefinition(defs []MetricDefinition, reading TelemetryReading) bool {
	for _, d := range defs {
		if d.Name != reading.MetricName {
			continue
		}
		if d.AssetType != "" && d.AssetType != reading.AssetType {
			continue
		}
		if reading.Unit != "" && d.Unit != reading.Unit {
			continue
		}
		return true
	}
	return false
}

// validate checks reading against every registered MetricDefinition (both
// type-scoped and any-type definitions).
func (v *ValidatingTelemetryRepository) validate(ctx context.Context, reading TelemetryReading) error {
	defs, err := v.Registry.ListMetricDefinitions(ctx, "")
	if err != nil {
		return fmt.Errorf("ValidatingTelemetryRepository: ListMetricDefinitions: %w", err)
	}
	if !matchesDefinition(defs, reading) {
		return fmt.Errorf("%w: %s asset_type=%s unit=%s", ErrUnknownMetric, reading.MetricName, reading.AssetType, reading.Unit)
	}
	return nil
}

// Record implements TelemetryRepository, validating before delegating.
func (v *ValidatingTelemetryRepository) Record(ctx context.Context, reading TelemetryReading) (TelemetryReading, error) {
	if err := v.validate(ctx, reading); err != nil {
		return TelemetryReading{}, err
	}
	return v.TelemetryRepository.Record(ctx, reading)
}

// RecordBatch implements TelemetryRepository. All-or-nothing, matching the
// wrapped repository's own contract: every reading is validated before any
// is delegated.
func (v *ValidatingTelemetryRepository) RecordBatch(ctx context.Context, readings []TelemetryReading) ([]TelemetryReading, error) {
	for _, reading := range readings {
		if err := v.validate(ctx, reading); err != nil {
			return nil, err
		}
	}
	return v.TelemetryRepository.RecordBatch(ctx, readings)
}

var _ TelemetryRepository = (*ValidatingTelemetryRepository)(nil)
