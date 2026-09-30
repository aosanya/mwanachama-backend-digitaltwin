//go:build integration

package digitaltwin_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func randomInstance(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random instance: %v", err)
	}
	return "pg" + hex.EncodeToString(b)
}

func newPostgres(t *testing.T) (digitaltwin.TwinManager, *gorm.DB, *spec.Spec, context.Context) {
	t.Helper()
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}
	db, err := gorm.Open(gormpostgres.Open(url), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	raw, err := os.ReadFile(examplePath("utility"))
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	doc["instance"] = randomInstance(t)
	rewritten, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("rewrite spec: %v", err)
	}

	s, err := digitaltwin.ParseSpec(rewritten)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := digitaltwin.Provision(db, s); err != nil {
		t.Fatalf("provision: %v", err)
	}
	t.Cleanup(func() {
		for _, o := range s.Objects {
			db.Exec("drop table if exists " + s.TableFor(o) + " cascade")
		}
		db.Exec("drop table if exists " + s.NameRegistryTable() + " cascade")
	})

	tm, err := digitaltwin.NewTwinManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return tm, db, s, context.Background()
}

func TestPG_ADocumentSurvivesJSONBRoundTrip(t *testing.T) {
	tm, _, _, ctx := newPostgres(t)

	n, err := tm.CreateNode(ctx, digitaltwin.Node{
		Name: "Line 4", Kind: "pipeline",
		Location: &digitaltwin.Location{Lat: -1.286389, Lng: 36.817223},
		Doc: map[string]any{
			"station_type": "compressor",
			"diameter_mm":  float64(600),
			"nested":       map[string]any{"material": "steel"},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	read, err := tm.GetNode(ctx, n.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Location == nil || read.Location.Lat != -1.286389 {
		t.Fatalf("jsonb lost the position: %+v", read.Location)
	}
	if read.Doc["station_type"] != "compressor" || read.Doc["diameter_mm"] != float64(600) {
		t.Fatalf("jsonb lost a document value: %+v", read.Doc)
	}
	nested, ok := read.Doc["nested"].(map[string]any)
	if !ok || nested["material"] != "steel" {
		t.Fatalf("jsonb lost the nested document: %+v", read.Doc["nested"])
	}
}

func TestPG_AnAbsentPositionStaysNull(t *testing.T) {
	tm, db, s, ctx := newPostgres(t)

	n, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Line 4", Kind: "pipeline"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	node, _ := s.ByRole("node")
	var nulls int64
	if err := db.Table(s.TableFor(node)).
		Where("id = ?", n.ID).Where("location is null").Count(&nulls).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if nulls != 1 {
		t.Fatalf("an unset position was written as something other than NULL")
	}

	read, err := tm.GetNode(ctx, n.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Location != nil {
		t.Fatalf("a NULL position read back as %+v rather than absent", read.Location)
	}
}

func TestPG_TheDomainsDocumentPathIndexAnswers(t *testing.T) {
	tm, db, s, ctx := newPostgres(t)

	for _, kind := range []string{"compressor", "pump"} {
		if _, err := tm.CreateNode(ctx, digitaltwin.Node{
			Name: kind + " station", Kind: "station",
			Doc: map[string]any{"station_type": kind},
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	node, _ := s.ByRole("node")
	var found int64
	if err := db.Table(s.TableFor(node)).
		Where("doc #>> '{station_type}' = ?", "pump").Count(&found).Error; err != nil {
		t.Fatalf("filter by document path: %v", err)
	}
	if found != 1 {
		t.Fatalf("the document path filter matched %d rows, want 1", found)
	}
}

func TestPG_ObservationsOrderAndPurgeByTextTimestamp(t *testing.T) {
	tm, _, _, ctx := newPostgres(t)
	n, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Line 4", Kind: "pipeline"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	if _, err := tm.RecordObservations(ctx, []digitaltwin.Observation{
		{NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi", Value: 1, RecordedAt: now.AddDate(0, 0, -40).Format(time.RFC3339)},
		{NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi", Value: 2, RecordedAt: now.AddDate(0, 0, -20).Format(time.RFC3339)},
		{NodeID: n.ID, NodeKind: "pipeline", MetricName: "pressure_psi", Value: 3, RecordedAt: now.AddDate(0, 0, -1).Format(time.RFC3339)},
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	got, err := tm.QueryObservations(ctx, digitaltwin.ObservationFilter{NodeID: n.ID})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 3 || got[0].Value != 1 || got[2].Value != 3 {
		t.Fatalf("RFC 3339 text did not order oldest first: %+v", got)
	}

	latest, err := tm.Latest(ctx, n.ID, "pressure_psi")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.(digitaltwin.Observation).Value != 3 {
		t.Fatalf("latest = %+v, want the most recent", latest)
	}

	if _, err := tm.SetRetention(ctx, digitaltwin.Retention{
		NodeKind: "pipeline", MetricName: "pressure_psi", RetentionDays: 30,
	}); err != nil {
		t.Fatalf("rule: %v", err)
	}
	purged, err := digitaltwin.Sweep(ctx, tm, now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if purged != 1 {
		t.Fatalf("purged %d, want only the 40-day-old observation", purged)
	}
}

func TestPG_EveryDeclaredColumnIsWrittenOnEveryWrite(t *testing.T) {
	tm, _, _, ctx := newPostgres(t)

	n, err := tm.CreateNode(ctx, digitaltwin.Node{
		Name: "Line 4", Kind: "pipeline", Notes: "watch the east joint",
		Doc: map[string]any{"station_type": "compressor"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	cleared, err := tm.UpdateNode(ctx, digitaltwin.Node{ID: n.ID, Name: "Line 4"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if cleared.Notes != "" || len(cleared.Doc) != 0 {
		t.Fatalf("an omitted value was left alone rather than cleared: %+v", cleared)
	}

	read, err := tm.GetNode(ctx, n.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Notes != "" || len(read.Doc) != 0 {
		t.Fatalf("the clear did not reach the row: %+v", read)
	}
}
