package digitaltwin

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"
)

// MemoryRegistryRepository is a reference RegistryRepository, in-process and
// unpersisted — the same role mwanachama-backend-accounting's
// MemoryRepository and mwanachama-backend-taskmanager's fakeDataManager
// play: a hand-rolled fake standing in for the Postgres/entity-graph
// backend (W4 on the task board) so the registry's rules can be built and
// tested before storage wiring lands. Safe for concurrent use.
type MemoryRegistryRepository struct {
	mu          sync.Mutex
	assets      map[string]Asset
	connections map[string]Connection
	metrics     map[string]MetricDefinition
	retention   map[string]RetentionPolicy
}

// NewMemoryRegistryRepository returns an empty repository.
func NewMemoryRegistryRepository() *MemoryRegistryRepository {
	return &MemoryRegistryRepository{
		assets:      make(map[string]Asset),
		connections: make(map[string]Connection),
		metrics:     make(map[string]MetricDefinition),
		retention:   make(map[string]RetentionPolicy),
	}
}

// CreateAsset implements RegistryRepository.
func (r *MemoryRegistryRepository) CreateAsset(_ context.Context, a Asset) (Asset, error) {
	if a.Status == "" {
		a.Status = AssetStatusPlanned
	}
	if err := a.Validate(); err != nil {
		return Asset{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	a.ID = uuid.NewString()
	now := timestamp()
	a.CreatedAt = now
	a.UpdatedAt = now
	r.assets[a.ID] = a
	return a, nil
}

// GetAsset implements RegistryRepository.
func (r *MemoryRegistryRepository) GetAsset(_ context.Context, id string) (Asset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.assets[id]
	if !ok {
		return Asset{}, ErrAssetNotFound
	}
	return a, nil
}

// ListAssets implements RegistryRepository.
func (r *MemoryRegistryRepository) ListAssets(_ context.Context, filter AssetFilter, limit int) ([]Asset, error) {
	if limit <= 0 {
		limit = DefaultPage
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Asset, 0)
	for _, a := range r.assets {
		if filter.AssetType != "" && a.AssetType != filter.AssetType {
			continue
		}
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// UpdateAsset implements RegistryRepository.
func (r *MemoryRegistryRepository) UpdateAsset(_ context.Context, a Asset) (Asset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.assets[a.ID]
	if !ok {
		return Asset{}, ErrAssetNotFound
	}
	updated := existing
	updated.Name = a.Name
	updated.Location = a.Location
	updated.InstalledAt = a.InstalledAt
	updated.Notes = a.Notes
	updated.Attributes = a.Attributes
	if err := updated.Validate(); err != nil {
		return Asset{}, err
	}
	updated.UpdatedAt = timestamp()
	r.assets[updated.ID] = updated
	return updated, nil
}

// UpdateAssetStatus implements RegistryRepository.
func (r *MemoryRegistryRepository) UpdateAssetStatus(_ context.Context, id string, next AssetStatus) (Asset, error) {
	if !IsAssetStatus(next) {
		return Asset{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.assets[id]
	if !ok {
		return Asset{}, ErrAssetNotFound
	}
	if !a.Status.CanTransitionTo(next) {
		return Asset{}, ErrInvalid
	}
	a.Status = next
	a.UpdatedAt = timestamp()
	r.assets[id] = a
	return a, nil
}

func (r *MemoryRegistryRepository) findConnectionLocked(fromID, toID string, kind ConnectionKind) (Connection, bool) {
	for _, c := range r.connections {
		if c.FromAssetID == fromID && c.ToAssetID == toID && c.Kind == kind {
			return c, true
		}
	}
	return Connection{}, false
}

// CreateConnection implements RegistryRepository.
func (r *MemoryRegistryRepository) CreateConnection(_ context.Context, c Connection) (Connection, error) {
	if err := c.Validate(); err != nil {
		return Connection{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.assets[c.FromAssetID]; !ok {
		return Connection{}, ErrInvalidConnection
	}
	if _, ok := r.assets[c.ToAssetID]; !ok {
		return Connection{}, ErrInvalidConnection
	}

	if existing, ok := r.findConnectionLocked(c.FromAssetID, c.ToAssetID, c.Kind); ok {
		return existing, nil
	}

	c.ID = uuid.NewString()
	c.CreatedAt = timestamp()
	r.connections[c.ID] = c
	return c, nil
}

// DeleteConnection implements RegistryRepository.
func (r *MemoryRegistryRepository) DeleteConnection(_ context.Context, fromAssetID, toAssetID string, kind ConnectionKind) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.findConnectionLocked(fromAssetID, toAssetID, kind)
	if !ok {
		return ErrConnectionNotFound
	}
	delete(r.connections, existing.ID)
	return nil
}

// ListConnections implements RegistryRepository.
func (r *MemoryRegistryRepository) ListConnections(_ context.Context, assetID string, kind ConnectionKind, dir Direction) ([]Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Connection, 0)
	for _, c := range r.connections {
		if kind != "" && c.Kind != kind {
			continue
		}
		switch dir {
		case DirectionInbound:
			if c.ToAssetID != assetID {
				continue
			}
		default:
			if c.FromAssetID != assetID {
				continue
			}
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}

func (r *MemoryRegistryRepository) findMetricLocked(assetType AssetType, name string) (MetricDefinition, bool) {
	for _, m := range r.metrics {
		if m.AssetType == assetType && m.Name == name {
			return m, true
		}
	}
	return MetricDefinition{}, false
}

// UpsertMetricDefinition implements RegistryRepository.
func (r *MemoryRegistryRepository) UpsertMetricDefinition(_ context.Context, m MetricDefinition) (MetricDefinition, error) {
	if err := m.Validate(); err != nil {
		return MetricDefinition{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := timestamp()
	if existing, ok := r.findMetricLocked(m.AssetType, m.Name); ok {
		m.ID = existing.ID
		m.CreatedAt = existing.CreatedAt
		m.UpdatedAt = now
		r.metrics[m.ID] = m
		return m, nil
	}
	m.ID = uuid.NewString()
	m.CreatedAt = now
	m.UpdatedAt = now
	r.metrics[m.ID] = m
	return m, nil
}

// ListMetricDefinitions implements RegistryRepository.
func (r *MemoryRegistryRepository) ListMetricDefinitions(_ context.Context, assetType AssetType) ([]MetricDefinition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]MetricDefinition, 0)
	for _, m := range r.metrics {
		if assetType != "" && m.AssetType != assetType {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (r *MemoryRegistryRepository) findRetentionLocked(assetType AssetType, metricName string) (RetentionPolicy, bool) {
	for _, p := range r.retention {
		if p.AssetType == assetType && p.MetricName == metricName {
			return p, true
		}
	}
	return RetentionPolicy{}, false
}

// SetRetentionPolicy implements RegistryRepository.
func (r *MemoryRegistryRepository) SetRetentionPolicy(_ context.Context, p RetentionPolicy) (RetentionPolicy, error) {
	if err := p.Validate(); err != nil {
		return RetentionPolicy{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := timestamp()
	if existing, ok := r.findRetentionLocked(p.AssetType, p.MetricName); ok {
		p.ID = existing.ID
		p.CreatedAt = existing.CreatedAt
		p.UpdatedAt = now
		r.retention[p.ID] = p
		return p, nil
	}
	p.ID = uuid.NewString()
	p.CreatedAt = now
	p.UpdatedAt = now
	r.retention[p.ID] = p
	return p, nil
}

// ListRetentionPolicies implements RegistryRepository.
func (r *MemoryRegistryRepository) ListRetentionPolicies(_ context.Context) ([]RetentionPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]RetentionPolicy, 0)
	for _, p := range r.retention {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}
