package digitaltwin_test

import (
	"errors"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
)

func bound(v float64) *float64 { return &v }

func TestTheLabExampleCarriesItsKitMetricsAndLinks(t *testing.T) {
	tm, ctx := newManagerFor(t, "lab")

	kinds := map[string]string{
		"Terminal 1":        "terminal",
		"Terminal 2":        "terminal",
		"Access point hall": "access_point",
		"Router":            "router",
		"Server":            "server",
		"Solar array":       "solar",
		"Battery bank":      "battery",
	}
	ids := map[string]string{}
	for name, kind := range kinds {
		ids[name] = seedNode(t, tm, ctx, name, kind).ID
	}

	held, err := tm.ListNodes(ctx, digitaltwin.NodeFilter{Kind: "terminal"})
	if err != nil {
		t.Fatalf("list terminals: %v", err)
	}
	if len(held) != 2 {
		t.Fatalf("terminals = %d, want 2", len(held))
	}

	metrics := []digitaltwin.Metric{
		{Name: "online", Unit: "bool", MinValue: bound(0), MaxValue: bound(1)},
		{Name: "throughput", Unit: "Mbit/s", NodeKind: "router", MinValue: bound(0), MaxValue: bound(1000)},
		{Name: "throughput", Unit: "Mbit/s", NodeKind: "access_point", MinValue: bound(0), MaxValue: bound(300)},
		{Name: "battery_charge", Unit: "%", NodeKind: "battery", MinValue: bound(20), MaxValue: bound(100)},
	}
	for _, m := range metrics {
		if _, err := tm.UpsertMetric(ctx, m); err != nil {
			t.Fatalf("declare %s for %q: %v", m.Name, m.NodeKind, err)
		}
	}
	declared, err := tm.ListMetrics(ctx, "battery")
	if err != nil {
		t.Fatalf("list battery metrics: %v", err)
	}
	var charge *digitaltwin.Metric
	for i := range declared {
		if declared[i].Name == "battery_charge" {
			charge = &declared[i]
		}
	}
	if charge == nil || charge.MinValue == nil || *charge.MinValue != 20 || charge.MaxValue == nil || *charge.MaxValue != 100 {
		t.Fatalf("battery_charge limits did not round trip: %+v", charge)
	}

	inverted := digitaltwin.Metric{Name: "battery_charge", Unit: "%", NodeKind: "battery", MinValue: bound(100), MaxValue: bound(20)}
	if _, err := tm.UpsertMetric(ctx, inverted); !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("inverted limits: err = %v, want ErrInvalid", err)
	}

	links := []digitaltwin.Link{
		{Relation: digitaltwin.RelationConnectsTo, FromNodeID: ids["Terminal 1"], ToNodeID: ids["Access point hall"]},
		{Relation: digitaltwin.RelationConnectsTo, FromNodeID: ids["Terminal 2"], ToNodeID: ids["Access point hall"]},
		{Relation: digitaltwin.RelationConnectsTo, FromNodeID: ids["Access point hall"], ToNodeID: ids["Router"]},
		{Relation: digitaltwin.RelationConnectsTo, FromNodeID: ids["Server"], ToNodeID: ids["Router"]},
		{Relation: digitaltwin.RelationConnectsTo, FromNodeID: ids["Solar array"], ToNodeID: ids["Battery bank"]},
		{Relation: digitaltwin.RelationMonitors, FromNodeID: ids["Server"], ToNodeID: ids["Battery bank"]},
	}
	for _, l := range links {
		if _, err := tm.CreateLink(ctx, l); err != nil {
			t.Fatalf("link %s %s -> %s: %v", l.Relation, l.FromNodeID, l.ToNodeID, err)
		}
	}
	into, err := tm.ListLinks(ctx, ids["Access point hall"], digitaltwin.RelationConnectsTo, digitaltwin.DirectionInbound)
	if err != nil {
		t.Fatalf("links into the access point: %v", err)
	}
	if len(into) != 2 {
		t.Fatalf("terminals on the access point = %d, want 2", len(into))
	}

	strict := digitaltwin.NewValidatingTwinManager(tm)
	readings := []digitaltwin.Observation{
		{NodeID: ids["Terminal 1"], NodeKind: "terminal", MetricName: "online", Value: 1, Unit: "bool", RecordedAt: "2026-10-21T08:00:00Z"},
		{NodeID: ids["Router"], NodeKind: "router", MetricName: "throughput", Value: 42.5, Unit: "Mbit/s", RecordedAt: "2026-10-21T08:00:00Z"},
		{NodeID: ids["Battery bank"], NodeKind: "battery", MetricName: "battery_charge", Value: 76, Unit: "%", RecordedAt: "2026-10-21T08:00:00Z"},
		{NodeID: ids["Battery bank"], NodeKind: "battery", MetricName: "battery_charge", Value: 71, Unit: "%", RecordedAt: "2026-10-21T09:00:00Z"},
	}
	if _, err := strict.RecordObservations(ctx, readings); err != nil {
		t.Fatalf("record lab readings: %v", err)
	}
	latest, err := tm.Latest(ctx, ids["Battery bank"], "battery_charge")
	if err != nil {
		t.Fatalf("latest charge: %v", err)
	}
	if o, ok := latest.(digitaltwin.Observation); !ok || o.Value != 71 {
		t.Fatalf("latest charge = %+v, want 71", latest)
	}

	undeclared := digitaltwin.Observation{NodeID: ids["Server"], NodeKind: "server", MetricName: "battery_charge", Value: 50, Unit: "%", RecordedAt: "2026-10-21T08:00:00Z"}
	if _, err := strict.RecordObservation(ctx, undeclared); !errors.Is(err, digitaltwin.ErrUnknownMetric) {
		t.Fatalf("battery_charge on a server: err = %v, want ErrUnknownMetric", err)
	}
}
