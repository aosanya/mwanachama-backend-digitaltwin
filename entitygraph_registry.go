// entitygraph_registry.go — EntitygraphRegistryRepository (W4): the
// Postgres/entity-graph-backed RegistryRepository, mirroring
// mwanachama-backend-taskmanager's task_impl_task.go + relationship.go +
// agent.go patterns onto Asset/Connection/MetricDefinition/RetentionPolicy.
//
// Telemetry readings are NOT handled here — see telemetry_postgres.go.
package digitaltwin

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/entitygraph"
)

// EntitygraphRegistryRepository is the entity-graph-backed RegistryRepository.
// Construct with a Postgres-backed entitygraph.DataManager (see
// mwanachama-backend-shared/postgres.NewBackend) once DefaultDigitalTwinSchema
// has been set/published/activated — the same sequence
// mwanachama-backend-taskmanager's postgres_integration_test.go follows for
// DefaultWorkSchema.
// dataManager is entitygraph.DataManager plus the relationship methods this
// package needs — CreateRelationship/DeleteRelationship/ListRelationships
// are no longer part of the shared interface (see its doc comment), since
// each consumer knows its own fixed set of relationship labels.
type dataManager interface {
	entitygraph.DataManager
	CreateRelationship(ctx context.Context, req entitygraph.CreateRelationshipRequest) (entitygraph.Relationship, error)
	DeleteRelationship(ctx context.Context, relationshipID string) error
	ListRelationships(ctx context.Context, filter entitygraph.RelationshipFilter) ([]entitygraph.Relationship, error)
}

type EntitygraphRegistryRepository struct {
	dm dataManager
}

// NewEntitygraphRegistryRepository constructs a RegistryRepository backed by dm.
func NewEntitygraphRegistryRepository(dm dataManager) (*EntitygraphRegistryRepository, error) {
	if dm == nil {
		return nil, fmt.Errorf("NewEntitygraphRegistryRepository: data manager must not be nil")
	}
	return &EntitygraphRegistryRepository{dm: dm}, nil
}

// ── Asset ────────────────────────────────────────────────────────────────

// CreateAsset implements RegistryRepository.
func (r *EntitygraphRegistryRepository) CreateAsset(ctx context.Context, a Asset) (Asset, error) {
	if a.Status == "" {
		a.Status = AssetStatusPlanned
	}
	if err := a.Validate(); err != nil {
		return Asset{}, err
	}
	now := timestamp()
	a.CreatedAt = now
	a.UpdatedAt = now

	created, err := r.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		TypeID:     assetEntityTypeID,
		Properties: assetToProperties(a),
	})
	if err != nil {
		return Asset{}, fmt.Errorf("CreateAsset: %w", err)
	}
	return assetFromEntity(created), nil
}

// GetAsset implements RegistryRepository.
func (r *EntitygraphRegistryRepository) GetAsset(ctx context.Context, id string) (Asset, error) {
	e, err := r.dm.GetEntity(ctx, id)
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Asset{}, ErrAssetNotFound
		}
		return Asset{}, fmt.Errorf("GetAsset: %w", err)
	}
	if e.TypeID != assetEntityTypeID {
		return Asset{}, ErrAssetNotFound
	}
	return assetFromEntity(e), nil
}

// ListAssets implements RegistryRepository.
func (r *EntitygraphRegistryRepository) ListAssets(ctx context.Context, filter AssetFilter, limit int) ([]Asset, error) {
	props := map[string]any{}
	if filter.AssetType != "" {
		props["asset_type"] = string(filter.AssetType)
	}
	if filter.Status != "" {
		props["status"] = string(filter.Status)
	}
	entities, err := r.dm.ListEntities(ctx, entitygraph.EntityFilter{
		TypeID:     assetEntityTypeID,
		Properties: props,
	})
	if err != nil {
		return nil, fmt.Errorf("ListAssets: %w", err)
	}
	if limit <= 0 {
		limit = DefaultPage
	}
	if len(entities) > limit {
		entities = entities[:limit]
	}
	out := make([]Asset, 0, len(entities))
	for _, e := range entities {
		out = append(out, assetFromEntity(e))
	}
	return out, nil
}

