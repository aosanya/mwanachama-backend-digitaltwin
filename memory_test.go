package digitaltwin

import (
	"errors"
	"testing"
	"time"
)

// TestMemoryRegistryRepository runs the shared conformance suite against
// MemoryRegistryRepository. The same suite runs against Postgres once W4
// lands — both backends must agree.
func TestMemoryRegistryRepository(t *testing.T) {
	RunRegistryConformance(t, func() RegistryRepository { return NewMemoryRegistryRepository() })
}

// TestMemoryTelemetryRepository runs the shared conformance suite against
// MemoryTelemetryRepository. The same suite runs against the dedicated
// Postgres telemetry table once W4 lands.
func TestMemoryTelemetryRepository(t *testing.T) {
	RunTelemetryConformance(t, func() TelemetryRepository { return NewMemoryTelemetryRepository() })
}

func TestConnectionValidateRejectsSelfLoop(t *testing.T) {
	err := Connection{Kind: ConnectionPartOf, FromAssetID: "a", ToAssetID: "a"}.Validate()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for a self-loop, got %v", err)
	}
}

func TestMetricDefinitionValidateRejectsBackwardsBounds(t *testing.T) {
	min, max := 100.0, 10.0
	err := MetricDefinition{
		Name: "pressure_psi", Unit: "psi",
		MinValue: &min, MaxValue: &max,
	}.Validate()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for min > max, got %v", err)
	}
}

func TestRetentionPolicyValidateRequiresPositiveDays(t *testing.T) {
	err := RetentionPolicy{RetentionDays: 0}.Validate()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for retention_days <= 0, got %v", err)
	}
}

func TestRetentionPolicyValidateRequiresAssetTypeWithMetricName(t *testing.T) {
	err := RetentionPolicy{MetricName: "pressure_psi", RetentionDays: 30}.Validate()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for a metric-scoped policy with no asset_type, got %v", err)
	}
}

func TestTelemetryReadingValidateRejectsUnknownAssetType(t *testing.T) {
	err := TelemetryReading{
		AssetID: "a", AssetType: "bogus",
		MetricName: "pressure_psi", RecordedAt: time.Now(),
	}.Validate()
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for an unknown asset_type, got %v", err)
	}
}
