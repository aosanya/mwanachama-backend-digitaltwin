package digitaltwin

import "context"

// DefaultPage is the page size a store falls back to when a caller passes
// limit <= 0.
const DefaultPage = 200

// RegistryRepository is the persistence boundary for the asset registry —
// Asset, Connection (topology), and MetricDefinition. Telemetry readings
// are deliberately NOT part of this interface — see TelemetryRepository
// (telemetry_repository.go) and the file doc comment on telemetry.go for
// why they're kept out of the same store.
//
// Modeled on mwanachama-backend-taskmanager's TaskManager: status changes
// go through a dedicated method that enforces
// AssetStatus.CanTransitionTo, separate from general attribute updates, the
// same separation Task keeps between UpdateTask and its status machine.
type RegistryRepository interface {
	// CreateAsset validates a and creates it. Status defaults to
	// AssetStatusPlanned when a.Status is empty. ID/CreatedAt/UpdatedAt are
	// set by the repository — callers should leave them empty.
	CreateAsset(ctx context.Context, a Asset) (Asset, error)

	// GetAsset returns one asset by id. ErrAssetNotFound if unknown.
	GetAsset(ctx context.Context, id string) (Asset, error)

	// ListAssets returns assets matching filter, oldest-created first.
	// limit <= 0 means DefaultPage.
	ListAssets(ctx context.Context, filter AssetFilter, limit int) ([]Asset, error)

	// UpdateAsset overwrites Name/Location/InstalledAt/Notes/Attributes on
	// an existing asset. AssetType, StationType, and Status are immutable
	// through this method — AssetType/StationType never change after
	// creation, and Status changes only through UpdateAssetStatus.
	// ErrAssetNotFound if id is unknown.
	UpdateAsset(ctx context.Context, a Asset) (Asset, error)

	// UpdateAssetStatus transitions the asset's status, enforcing
	// AssetStatus.CanTransitionTo. ErrAssetNotFound if id is unknown;
	// ErrInvalid if the transition is not allowed.
	UpdateAssetStatus(ctx context.Context, id string, next AssetStatus) (Asset, error)

	// CreateConnection validates the (Kind, FromAssetID, ToAssetID)
	// triple — both endpoints must resolve to known assets — and creates
	// the edge. Re-creating an existing edge is idempotent: the existing
	// edge is returned with no error, the same rule
	// taskmanager.CreateRelationship uses.
	CreateConnection(ctx context.Context, c Connection) (Connection, error)

	// DeleteConnection removes the single edge identified by
	// (fromAssetID, toAssetID, kind). ErrConnectionNotFound if no such edge
	// exists.
	DeleteConnection(ctx context.Context, fromAssetID, toAssetID string, kind ConnectionKind) error

	// ListConnections returns the single-hop edges incident on assetID
	// matching kind and direction. Empty kind matches every kind.
	ListConnections(ctx context.Context, assetID string, kind ConnectionKind, dir Direction) ([]Connection, error)

	// UpsertMetricDefinition creates or updates the definition identified by
	// (AssetType, Name) — find-or-update semantics, the registry analogue
	// of taskmanager.UpsertAgent.
	UpsertMetricDefinition(ctx context.Context, m MetricDefinition) (MetricDefinition, error)

	// ListMetricDefinitions returns every definition, optionally narrowed
	// to one asset type (empty assetType matches all).
	ListMetricDefinitions(ctx context.Context, assetType AssetType) ([]MetricDefinition, error)

	// SetRetentionPolicy creates or updates the policy identified by
	// (AssetType, MetricName). Config only — see RetentionPolicy's doc
	// comment for what this does and does not enforce.
	SetRetentionPolicy(ctx context.Context, p RetentionPolicy) (RetentionPolicy, error)

	// ListRetentionPolicies returns every policy.
	ListRetentionPolicies(ctx context.Context) ([]RetentionPolicy, error)
}
