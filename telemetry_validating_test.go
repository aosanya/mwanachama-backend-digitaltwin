package digitaltwin

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestValidatingTelemetryRepositoryRejectsUnregisteredMetric(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	v := NewValidatingTelemetryRepository(NewMemoryTelemetryRepository(), registry)

	_, err := v.Record(ctx, TelemetryReading{
		AssetID: "pipeline-1", AssetType: AssetTypePipeline,
		MetricName: "pressure_psi", Value: 42, Unit: "psi", RecordedAt: time.Now(),
	})
	if !errors.Is(err, ErrUnknownMetric) {
		t.Fatalf("want ErrUnknownMetric, got %v", err)
	}
}

func TestValidatingTelemetryRepositoryAcceptsRegisteredMetric(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	v := NewValidatingTelemetryRepository(NewMemoryTelemetryRepository(), registry)

	if _, err := registry.UpsertMetricDefinition(ctx, MetricDefinition{
		Name: "pressure_psi", Unit: "psi", AssetType: AssetTypePipeline,
	}); err != nil {
		t.Fatal(err)
	}

	out, err := v.Record(ctx, TelemetryReading{
		AssetID: "pipeline-1", AssetType: AssetTypePipeline,
		MetricName: "pressure_psi", Value: 42, Unit: "psi", RecordedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if out.ID == "" {
		t.Fatal("Record did not delegate to the wrapped repository")
	}
}

func TestValidatingTelemetryRepositoryAcceptsAnyTypeScopedMetric(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	v := NewValidatingTelemetryRepository(NewMemoryTelemetryRepository(), registry)

	// AssetType left empty — applies to any asset type.
	if _, err := registry.UpsertMetricDefinition(ctx, MetricDefinition{
		Name: "battery_pct", Unit: "%",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := v.Record(ctx, TelemetryReading{
		AssetID: "sensor-1", AssetType: AssetTypeSensor,
		MetricName: "battery_pct", Value: 87, Unit: "%", RecordedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
}

func TestValidatingTelemetryRepositoryRejectsUnitMismatch(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	v := NewValidatingTelemetryRepository(NewMemoryTelemetryRepository(), registry)

	if _, err := registry.UpsertMetricDefinition(ctx, MetricDefinition{
		Name: "pressure_psi", Unit: "psi", AssetType: AssetTypePipeline,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := v.Record(ctx, TelemetryReading{
		AssetID: "pipeline-1", AssetType: AssetTypePipeline,
		MetricName: "pressure_psi", Value: 42, Unit: "bar", RecordedAt: time.Now(),
	})
	if !errors.Is(err, ErrUnknownMetric) {
		t.Fatalf("want ErrUnknownMetric for a unit mismatch, got %v", err)
	}
}

func TestValidatingTelemetryRepositoryRecordBatchIsAllOrNothing(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistryRepository()
	memory := NewMemoryTelemetryRepository()
	v := NewValidatingTelemetryRepository(memory, registry)

	if _, err := registry.UpsertMetricDefinition(ctx, MetricDefinition{
		Name: "pressure_psi", Unit: "psi", AssetType: AssetTypePipeline,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := v.RecordBatch(ctx, []TelemetryReading{
		{AssetID: "pipeline-1", AssetType: AssetTypePipeline, MetricName: "pressure_psi", Value: 1, Unit: "psi", RecordedAt: time.Now()},
		{AssetID: "pipeline-1", AssetType: AssetTypePipeline, MetricName: "unregistered_metric", Value: 2, RecordedAt: time.Now()},
	})
	if !errors.Is(err, ErrUnknownMetric) {
		t.Fatalf("want ErrUnknownMetric, got %v", err)
	}

	got, err := memory.Query(ctx, TelemetryFilter{AssetID: "pipeline-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("a failed batch must store nothing, got %d readings", len(got))
	}
}
