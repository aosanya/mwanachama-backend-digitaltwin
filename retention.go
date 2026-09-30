package digitaltwin

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func (m *twinManager) SetRetention(ctx context.Context, r Retention) (Retention, error) {
	if err := m.checks(roleRetention, r); err != nil {
		return Retention{}, err
	}
	if !r.ScopeIsOrdered() {
		return Retention{}, fmt.Errorf("%w: a rule scoped to a metric_name must also name a node_kind", ErrInvalid)
	}
	if !r.KeepsForAPositivePeriod() {
		return Retention{}, fmt.Errorf("%w: retention_days must be positive (%d given); keeping forever is expressed by declaring no rule",
			ErrInvalid, r.RetentionDays)
	}

	held, err := m.retentionFor(ctx, r.NodeKind, r.MetricName)
	switch {
	case err == nil:
		r.ID = held.ID
		r.CreatedAt = held.CreatedAt
		r.UpdatedAt = m.now()
		if err := m.update(ctx, roleRetention, "id", r.ID, r); err != nil {
			return Retention{}, err
		}
		return r, nil
	case errors.Is(err, ErrRetentionNotFound):
		r.ID = newID()
		r.CreatedAt = m.now()
		r.UpdatedAt = r.CreatedAt
		if err := m.insert(ctx, roleRetention, r); err != nil {
			return Retention{}, err
		}
		return r, nil
	default:
		return Retention{}, err
	}
}

func (m *twinManager) retentionFor(ctx context.Context, nodeKind, metricName string) (Retention, error) {
	q := m.q(ctx, roleRetention).Where("node_kind = ?", nodeKind).Where("metric_name = ?", metricName)
	var out Retention
	err := m.take(q, roleRetention, &out, ErrRetentionNotFound)
	return out, err
}

func (m *twinManager) ListRetentions(ctx context.Context) ([]Retention, error) {
	return listOf[Retention](m, m.q(ctx, roleRetention).Order("created_at"), roleRetention)
}

func Sweep(ctx context.Context, tm TwinManager, now time.Time) (int, error) {
	rules, err := tm.ListRetentions(ctx)
	if err != nil {
		return 0, fmt.Errorf("Sweep: ListRetentions: %w", err)
	}

	total := 0
	for _, r := range rules {
		cutoff := now.AddDate(0, 0, -r.RetentionDays).UTC().Format(time.RFC3339)
		purged, err := tm.PurgeObservationsBefore(ctx, r.NodeKind, r.MetricName, cutoff)
		if err != nil {
			return total, fmt.Errorf("Sweep: PurgeObservationsBefore(node_kind=%q, metric_name=%q): %w",
				r.NodeKind, r.MetricName, err)
		}
		total += purged
	}
	return total, nil
}
