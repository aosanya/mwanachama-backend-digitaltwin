package digitaltwin

import (
	"context"
	"errors"
	"fmt"
)

func (m *twinManager) UpsertMetric(ctx context.Context, d Metric) (Metric, error) {
	if err := m.checks(roleMetric, d); err != nil {
		return Metric{}, err
	}
	if !d.BoundsOrdered() {
		return Metric{}, fmt.Errorf("%w: metric %q has a min_value above its max_value", ErrInvalid, d.Name)
	}

	held, err := m.metricNamed(ctx, d.NodeKind, d.Name)
	switch {
	case err == nil:
		d.ID = held.ID
		d.CreatedAt = held.CreatedAt
		d.UpdatedAt = m.now()
		if err := m.update(ctx, roleMetric, "id", d.ID, d); err != nil {
			return Metric{}, err
		}
		return d, nil
	case errors.Is(err, ErrMetricNotFound):
		d.ID = newID()
		d.CreatedAt = m.now()
		d.UpdatedAt = d.CreatedAt
		if err := m.insert(ctx, roleMetric, d); err != nil {
			return Metric{}, err
		}
		return d, nil
	default:
		return Metric{}, err
	}
}

func (m *twinManager) metricNamed(ctx context.Context, nodeKind, name string) (Metric, error) {
	q := m.q(ctx, roleMetric).Where("node_kind = ?", nodeKind).Where("name = ?", name)
	var out Metric
	err := m.take(q, roleMetric, &out, ErrMetricNotFound)
	return out, err
}

func (m *twinManager) ListMetrics(ctx context.Context, nodeKind string) ([]Metric, error) {
	q := m.q(ctx, roleMetric)
	if nodeKind != "" {
		q = q.Where("node_kind = ?", nodeKind)
	}
	return listOf[Metric](m, q.Order("name"), roleMetric)
}
