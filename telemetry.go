package digitaltwin

import (
	"fmt"
	"strings"
	"time"
)

// TelemetryReading and RetentionPolicy — DSN-1701 decisions 2/3/6. A
// reading is deliberately NOT a registry vertex: mwanachama-backend-shared's
// entity-graph store is a single Postgres `entities` table shared across
// every service in this family, and routing "very high frequency"
// API-pushed readings through it would make it the hottest table in the
// platform for every unrelated service. TelemetryRepository (separate
// interface, telemetry_repository.go) is the boundary that keeps readings
// out of that table — a dedicated, service-owned store instead.
//
// The reading shape is generic (asset_id/asset_type/metric_name/value/unit)
// rather than fixed columns per asset type, because DSN-1701 decision 6
// established the metric vocabulary itself is expert-configured, not
// hardcoded here — see MetricDefinition (metric.go).

// TelemetryReading is one sample: one metric, one asset, one point in time.
type TelemetryReading struct {
	// ID is the storage-assigned reading identifier.
	ID string `json:"id"`

	// AssetID is the Asset this reading was measured against. Required —
	// DSN-1701 decision 6 confirmed a reading is not required to route
	// through a Sensor asset; it may name any asset directly.
	AssetID string `json:"asset_id"`

	// AssetType denormalises the target asset's type onto the reading so
	// queries can filter by type without a registry join on every read.
	AssetType AssetType `json:"asset_type"`

	// MetricName identifies what was measured (e.g. "pressure_psi").
	// Expected to match a registered MetricDefinition.Name, though this is
	// not enforced at ingest time — see metric.go's doc comment.
	MetricName string `json:"metric_name"`

	// Value is the sampled value.
	Value float64 `json:"value"`

	// Unit is the unit of measure. Carried on the reading itself (rather
	// than looked up from MetricDefinition on every read) so a reading
	// remains self-describing even if a definition is later edited or
	// removed.
	Unit string `json:"unit,omitempty"`

	// RecordedAt is when the physical measurement was taken, per the
	// source system. This is the field readings should be ordered and
	// queried on.
	RecordedAt time.Time `json:"recorded_at"`

	// IngestedAt is when this service received the reading. Distinct from
	// RecordedAt because high-frequency push sources are not guaranteed to
	// deliver in order or without delay.
	IngestedAt time.Time `json:"ingested_at"`

	// Quality is optional, free-form provenance/confidence commentary on
	// the sample (e.g. "good", "estimated", "stale") — not a validated
	// vocabulary; a real quality-code scheme is an open question, deferred
	// to whoever needs it first.
	Quality string `json:"quality,omitempty"`
}

// Validate refuses a malformed reading. Does not check ID/IngestedAt —
// those are the repository's concern (set on ingest).
func (r TelemetryReading) Validate() error {
	if strings.TrimSpace(r.AssetID) == "" {
		return fmt.Errorf("%w: a reading with no asset_id", ErrInvalid)
	}
	if !IsAssetType(r.AssetType) {
		return fmt.Errorf("%w: reading asset_type %q is not one of the six v1 types", ErrInvalid, r.AssetType)
	}
	if strings.TrimSpace(r.MetricName) == "" {
		return fmt.Errorf("%w: a reading with no metric_name", ErrInvalid)
	}
	if r.RecordedAt.IsZero() {
		return fmt.Errorf("%w: reading for metric %q has a zero recorded_at", ErrInvalid, r.MetricName)
	}
	return nil
}

// TelemetryFilter constrains the results returned by
// [TelemetryRepository.Query]. Zero values mean "no filter" for that
// field — all values match. From/To bound RecordedAt; either may be the
// zero time to leave that side unbounded.
type TelemetryFilter struct {
	AssetID    string
	MetricName string
	From       time.Time
	To         time.Time
	Limit      int
}

// RetentionPolicy is customer-defined retention, scoped by AssetType and/or
// MetricName — DSN-1701 flagged retention as "customer defined" with no
// default period or mechanism supplied in the research session. This type
// ships the configuration vocabulary ahead of enforcement, the same
// precedent mwanachama-backend-accounting's DocumentKind set: a value a
// caller can store and read back today, with the sweep/deletion job that
// actually enforces it left as open work (see
// documentation/3. implementation/todo.md). Reading this policy back and
// acting on it is the caller's responsibility until that job exists.
type RetentionPolicy struct {
	// ID is the storage-assigned identifier.
	ID string `json:"id"`

	// AssetType scopes this policy to one asset type. Empty means it
	// applies to every asset type not covered by a more specific policy.
	AssetType AssetType `json:"asset_type,omitempty"`

	// MetricName scopes this policy to one metric. Empty means it applies
	// to every metric of AssetType.
	MetricName string `json:"metric_name,omitempty"`

	// RetentionDays is how long a matching reading is kept. Must be
	// positive — "keep forever" is expressed by having no policy at all,
	// not by a zero/negative value, so an unset policy and an explicit
	// "forever" are never confused.
	RetentionDays int `json:"retention_days"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Validate refuses a malformed retention policy.
func (p RetentionPolicy) Validate() error {
	if p.AssetType != "" && !IsAssetType(p.AssetType) {
		return fmt.Errorf("%w: retention policy asset_type %q is not one of the six v1 types", ErrInvalid, p.AssetType)
	}
	if p.MetricName != "" && p.AssetType == "" {
		return fmt.Errorf("%w: a retention policy scoped to a metric_name must also name an asset_type", ErrInvalid)
	}
	if p.RetentionDays <= 0 {
		return fmt.Errorf("%w: retention_days must be positive (%d given); keep-forever is expressed by having no policy", ErrInvalid, p.RetentionDays)
	}
	return nil
}
