package digitaltwin

import (
	"fmt"
	"strings"
)

// MetricDefinition is the registry-side half of DSN-1701 decision 6:
// reading/metric shape is "configurable once we have an expert, we only
// supply the tool" — so the set of metric names, units, and (optionally)
// sane-value bounds a deployment cares about is data, not a hardcoded Go
// type per asset type. TelemetryReading.MetricName / .Unit are expected to
// name a MetricDefinition registered here, though this package does not
// enforce that at ingest time (open question — see
// documentation/2. design/README.md; enforcing it means every high-frequency
// write pays a registry lookup, which needs to be measured before it's
// required).
type MetricDefinition struct {
	// ID is the entity-graph storage key — opaque to callers.
	ID string `json:"id"`

	// Name is the metric name a TelemetryReading.MetricName is expected to
	// match (e.g. "pressure_psi", "voltage_kv"). Required, unique per
	// AssetType.
	Name string `json:"name"`

	// Unit is the unit of measure (e.g. "psi", "kV", "m3/h"). Required.
	Unit string `json:"unit"`

	// AssetType scopes this definition to one asset type. Empty means it
	// applies to any asset type.
	AssetType AssetType `json:"asset_type,omitempty"`

	// Description is free-form context for what this metric means and how
	// it's measured.
	Description string `json:"description,omitempty"`

	// MinValue / MaxValue are optional sane-value bounds, expert-configured.
	// Nil means "no bound". Not enforced by this package — a future
	// alerting/threshold mechanism (not yet designed) is the natural
	// consumer.
	MinValue *float64 `json:"min_value,omitempty"`
	MaxValue *float64 `json:"max_value,omitempty"`

	// CreatedAt is the RFC 3339 timestamp this definition was first created.
	CreatedAt string `json:"created_at"`
	// UpdatedAt is the RFC 3339 timestamp of the most recent mutation.
	UpdatedAt string `json:"updated_at"`
}

// Validate refuses a malformed metric definition.
func (m MetricDefinition) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: a metric definition with no name", ErrInvalid)
	}
	if strings.TrimSpace(m.Unit) == "" {
		return fmt.Errorf("%w: metric %q has no unit", ErrInvalid, m.Name)
	}
	if m.AssetType != "" && !IsAssetType(m.AssetType) {
		return fmt.Errorf("%w: metric %q scopes to asset type %q, which is not one of the six v1 types", ErrInvalid, m.Name, m.AssetType)
	}
	if m.MinValue != nil && m.MaxValue != nil && *m.MinValue > *m.MaxValue {
		return fmt.Errorf("%w: metric %q has min_value %v greater than max_value %v", ErrInvalid, m.Name, *m.MinValue, *m.MaxValue)
	}
	return nil
}
