package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	twinmcp "github.com/aosanya/mwanachama-backend-digitaltwin/mcp"
	"github.com/aosanya/mwanachama-backend-digitaltwin/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newManager(t *testing.T) digitaltwin.TwinManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	return tm
}

func connect(t *testing.T, resolve twinmcp.ManagerResolver, m routes.Mount) (*gosdkmcp.ClientSession, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	server := gosdkmcp.NewServer(&gosdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := twinmcp.RegisterTools(server, resolve, m); err != nil {
		t.Fatalf("register: %v", err)
	}
	st, ct := gosdkmcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := gosdkmcp.NewClient(&gosdkmcp.Implementation{Name: "client", Version: "0.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, ctx
}

func byInstance(managers map[string]digitaltwin.TwinManager) twinmcp.ManagerResolver {
	return func(_ context.Context, id string) (digitaltwin.TwinManager, error) {
		tm, ok := managers[id]
		if !ok {
			return nil, errors.New("not found")
		}
		return tm, nil
	}
}

func callTool(t *testing.T, ctx context.Context, cs *gosdkmcp.ClientSession, name string, args map[string]any) *gosdkmcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func structured(t *testing.T, res *gosdkmcp.CallToolResult, into any) {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content[0].(*gosdkmcp.TextContent).Text)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

func TestEveryDeclaredOperationIsAToolThatNamesItsInstance(t *testing.T) {
	cs, ctx := connect(t, byInstance(nil), routes.Mount{})
	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Tools) != 17 {
		t.Fatalf("listed %d tools, want the 17 the operations file declares", len(list.Tools))
	}
	names := map[string]bool{}
	for _, tl := range list.Tools {
		names[tl.Name] = true
		raw, _ := json.Marshal(tl.InputSchema)
		if !strings.Contains(string(raw), `"instance_id"`) {
			t.Errorf("%s does not take an instance_id: %s", tl.Name, raw)
		}
	}
	for _, want := range []string{
		"digitaltwin_observation_latest", "digitaltwin_observation_query",
		"digitaltwin_node_set_status", "digitaltwin_observation_record_batch",
	} {
		if !names[want] {
			t.Errorf("tools/list is missing %s", want)
		}
	}
}

func TestBatchQueryLatestAndStatusThroughAClient(t *testing.T) {
	tm := newManager(t)
	cs, ctx := connect(t, byInstance(map[string]digitaltwin.TwinManager{"lab": tm}), routes.Mount{})

	var node digitaltwin.Node
	structured(t, callTool(t, ctx, cs, "digitaltwin_node_create", map[string]any{
		"instance_id": "lab", "name": "Line 4", "kind": "pipeline",
	}), &node)

	var recorded struct{ Items []digitaltwin.Observation }
	structured(t, callTool(t, ctx, cs, "digitaltwin_observation_record_batch", map[string]any{
		"instance_id": "lab",
		"items": []map[string]any{
			{"node_id": node.ID, "metric_name": "pressure_psi", "value": 1, "recorded_at": "2026-09-01T00:00:00Z"},
			{"node_id": node.ID, "metric_name": "pressure_psi", "value": 2, "recorded_at": "2026-09-20T00:00:00Z"},
			{"node_id": node.ID, "metric_name": "flow_m3h", "value": 3, "recorded_at": "2026-09-20T00:00:00Z"},
		},
	}), &recorded)

	var queried struct{ Items []digitaltwin.Observation }
	structured(t, callTool(t, ctx, cs, "digitaltwin_observation_query", map[string]any{
		"instance_id": "lab", "node_id": node.ID, "metric_name": "pressure_psi", "from": "2026-09-10T00:00:00Z",
	}), &queried)
	if len(queried.Items) != 1 || queried.Items[0].Value != 2 {
		t.Fatalf("query answered %+v", queried.Items)
	}

	var latest struct{ Items []digitaltwin.Observation }
	structured(t, callTool(t, ctx, cs, "digitaltwin_observation_latest", map[string]any{
		"instance_id": "lab", "nodeID": node.ID,
	}), &latest)
	if len(latest.Items) != 2 {
		t.Fatalf("latest answered %d observations, want one per metric", len(latest.Items))
	}

	var moved digitaltwin.Node
	structured(t, callTool(t, ctx, cs, "digitaltwin_node_set_status", map[string]any{
		"instance_id": "lab", "nodeID": node.ID, "status": "operational",
	}), &moved)
	if string(moved.Status) != "operational" {
		t.Fatalf("status = %q, want operational", moved.Status)
	}
}

func TestEachInstanceReachesOnlyItsOwnTwin(t *testing.T) {
	one, two := newManager(t), newManager(t)
	cs, ctx := connect(t, byInstance(map[string]digitaltwin.TwinManager{"one": one, "two": two}), routes.Mount{})

	n, err := one.CreateNode(context.Background(), digitaltwin.Node{Name: "Line 4", Kind: "pipeline"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if res := callTool(t, ctx, cs, "digitaltwin_node_read", map[string]any{"instance_id": "two", "nodeID": n.ID}); !res.IsError {
		t.Fatalf("instance two read instance one's node")
	}
	if res := callTool(t, ctx, cs, "digitaltwin_node_read", map[string]any{"instance_id": "one", "nodeID": n.ID}); res.IsError {
		t.Fatalf("instance one could not read its own node")
	}
	if res := callTool(t, ctx, cs, "digitaltwin_node_read", map[string]any{"instance_id": "nobody", "nodeID": n.ID}); !res.IsError {
		t.Fatalf("an unknown instance resolved")
	}
	if res := callTool(t, ctx, cs, "digitaltwin_node_read", map[string]any{"nodeID": n.ID}); !res.IsError {
		t.Fatalf("a call naming no instance was answered")
	}
}

func TestARefusedActionNeverReachesTheManager(t *testing.T) {
	tm := newManager(t)
	deny := func(context.Context, string) error { return errors.New("refused") }
	cs, ctx := connect(t, byInstance(map[string]digitaltwin.TwinManager{"lab": tm}), routes.Mount{Authorize: deny})
	if res := callTool(t, ctx, cs, "digitaltwin_node_create", map[string]any{
		"instance_id": "lab", "name": "Line 4", "kind": "pipeline",
	}); !res.IsError {
		t.Fatalf("a refused create was answered")
	}
	if res := callTool(t, ctx, cs, "digitaltwin_observation_record_batch", map[string]any{
		"instance_id": "lab", "items": []map[string]any{},
	}); !res.IsError {
		t.Fatalf("a refused batch was answered")
	}
	nodes, err := tm.ListNodes(context.Background(), digitaltwin.NodeFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("a refused create stored %d node(s)", len(nodes))
	}
}
