package digitaltwin

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func (m *twinManager) RecordObservation(ctx context.Context, o Observation) (Observation, error) {
	stamped, err := m.stamp(o)
	if err != nil {
		return Observation{}, err
	}
	if err := m.insert(ctx, roleObservation, stamped); err != nil {
		return Observation{}, err
	}
	return stamped, nil
}

func (m *twinManager) RecordObservations(ctx context.Context, os []Observation) ([]Observation, error) {
	out := make([]Observation, 0, len(os))
	for _, o := range os {
		stamped, err := m.stamp(o)
		if err != nil {
			return nil, err
		}
		out = append(out, stamped)
	}
	if len(out) == 0 {
		return out, nil
	}

	rows := make([]map[string]any, 0, len(out))
	object := m.st.Object(roleObservation)
	for _, o := range out {
		row, err := encode(object, o)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if err := m.q(ctx, roleObservation).Create(rows).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (m *twinManager) stamp(o Observation) (Observation, error) {
	if err := m.checks(roleObservation, o); err != nil {
		return Observation{}, err
	}
	if !isRFC3339(o.RecordedAt) {
		return Observation{}, fmt.Errorf("%w: observation.recorded_at %q is not an RFC 3339 instant",
			ErrInvalid, o.RecordedAt)
	}
	o.ID = newID()
	o.IngestedAt = m.now()
	return o, nil
}

func isRFC3339(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

func (m *twinManager) QueryObservations(ctx context.Context, filter ObservationFilter) ([]Observation, error) {
	q := m.q(ctx, roleObservation)
	if filter.NodeID != "" {
		q = q.Where("node_id = ?", filter.NodeID)
	}
	if filter.MetricName != "" {
		q = q.Where("metric_name = ?", filter.MetricName)
	}
	if isRFC3339(filter.From) {
		q = q.Where("recorded_at >= ?", filter.From)
	}
	if isRFC3339(filter.To) {
		q = q.Where("recorded_at <= ?", filter.To)
	}
	return listOf[Observation](m, q.Order("recorded_at").Limit(capped(filter.Limit)), roleObservation)
}

func (m *twinManager) Latest(ctx context.Context, nodeID, metricName string) (any, error) {
	if metricName != "" {
		return m.latestOne(ctx, nodeID, metricName)
	}
	return m.latestPerMetric(ctx, nodeID)
}

func (m *twinManager) latestOne(ctx context.Context, nodeID, metricName string) (Observation, error) {
	q := m.q(ctx, roleObservation).
		Where("node_id = ?", nodeID).
		Where("metric_name = ?", metricName).
		Order("recorded_at DESC")
	var out Observation
	err := m.take(q, roleObservation, &out, ErrObservationNotFound)
	return out, err
}

func (m *twinManager) latestPerMetric(ctx context.Context, nodeID string) ([]Observation, error) {
	names, err := m.metricNamesFor(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	out := make([]Observation, 0, len(names))
	for _, name := range names {
		o, err := m.latestOne(ctx, nodeID, name)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func (m *twinManager) metricNamesFor(ctx context.Context, nodeID string) ([]string, error) {
	var names []string
	err := m.q(ctx, roleObservation).
		Where("node_id = ?", nodeID).
		Distinct("metric_name").
		Order("metric_name").
		Pluck("metric_name", &names).Error
	return names, err
}

func (m *twinManager) PurgeObservationsBefore(ctx context.Context, nodeKind, metricName, cutoff string) (int, error) {
	q := m.q(ctx, roleObservation).Where("recorded_at < ?", cutoff)
	if nodeKind != "" {
		q = q.Where("node_kind = ?", nodeKind)
	}
	if metricName != "" {
		q = q.Where("metric_name = ?", metricName)
	}
	return deleted(q.Delete(nil))
}

func deleted(tx *gorm.DB) (int, error) {
	if tx.Error != nil {
		return 0, tx.Error
	}
	return int(tx.RowsAffected), nil
}