// UpdateAsset implements RegistryRepository. AssetType/StationType/Status
// are immutable through this method.
func (r *EntitygraphRegistryRepository) UpdateAsset(ctx context.Context, a Asset) (Asset, error) {
	current, err := r.GetAsset(ctx, a.ID)
	if err != nil {
		return Asset{}, err
	}
	updated := current
	updated.Name = a.Name
	updated.Location = a.Location
	updated.InstalledAt = a.InstalledAt
	updated.Notes = a.Notes
	updated.Attributes = a.Attributes
	if err := updated.Validate(); err != nil {
		return Asset{}, err
	}
	updated.UpdatedAt = timestamp()

	saved, err := r.dm.UpdateEntity(ctx, a.ID, entitygraph.UpdateEntityRequest{
		Properties: assetToProperties(updated),
	})
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Asset{}, ErrAssetNotFound
		}
		return Asset{}, fmt.Errorf("UpdateAsset: %w", err)
	}
	return assetFromEntity(saved), nil
}

// UpdateAssetStatus implements RegistryRepository.
func (r *EntitygraphRegistryRepository) UpdateAssetStatus(ctx context.Context, id string, next AssetStatus) (Asset, error) {
	if !IsAssetStatus(next) {
		return Asset{}, ErrInvalid
	}
	current, err := r.GetAsset(ctx, id)
	if err != nil {
		return Asset{}, err
	}
	if !current.Status.CanTransitionTo(next) {
		return Asset{}, ErrInvalid
	}
	saved, err := r.dm.UpdateEntity(ctx, id, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"status":     string(next),
			"updated_at": timestamp(),
		},
	})
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Asset{}, ErrAssetNotFound
		}
		return Asset{}, fmt.Errorf("UpdateAssetStatus: %w", err)
	}
	return assetFromEntity(saved), nil
}

// ── Connection ───────────────────────────────────────────────────────────

// CreateConnection implements RegistryRepository.
func (r *EntitygraphRegistryRepository) CreateConnection(ctx context.Context, c Connection) (Connection, error) {
	if err := c.Validate(); err != nil {
		return Connection{}, err
	}

	from, err := r.dm.GetEntity(ctx, c.FromAssetID)
	if err != nil || from.TypeID != assetEntityTypeID {
		return Connection{}, ErrInvalidConnection
	}
	to, err := r.dm.GetEntity(ctx, c.ToAssetID)
	if err != nil || to.TypeID != assetEntityTypeID {
		return Connection{}, ErrInvalidConnection
	}

	existing, err := r.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		FromID: c.FromAssetID, ToID: c.ToAssetID, Name: string(c.Kind),
	})
	if err != nil {
		return Connection{}, fmt.Errorf("CreateConnection: list: %w", err)
	}
	if len(existing) > 0 {
		return connectionFromEntitygraph(existing[0]), nil
	}

	props := map[string]any{"created_at": timestamp()}
	for k, v := range c.Properties {
		props[k] = v
	}
	created, err := r.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		Name: string(c.Kind), FromID: c.FromAssetID, ToID: c.ToAssetID, Properties: props,
	})
	if err != nil {
		if errors.Is(err, entitygraph.ErrInvalidRelationship) || errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Connection{}, ErrInvalidConnection
		}
		return Connection{}, fmt.Errorf("CreateConnection: %w", err)
	}
	return connectionFromEntitygraph(created), nil
}

// DeleteConnection implements RegistryRepository.
func (r *EntitygraphRegistryRepository) DeleteConnection(ctx context.Context, fromAssetID, toAssetID string, kind ConnectionKind) error {
	edges, err := r.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		FromID: fromAssetID, ToID: toAssetID, Name: string(kind),
	})
	if err != nil {
		return fmt.Errorf("DeleteConnection: list: %w", err)
	}
	if len(edges) == 0 {
		return ErrConnectionNotFound
	}
	if err := r.dm.DeleteRelationship(ctx, edges[0].ID); err != nil {
		if errors.Is(err, entitygraph.ErrRelationshipNotFound) {
			return ErrConnectionNotFound
		}
		return fmt.Errorf("DeleteConnection: %w", err)
	}
	return nil
}

// ListConnections implements RegistryRepository.
func (r *EntitygraphRegistryRepository) ListConnections(ctx context.Context, assetID string, kind ConnectionKind, dir Direction) ([]Connection, error) {
	filter := entitygraph.RelationshipFilter{Name: string(kind)}
	if dir == DirectionInbound {
		filter.ToID = assetID
	} else {
		filter.FromID = assetID
	}
	edges, err := r.dm.ListRelationships(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("ListConnections: %w", err)
	}
	out := make([]Connection, 0, len(edges))
	for _, e := range edges {
		out = append(out, connectionFromEntitygraph(e))
	}
	return out, nil
}

