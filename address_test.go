package digitaltwin_test

import (
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

func TestAnAddressIsMatchedByItsStreetsNamesThenItsLeadingNumber(t *testing.T) {
	tm, ctx := newManagerFor(t, "place")
	labels := models.ExpandLabels([]digitaltwin.Label{
		{Text: "Fő utca", Lang: "hu", Form: models.LabelPreferred},
	}, digitaltwin.Abbreviations{"hu": {"utca": "u."}})
	main, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Fő utca", Kind: "street", Labels: labels})
	if err != nil {
		t.Fatal(err)
	}
	other, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Kossuth utca", Kind: "street"})
	if err != nil {
		t.Fatal(err)
	}
	address := func(name, number string, street digitaltwin.Node) digitaltwin.Node {
		n, err := tm.CreateNode(ctx, digitaltwin.Node{Name: name, Kind: "address", Doc: map[string]any{"house_number": number}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tm.CreateLink(ctx, digitaltwin.Link{FromNodeID: n.ID, ToNodeID: street.ID, Relation: models.RelationPartOf}); err != nil {
			t.Fatal(err)
		}
		return n
	}
	twelve := address("Fő utca 12/A", "12/A", main)
	address("Fő utca 14", "14", main)
	address("Kossuth utca 12", "12", other)

	got, err := tm.MatchAddress(ctx, "street", "Fő u.", "12")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != twelve.ID {
		t.Fatalf("got %+v, want only %s", got, twelve.ID)
	}
	if got, _ := tm.MatchAddress(ctx, "street", "Fő utca", "A12"); len(got) != 0 {
		t.Errorf("a number with no leading digits matched %d addresses", len(got))
	}
}
