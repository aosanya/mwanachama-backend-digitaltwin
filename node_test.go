package digitaltwin_test

import (
	"errors"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
)

func TestANodeStartsPlannedAndKeepsItsKind(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")

	if n.Status != digitaltwin.StatusPlanned {
		t.Fatalf("status = %q, want planned", n.Status)
	}
	if n.ID == "" || n.CreatedAt == "" {
		t.Fatalf("the store assigns an id and a created_at, got %+v", n)
	}

	read, err := tm.GetNode(ctx, n.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if read.Kind != "pipeline" || read.Name != "Line 4" {
		t.Fatalf("round trip lost something: %+v", read)
	}
}

func TestANodeWithNoKindIsRefused(t *testing.T) {
	tm, ctx := newManager(t)
	if _, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "nameless kind"}); !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestOnlyDeclaredStatusStepsAreAccepted(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")

	if _, err := tm.SetNodeStatus(ctx, n.ID, digitaltwin.StatusMaintenance); !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("planned to maintenance: err = %v, want ErrInvalid", err)
	}
	moved, err := tm.SetNodeStatus(ctx, n.ID, digitaltwin.StatusOperational)
	if err != nil {
		t.Fatalf("planned to operational: %v", err)
	}
	if moved.Status != digitaltwin.StatusOperational {
		t.Fatalf("status = %q", moved.Status)
	}

	if _, err := tm.SetNodeStatus(ctx, n.ID, "melted"); !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("undeclared status: err = %v, want ErrInvalid", err)
	}
}

func TestADecommissionedNodeStaysThere(t *testing.T) {
	tm, ctx := newManager(t)
	n := seedNode(t, tm, ctx, "Line 4", "pipeline")
	if _, err := tm.SetNodeStatus(ctx, n.ID, digitaltwin.StatusDecommissioned); err != nil {
		t.Fatalf("decommission: %v", err)
	}
	if _, err := tm.SetNodeStatus(ctx, n.ID, digitaltwin.StatusOperational); !errors.Is(err, digitaltwin.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestLinkingIsIdempotentAndNeedsBothEnds(t *testing.T) {
	tm, ctx := newManager(t)
	from := seedNode(t, tm, ctx, "Line 4", "pipeline")
	to := seedNode(t, tm, ctx, "Valve 9", "valve")

	first, err := tm.CreateLink(ctx, digitaltwin.Link{
		Relation: digitaltwin.RelationPartOf, FromNodeID: to.ID, ToNodeID: from.ID,
	})
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	again, err := tm.CreateLink(ctx, digitaltwin.Link{
		Relation: digitaltwin.RelationPartOf, FromNodeID: to.ID, ToNodeID: from.ID,
	})
	if err != nil {
		t.Fatalf("relink: %v", err)
	}
	if first.ID != again.ID {
		t.Fatalf("relinking made a second edge: %s then %s", first.ID, again.ID)
	}

	_, err = tm.CreateLink(ctx, digitaltwin.Link{
		Relation: digitaltwin.RelationPartOf, FromNodeID: to.ID, ToNodeID: "nobody",
	})
	if !errors.Is(err, digitaltwin.ErrInvalidLink) {
		t.Fatalf("err = %v, want ErrInvalidLink", err)
	}
}

func TestALinkWalksBothWays(t *testing.T) {
	tm, ctx := newManager(t)
	line := seedNode(t, tm, ctx, "Line 4", "pipeline")
	valve := seedNode(t, tm, ctx, "Valve 9", "valve")
	if _, err := tm.CreateLink(ctx, digitaltwin.Link{
		Relation: digitaltwin.RelationPartOf, FromNodeID: valve.ID, ToNodeID: line.ID,
	}); err != nil {
		t.Fatalf("link: %v", err)
	}

	out, err := tm.ListLinks(ctx, valve.ID, "", digitaltwin.DirectionOutbound)
	if err != nil || len(out) != 1 {
		t.Fatalf("outbound from the valve = %d edges, err %v", len(out), err)
	}
	in, err := tm.ListLinks(ctx, line.ID, "", digitaltwin.DirectionInbound)
	if err != nil || len(in) != 1 {
		t.Fatalf("inbound to the line = %d edges, err %v", len(in), err)
	}
	none, err := tm.ListLinks(ctx, line.ID, "", digitaltwin.DirectionOutbound)
	if err != nil || len(none) != 0 {
		t.Fatalf("outbound from the line = %d edges, err %v", len(none), err)
	}
}
