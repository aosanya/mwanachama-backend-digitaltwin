package digitaltwin_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func examplePath(domain string) string {
	return filepath.Join(".", "spec", "examples", domain+".digitaltwin.json")
}

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func newManagerFor(t *testing.T, domain string) (digitaltwin.TwinManager, context.Context) {
	t.Helper()
	db := newDB(t)
	s, err := digitaltwin.LoadSpec(examplePath(domain))
	if err != nil {
		t.Fatalf("load spec %s: %v", domain, err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate %s: %v", domain, err)
	}
	tm, err := digitaltwin.NewTwinManager(db, s)
	if err != nil {
		t.Fatalf("new manager %s: %v", domain, err)
	}
	return tm, context.Background()
}

func newManager(t *testing.T) (digitaltwin.TwinManager, context.Context) {
	t.Helper()
	return newManagerFor(t, "utility")
}

func seedNode(t *testing.T, tm digitaltwin.TwinManager, ctx context.Context, name, kind string) digitaltwin.Node {
	t.Helper()
	n, err := tm.CreateNode(ctx, digitaltwin.Node{Name: name, Kind: kind})
	if err != nil {
		t.Fatalf("create node %s: %v", name, err)
	}
	return n
}
