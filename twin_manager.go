package digitaltwin

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

const DefaultPage = 200

type (
	Node              = models.Node
	NodeFilter        = models.NodeFilter
	Location          = models.Location
	Link              = models.Link
	Metric            = models.Metric
	Observation       = models.Observation
	ObservationFilter = models.ObservationFilter
	Retention         = models.Retention
	Status            = models.Status
	Relation          = models.Relation
	Direction         = models.Direction
)

const (
	StatusPlanned        = models.StatusPlanned
	StatusOperational    = models.StatusOperational
	StatusMaintenance    = models.StatusMaintenance
	StatusFault          = models.StatusFault
	StatusDecommissioned = models.StatusDecommissioned

	RelationConnectsTo = models.RelationConnectsTo
	RelationPartOf     = models.RelationPartOf
	RelationMonitors   = models.RelationMonitors

	DirectionOutbound = models.DirectionOutbound
	DirectionInbound  = models.DirectionInbound
)

type TwinManager interface {
	CreateNode(ctx context.Context, n Node) (Node, error)
	GetNode(ctx context.Context, id string) (Node, error)
	ListNodes(ctx context.Context, filter NodeFilter) ([]Node, error)
	UpdateNode(ctx context.Context, n Node) (Node, error)
	SetNodeStatus(ctx context.Context, id string, next Status) (Node, error)

	CreateLink(ctx context.Context, l Link) (Link, error)
	DeleteLink(ctx context.Context, relation Relation, fromNodeID, toNodeID string) error
	ListLinks(ctx context.Context, nodeID string, relation Relation, direction Direction) ([]Link, error)

	UpsertMetric(ctx context.Context, m Metric) (Metric, error)
	ListMetrics(ctx context.Context, nodeKind string) ([]Metric, error)

	SetRetention(ctx context.Context, r Retention) (Retention, error)
	ListRetentions(ctx context.Context) ([]Retention, error)

	RecordObservation(ctx context.Context, o Observation) (Observation, error)
	RecordObservations(ctx context.Context, os []Observation) ([]Observation, error)
	QueryObservations(ctx context.Context, filter ObservationFilter) ([]Observation, error)
	Latest(ctx context.Context, nodeID, metricName string) (any, error)
	PurgeObservationsBefore(ctx context.Context, nodeKind, metricName, cutoff string) (int, error)
}

type twinManager struct {
	db  *gorm.DB
	st  *store
	now func() string
}

func NewTwinManager(db *gorm.DB, s *spec.Spec) (TwinManager, error) {
	st, err := newStore(db, s, map[string]any{
		roleNode:        models.Node{},
		roleLink:        models.Link{},
		roleMetric:      models.Metric{},
		roleObservation: models.Observation{},
		roleRetention:   models.Retention{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewTwinManager: %w", err)
	}
	return &twinManager{db: db, st: st, now: nowRFC3339}, nil
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func (m *twinManager) q(ctx context.Context, role string) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.st.Table(role))
}

func (m *twinManager) take(q *gorm.DB, role string, out any, notFound error) error {
	var rows []map[string]any
	if err := q.Limit(1).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return notFound
	}
	return decode(m.st.Object(role), rows[0], out)
}

func listOf[T any](m *twinManager, q *gorm.DB, role string) ([]T, error) {
	return specstore.List[T](m.st, q, role)
}

func (m *twinManager) insert(ctx context.Context, role string, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Create(row).Error
}

func (m *twinManager) update(ctx context.Context, role, key string, id any, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Where(key+" = ?", id).Updates(row).Error
}

func capped(limit int) int {
	if limit <= 0 {
		return DefaultPage
	}
	return limit
}
