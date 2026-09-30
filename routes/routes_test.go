package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newManager(t *testing.T) (digitaltwin.TwinManager, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s, err := digitaltwin.LoadSpec(filepath.Join("..", "spec", "examples", "utility.digitaltwin.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	tm, err := digitaltwin.NewTwinManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return tm, context.Background()
}

func newMux(t *testing.T) (*http.ServeMux, digitaltwin.TwinManager, context.Context) {
	t.Helper()
	tm, ctx := newManager(t)
	built, err := routes.Build(tm)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	mux := http.NewServeMux()
	for _, rt := range built {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return mux, tm, ctx
}

func do(t *testing.T, mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestTheDeclaredTableBuilds(t *testing.T) {
	tm, _ := newManager(t)
	built, err := routes.Build(tm)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(built) != 16 {
		t.Fatalf("built %d routes, want the 16 the operations file declares", len(built))
	}
	for _, rt := range built {
		if rt.Action == "" {
			t.Errorf("%s %s carries no action, so nothing can gate it", rt.Method, rt.Path)
		}
	}
}

func TestEveryRouteIsGatedUntilNamedAnonymous(t *testing.T) {
	tm, _ := newManager(t)
	if len(routes.AnonymousActions) != 0 {
		t.Fatalf("AnonymousActions is %v; this module publishes nothing anonymously", routes.AnonymousActions)
	}
	if public := routes.PublicRoutes(tm); len(public) != 0 {
		t.Fatalf("%d routes are reachable without a caller", len(public))
	}
	if gated := routes.OperatorRoutes(tm); len(gated) != 16 {
		t.Fatalf("%d routes are gated, want all 16", len(gated))
	}
}

func TestARefusedActionNeverReachesTheManager(t *testing.T) {
	tm, _ := newManager(t)
	deny := func(ctx context.Context, action string) error { return context.Canceled }
	built, err := routes.BuildWith(tm, deny)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	mux := http.NewServeMux()
	for _, rt := range built {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	w := do(t, mux, "POST", "/nodes", `{"name":"Line 4","kind":"pipeline"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", w.Code)
	}
}

func TestTheAddressOutranksTheBody(t *testing.T) {
	mux, tm, ctx := newMux(t)
	kept, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Line 4", Kind: "pipeline"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	other, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Line 9", Kind: "pipeline"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	w := do(t, mux, "PUT", "/nodes/"+kept.ID, `{"id":"`+other.ID+`","name":"renamed by the body"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d: %s", w.Code, w.Body)
	}
	var got digitaltwin.Node
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != kept.ID {
		t.Fatalf("the body's id won: updated %s, address named %s", got.ID, kept.ID)
	}

	untouched, err := tm.GetNode(ctx, other.ID)
	if err != nil {
		t.Fatalf("read the other node: %v", err)
	}
	if untouched.Name != "Line 9" {
		t.Fatalf("a PUT to one address rewrote another node: %+v", untouched)
	}
}

func TestSentinelsArriveAtTheirDeclaredStatus(t *testing.T) {
	mux, _, _ := newMux(t)

	for _, c := range []struct {
		name   string
		method string
		target string
		body   string
		want   int
	}{
		{"an unknown node", "GET", "/nodes/nobody", "", http.StatusNotFound},
		{"a node with no kind", "POST", "/nodes", `{"name":"nameless"}`, http.StatusBadRequest},
		{"an unknown link", "DELETE", "/links/part_of/a/b", "", http.StatusNotFound},
		{"a link to nothing", "POST", "/links", `{"relation":"part_of","from_node_id":"a","to_node_id":"b"}`, http.StatusBadRequest},
		{"no observation yet", "GET", "/nodes/nobody/observations/latest?metric_name=pressure_psi", "", http.StatusNotFound},
		{"a malformed body", "POST", "/nodes", `not json`, http.StatusBadRequest},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := do(t, mux, c.method, c.target, c.body)
			if w.Code != c.want {
				t.Fatalf("code = %d, want %d: %s", w.Code, c.want, w.Body)
			}
		})
	}
}

func TestADeleteAnswersNoContent(t *testing.T) {
	mux, tm, ctx := newMux(t)
	from, _ := tm.CreateNode(ctx, digitaltwin.Node{Name: "Valve 9", Kind: "valve"})
	to, _ := tm.CreateNode(ctx, digitaltwin.Node{Name: "Line 4", Kind: "pipeline"})
	if _, err := tm.CreateLink(ctx, digitaltwin.Link{
		Relation: digitaltwin.RelationPartOf, FromNodeID: from.ID, ToNodeID: to.ID,
	}); err != nil {
		t.Fatalf("link: %v", err)
	}

	w := do(t, mux, "DELETE", "/links/part_of/"+from.ID+"/"+to.ID, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204: %s", w.Code, w.Body)
	}
}

func TestAWriteAnswersCreated(t *testing.T) {
	mux, _, _ := newMux(t)
	w := do(t, mux, "POST", "/nodes", `{"name":"Line 4","kind":"pipeline"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201: %s", w.Code, w.Body)
	}
}

func TestQueryBindsTheWholeFilter(t *testing.T) {
	mux, tm, ctx := newMux(t)
	n, _ := tm.CreateNode(ctx, digitaltwin.Node{Name: "Line 4", Kind: "pipeline"})
	if _, err := tm.RecordObservations(ctx, []digitaltwin.Observation{
		{NodeID: n.ID, MetricName: "pressure_psi", Value: 1, RecordedAt: "2026-09-01T00:00:00Z"},
		{NodeID: n.ID, MetricName: "pressure_psi", Value: 2, RecordedAt: "2026-09-20T00:00:00Z"},
		{NodeID: n.ID, MetricName: "flow_m3h", Value: 3, RecordedAt: "2026-09-20T00:00:00Z"},
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	w := do(t, mux, "GET", "/observations?node_id="+n.ID+"&metric_name=pressure_psi&from=2026-09-10T00:00:00Z", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d: %s", w.Code, w.Body)
	}
	var got []digitaltwin.Observation
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Value != 2 {
		t.Fatalf("the query filter did not bind: %+v", got)
	}
}

func TestLatestSwitchesOnTheMetricName(t *testing.T) {
	mux, tm, ctx := newMux(t)
	n, _ := tm.CreateNode(ctx, digitaltwin.Node{Name: "Line 4", Kind: "pipeline"})
	if _, err := tm.RecordObservations(ctx, []digitaltwin.Observation{
		{NodeID: n.ID, MetricName: "pressure_psi", Value: 1, RecordedAt: "2026-09-01T00:00:00Z"},
		{NodeID: n.ID, MetricName: "flow_m3h", Value: 3, RecordedAt: "2026-09-20T00:00:00Z"},
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	one := do(t, mux, "GET", "/nodes/"+n.ID+"/observations/latest?metric_name=pressure_psi", "")
	var single digitaltwin.Observation
	if err := json.Unmarshal(one.Body.Bytes(), &single); err != nil {
		t.Fatalf("decode one: %v — %s", err, one.Body)
	}
	if single.MetricName != "pressure_psi" {
		t.Fatalf("named metric answered %+v", single)
	}

	all := do(t, mux, "GET", "/nodes/"+n.ID+"/observations/latest", "")
	var many []digitaltwin.Observation
	if err := json.Unmarshal(all.Body.Bytes(), &many); err != nil {
		t.Fatalf("decode all: %v — %s", err, all.Body)
	}
	if len(many) != 2 {
		t.Fatalf("no metric named answered %d observations, want 2", len(many))
	}
}
