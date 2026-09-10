// postgres_telemetry_integration_test.go exercises PostgresTelemetryRepository
// against a real, dedicated Postgres table — the same conformance suite
// that runs against MemoryTelemetryRepository in memory_test.go.
//
// Skipped unless POSTGRES_URL is set.
package digitaltwin_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-shared/postgres"
	"github.com/google/uuid"
)

// newPostgresTelemetry opens a fresh connection to dsn and creates a
// fresh, uniquely-named telemetry table. See newPostgresRegistry's doc
// comment (postgres_registry_integration_test.go) for why this takes no
// *testing.T and reports setup failures via panic rather than t.Fatalf.
func newPostgresTelemetry(dsn string) digitaltwin.TelemetryRepository {
	ctx := context.Background()
	db, err := postgres.Open(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		panic(fmt.Errorf("newPostgresTelemetry: Open: %w", err))
	}

	table := "dtwtel_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if err := applyDDL(ctx, db, digitaltwin.TelemetryDDL(table)); err != nil {
		panic(fmt.Errorf("newPostgresTelemetry: applying DDL: %w", err))
	}
	pendingCleanupsMu.Lock()
	pendingCleanups = append(pendingCleanups, func() {
		_ = applyDDL(context.Background(), db, digitaltwin.TelemetryDropDDL(table))
		_ = db.Close()
	})
	pendingCleanupsMu.Unlock()

	return digitaltwin.NewPostgresTelemetryRepository(db, table)
}

// TestPostgresTelemetryRepository runs the shared conformance suite against
// PostgresTelemetryRepository — the same suite that runs against
// MemoryTelemetryRepository in memory_test.go. Both backends must agree.
func TestPostgresTelemetryRepository(t *testing.T) {
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test (see Makefile's test-pg target)")
	}
	digitaltwin.RunTelemetryConformance(t, func() digitaltwin.TelemetryRepository {
		return newPostgresTelemetry(dsn)
	})
}
