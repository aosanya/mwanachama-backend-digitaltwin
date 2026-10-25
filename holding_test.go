package digitaltwin_test

import (
	"errors"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func square(minLon, minLat, maxLon, maxLat float64) *geo.Geometry {
	g := geo.NewPolygon(geo.Ring{
		{minLon, minLat}, {maxLon, minLat}, {maxLon, maxLat}, {minLon, maxLat}, {minLon, minLat},
	})
	return &g
}

func TestTheDeepestAreaHoldingAPointIsTheSmallestThatContainsIt(t *testing.T) {
	tm, ctx := newManagerFor(t, "place")
	for _, n := range []digitaltwin.Node{
		{Name: "County", Kind: "area", Shape: square(0, 0, 10, 10)},
		{Name: "Town", Kind: "area", Shape: square(1, 1, 5, 5)},
		{Name: "Ward", Kind: "area", Shape: square(2, 2, 3, 3)},
		{Name: "Elsewhere", Kind: "area", Shape: square(20, 20, 21, 21)},
		{Name: "Plot", Kind: "address", Shape: square(2.4, 2.4, 2.6, 2.6)},
	} {
		if _, err := tm.CreateNode(ctx, n); err != nil {
			t.Fatalf("create %s: %v", n.Name, err)
		}
	}

	cases := map[string]geo.Position{"Ward": {2.5, 2.5}, "Town": {4, 4}, "County": {9, 9}}
	for want, at := range cases {
		got, err := tm.DeepestHolding(ctx, "area", at)
		if err != nil {
			t.Fatalf("%v: %v", at, err)
		}
		if got.Name != want {
			t.Errorf("%v: got %s, want %s", at, got.Name, want)
		}
	}

	if got, err := tm.DeepestHolding(ctx, "", geo.Position{2.5, 2.5}); err != nil || got.Name != "Plot" {
		t.Errorf("any kind: got %q, %v; want Plot", got.Name, err)
	}

	if _, err := tm.DeepestHolding(ctx, "area", geo.Position{50, 50}); !errors.Is(err, digitaltwin.ErrNodeNotFound) {
		t.Errorf("outside every area: got %v, want ErrNodeNotFound", err)
	}
}

func TestANodeShapeSurvivesAnUpdate(t *testing.T) {
	tm, ctx := newManagerFor(t, "place")
	n, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Ward", Kind: "area", Shape: square(0, 0, 1, 1)})
	if err != nil {
		t.Fatal(err)
	}
	n.Shape = square(0, 0, 2, 2)
	if _, err := tm.UpdateNode(ctx, n); err != nil {
		t.Fatal(err)
	}
	got, err := tm.GetNode(ctx, n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Shape == nil || geo.BoundingBox(*got.Shape).MaxLon != 2 {
		t.Errorf("shape after update: %+v", got.Shape)
	}
}

func TestAStreetCarriesItsLine(t *testing.T) {
	tm, ctx := newManagerFor(t, "place")
	line := geo.NewLineString(geo.Position{1, 1}, geo.Position{4, 1})
	n, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Main", Kind: "street", Shape: &line})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := tm.GetNode(ctx, n.ID)
	if err != nil || got.Shape == nil || got.Shape.Kind != geo.KindLineString || len(got.Shape.LineString) != 2 {
		t.Fatalf("read back %+v, %v", got.Shape, err)
	}
	if held, err := tm.DeepestHolding(ctx, "street", geo.Position{2, 1}); err != nil || held.ID != n.ID {
		t.Fatalf("a point on the street answered %+v, %v", held, err)
	}
}
