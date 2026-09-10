// postgres_registry_integration_test.go exercises EntitygraphRegistryRepository
// against a real Postgres-backed entitygraph.DataManager
// (mwanachama-backend-shared's postgres.Backend), rather than
// MemoryRegistryRepository.
//
// Skipped unless POSTGRES_URL is set — mirrors
// mwanachama-backend-taskmanager's own postgres_integration_test.go split.
package digitaltwin_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-shared/postgres"
	"github.com/google/uuid"
)

// applyDDL runs a multi-statement SQL script as one command.
func applyDDL(ctx context.Context, db *sql.DB, script string) error {
	_, err := db.ExecContext(ctx, script)
	return err
}

// pendingCleanups accumulates a drop/close for every scratch table set
// newPostgresRegistry creates. newRepo() has no per-subtest *testing.T to
// register t.Cleanup on (see its doc comment), so cleanup is deferred to
// TestMain, which runs once after every test in this package has finished.
var (
	pendingCleanupsMu sync.Mutex
	pendingCleanups   []func()
)

// TestMain drops every scratch table set created by newPostgresRegistry
// across this package's tests, after they've all run.
func TestMain(m *testing.M) {
	code := m.Run()
	pendingCleanupsMu.Lock()
	for _, cleanup := range pendingCleanups {
		cleanup()
	}
	pendingCleanupsMu.Unlock()
	os.Exit(code)
}

// newPostgresRegistry opens a fresh connection to dsn, creates a
// fresh, uniquely-prefixed set of entity-graph tables (RunRegistryConformance
// calls newRepo() once per subtest — a unique table set per call is what
// keeps subtests isolated from each other, since they'd otherwise share
// state through the same physical tables), and seeds+activates
// DefaultDigitalTwinSchema.
//
// Deliberately takes no *testing.T: RunRegistryConformance invokes newRepo()
// from inside each subtest's own goroutine, and calling Fatal/Skip on a
// *testing.T captured from the enclosing (parent) test — rather than the
// subtest's own T, which this closure signature has no access to — panics
// with "subtest may have called FailNow on a parent test" (Fatal/Skip call
// runtime.Goexit, which only unwinds the calling goroutine). Every setup
// step here is expected to always succeed once the top-level connectivity
// check in TestPostgresRegistryRepository has passed, so a genuine failure
// here is treated as a test-infrastructure bug and reported via panic,
// which go test's tRunner recovers from and reports as a normal failure.
func newPostgresRegistry(dsn string) digitaltwin.RegistryRepository {
	ctx := context.Background()
	db, err := postgres.Open(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		panic(fmt.Errorf("newPostgresRegistry: Open: %w", err))
	}

	prefix := "dtwreg_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12] + "_"
	tables := postgres.DefaultTableNames(prefix)
	if err := applyDDL(ctx, db, postgres.DDL(tables)); err != nil {
		panic(fmt.Errorf("newPostgresRegistry: applying DDL: %w", err))
	}
	pendingCleanupsMu.Lock()
	pendingCleanups = append(pendingCleanups, func() {
		_ = applyDDL(context.Background(), db, postgres.DropDDL(tables))
		_ = db.Close()
	})
	pendingCleanupsMu.Unlock()

	backend := postgres.NewBackend(db, tables)

	s := digitaltwin.DefaultDigitalTwinSchema()
	if err := backend.SetSchema(ctx, s); err != nil {
		panic(fmt.Errorf("newPostgresRegistry: SetSchema: %w", err))
	}
	if err := backend.Publish(ctx); err != nil {
		panic(fmt.Errorf("newPostgresRegistry: Publish: %w", err))
	}
	if err := backend.Activate(ctx, 1); err != nil {
		panic(fmt.Errorf("newPostgresRegistry: Activate: %w", err))
	}

	repo, err := digitaltwin.NewEntitygraphRegistryRepository(backend)
	if err != nil {
		panic(fmt.Errorf("newPostgresRegistry: NewEntitygraphRegistryRepository: %w", err))
	}
	return repo
}

// TestPostgresRegistryRepository runs the shared conformance suite against
// EntitygraphRegistryRepository — the same suite that runs against
// MemoryRegistryRepository in memory_test.go. Both backends must agree.
func TestPostgresRegistryRepository(t *testing.T) {
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test (see Makefile's test-pg target)")
	}
	digitaltwin.RunRegistryConformance(t, func() digitaltwin.RegistryRepository {
		return newPostgresRegistry(dsn)
	})
}