// ── MetricDefinition ─────────────────────────────────────────────────────

// UpsertMetricDefinition implements RegistryRepository.
//
// UpsertEntity's merge writes exactly the properties supplied — so
// CreatedAt has to be resolved against any existing match first, or a
// repeat upsert would silently overwrite the original creation time on
// every call.
func (r *EntitygraphRegistryRepository) UpsertMetricDefinition(ctx context.Context, m MetricDefinition) (MetricDefinition, error) {
	if err := m.Validate(); err != nil {
		return MetricDefinition{}, err
	}
	now := timestamp()
	existing, err := r.dm.ListEntities(ctx, entitygraph.EntityFilter{
		TypeID:     metricDefinitionEntityTypeID,
		Properties: map[string]any{"asset_type": string(m.AssetType), "name": m.Name},
	})
	if err != nil {
		return MetricDefinition{}, fmt.Errorf("UpsertMetricDefinition: lookup: %w", err)
	}
	if len(existing) > 0 {
		m.CreatedAt = entitygraph.StringProp(existing[0].Properties, "created_at")
	} else {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	saved, err := r.dm.UpsertEntity(ctx, entitygraph.CreateEntityRequest{
		TypeID:     metricDefinitionEntityTypeID,
		Properties: metricDefinitionToProperties(m),
	})
	if err != nil {
		return MetricDefinition{}, fmt.Errorf("UpsertMetricDefinition: %w", err)
	}
	return metricDefinitionFromEntity(saved), nil
}

// ListMetricDefinitions implements RegistryRepository.
func (r *EntitygraphRegistryRepository) ListMetricDefinitions(ctx context.Context, assetType AssetType) ([]MetricDefinition, error) {
	props := map[string]any{}
	if assetType != "" {
		props["asset_type"] = string(assetType)
	}
	entities, err := r.dm.ListEntities(ctx, entitygraph.EntityFilter{
		TypeID: metricDefinitionEntityTypeID, Properties: props,
	})
	if err != nil {
		return nil, fmt.Errorf("ListMetricDefinitions: %w", err)
	}
	out := make([]MetricDefinition, 0, len(entities))
	for _, e := range entities {
		out = append(out, metricDefinitionFromEntity(e))
	}
	return out, nil
}

// ── RetentionPolicy ──────────────────────────────────────────────────────

// SetRetentionPolicy implements RegistryRepository. See
// UpsertMetricDefinition's doc comment for why CreatedAt must be resolved
// against an existing match before upserting.
func (r *EntitygraphRegistryRepository) SetRetentionPolicy(ctx context.Context, p RetentionPolicy) (RetentionPolicy, error) {
	if err := p.Validate(); err != nil {
		return RetentionPolicy{}, err
	}
	now := timestamp()
	existing, err := r.dm.ListEntities(ctx, entitygraph.EntityFilter{
		TypeID:     retentionPolicyEntityTypeID,
		Properties: map[string]any{"asset_type": string(p.AssetType), "metric_name": p.MetricName},
	})
	if err != nil {
		return RetentionPolicy{}, fmt.Errorf("SetRetentionPolicy: lookup: %w", err)
	}
	if len(existing) > 0 {
		p.CreatedAt = entitygraph.StringProp(existing[0].Properties, "created_at")
	} else {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	saved, err := r.dm.UpsertEntity(ctx, entitygraph.CreateEntityRequest{
		TypeID:     retentionPolicyEntityTypeID,
		Properties: retentionPolicyToProperties(p),
	})
	if err != nil {
		return RetentionPolicy{}, fmt.Errorf("SetRetentionPolicy: %w", err)
	}
	return retentionPolicyFromEntity(saved), nil
}

// ListRetentionPolicies implements RegistryRepository.
func (r *EntitygraphRegistryRepository) ListRetentionPolicies(ctx context.Context) ([]RetentionPolicy, error) {
	entities, err := r.dm.ListEntities(ctx, entitygraph.EntityFilter{
		TypeID: retentionPolicyEntityTypeID,
	})
	if err != nil {
		return nil, fmt.Errorf("ListRetentionPolicies: %w", err)
	}
	out := make([]RetentionPolicy, 0, len(entities))
	for _, e := range entities {
		out = append(out, retentionPolicyFromEntity(e))
	}
	return out, nil
}

var _ RegistryRepository = (*EntitygraphRegistryRepository)(nil)
