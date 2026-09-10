package digitaltwin

import (
	"fmt"
	"strings"
	"time"
)

// The asset registry — Asset, the vertex type for every physical thing this
// twin tracks (Pipeline, TransmissionLine, Valve, Station, Substation,
// Sensor). Decided by the 2026-09-03 dev-research session (DSN-1701,
// documentation/2. design/README.md) and architected like
// mwanachama-backend-taskmanager's Task: a concrete Go struct per registry
// concern, a hand-rolled status state machine ([AssetStatus.CanTransitionTo]
// in vocabulary.go), and no live telemetry on the struct itself — readings
// are a deliberately separate concern (telemetry.go).
//
// **Asset carries no fixed per-type schema.** Pipeline diameter, material,
// TransmissionLine voltage rating, and every other attribute specific to
// one AssetType were never specified in the research session — DSN-1701
// decision 6 established that reading/metric shape is "configurable once we
// have an expert, we only supply the tool", and the same reasoning is
// applied here rather than inventing field names with no grounding:
// Attributes carries whatever a deployment's expert configures, keyed by
// name. This is a real open question, not a permanent design stance — see
// documentation/2. design/README.md.

// Location is a physical position, optional on every Asset.
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Asset is one physical infrastructure item — a Pipeline, TransmissionLine,
// Valve, Station, Substation, or Sensor. All timestamps are RFC 3339
// strings, matching the entitygraph property storage convention used
// across mwanachama-backend-shared (once the registry is wired onto it —
// see W4 on the task board).
type Asset struct {
	// ID is the entity-graph storage key — opaque to callers.
	ID string `json:"id"`

	// Name is the short human-readable label (e.g. "Line 4 — Nakuru East").
	Name string `json:"name"`

	// AssetType is one of the six v1 types. Required, immutable after
	// creation.
	AssetType AssetType `json:"asset_type"`

	// StationType is set only when AssetType == AssetTypeStation
	// (compressor or pump); empty and rejected by Validate otherwise.
	StationType StationType `json:"station_type,omitempty"`

	// Status is the current lifecycle state. Always starts as
	// AssetStatusPlanned on creation; changed only via UpdateAssetStatus,
	// which enforces AssetStatus.CanTransitionTo.
	Status AssetStatus `json:"status"`

	// Location is the asset's physical position. Optional — nil when unset.
	Location *Location `json:"location,omitempty"`

	// InstalledAt is the RFC 3339 install/commission date; empty when
	// unknown.
	InstalledAt string `json:"installed_at,omitempty"`

	// Notes is free-form operator commentary.
	Notes string `json:"notes,omitempty"`

	// Attributes carries whatever static, type-specific properties a
	// deployment's expert has configured (diameter, material, voltage
	// rating, ...). Not validated against a fixed schema — see the file
	// doc comment.
	Attributes map[string]string `json:"attributes,omitempty"`

	// CreatedAt is the RFC 3339 timestamp the asset was first registered.
	CreatedAt string `json:"created_at"`

	// UpdatedAt is the RFC 3339 timestamp of the most recent mutation.
	UpdatedAt string `json:"updated_at"`
}

// AssetFilter constrains the results returned by
// [RegistryRepository.ListAssets]. Zero values mean "no filter" for that
// field — all values match.
type AssetFilter struct {
	// AssetType filters to the given type. Empty matches all.
	AssetType AssetType
	// Status filters to the given status. Empty matches all.
	Status AssetStatus
}

// Validate refuses a malformed asset. Does not check ID — that's the
// repository's concern (set on create, immutable after).
func (a Asset) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("%w: an asset with no name", ErrInvalid)
	}
	if !IsAssetType(a.AssetType) {
		return fmt.Errorf("%w: asset type %q is not one of the six v1 types", ErrInvalid, a.AssetType)
	}
	if a.AssetType == AssetTypeStation {
		if !IsStationType(a.StationType) {
			return fmt.Errorf("%w: a station asset must carry a valid station_type (compressor or pump)", ErrInvalid)
		}
	} else if a.StationType != "" {
		return fmt.Errorf("%w: station_type is only valid on a station asset, got asset_type %q", ErrInvalid, a.AssetType)
	}
	if a.Status != "" && !IsAssetStatus(a.Status) {
		return fmt.Errorf("%w: asset status %q is not one of the five lifecycle states", ErrInvalid, a.Status)
	}
	return nil
}

// timestamp is the shared RFC 3339 formatter used across this package's
// domain types, matching mwanachama-backend-shared's storage convention.
func timestamp() string { return time.Now().UTC().Format(time.RFC3339) }
