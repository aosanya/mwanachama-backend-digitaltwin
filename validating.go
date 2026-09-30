package digitaltwin

import (
	"context"
	"fmt"
)

type ValidatingTwinManager struct {
	TwinManager
}

func NewValidatingTwinManager(tm TwinManager) *ValidatingTwinManager {
	return &ValidatingTwinManager{TwinManager: tm}
}

func (v *ValidatingTwinManager) declared(ctx context.Context, o Observation) error {
	metrics, err := v.TwinManager.ListMetrics(ctx, "")
	if err != nil {
		return fmt.Errorf("ValidatingTwinManager: ListMetrics: %w", err)
	}
	for _, d := range metrics {
		if !d.Covers(o.NodeKind, o.MetricName) {
			continue
		}
		if o.Unit != "" && d.Unit != o.Unit {
			continue
		}
		return nil
	}
	return fmt.Errorf("%w: %s node_kind=%s unit=%s", ErrUnknownMetric, o.MetricName, o.NodeKind, o.Unit)
}

func (v *ValidatingTwinManager) RecordObservation(ctx context.Context, o Observation) (Observation, error) {
	if err := v.declared(ctx, o); err != nil {
		return Observation{}, err
	}
	return v.TwinManager.RecordObservation(ctx, o)
}

func (v *ValidatingTwinManager) RecordObservations(ctx context.Context, os []Observation) ([]Observation, error) {
	for _, o := range os {
		if err := v.declared(ctx, o); err != nil {
			return nil, err
		}
	}
	return v.TwinManager.RecordObservations(ctx, os)
}

var _ TwinManager = (*ValidatingTwinManager)(nil)
