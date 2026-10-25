package digitaltwin_test

import (
	"errors"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

func TestAbbreviationsGenerateHiddenFormsBothWays(t *testing.T) {
	table := digitaltwin.Abbreviations{"hu": {"utca": "u."}}
	got := models.ExpandLabels([]digitaltwin.Label{
		{Text: "Fő utca", Lang: "hu", Form: models.LabelPreferred},
		{Text: "Kossuth u.", Lang: "hu", Form: models.LabelAlternative},
		{Text: "Main Road", Lang: "en", Form: models.LabelPreferred},
	}, table)
	want := map[string]bool{"fő u": true, "kossuth utca": true}
	hidden := 0
	for _, l := range got {
		if l.Form != models.LabelHidden {
			continue
		}
		hidden++
		if !want[l.Text] || l.Lang != "hu" {
			t.Errorf("unexpected hidden label %+v", l)
		}
	}
	if hidden != len(want) {
		t.Errorf("hidden forms: got %d, want %d (%+v)", hidden, len(want), got)
	}
	if again := models.ExpandLabels(got, table); len(again) != len(got) {
		t.Errorf("expanding twice added %d labels", len(again)-len(got))
	}
}

func TestALeadingNumberIsReadAndTheRestIgnored(t *testing.T) {
	for in, want := range map[string]int{"12": 12, "12/A": 12, " 7b": 7, "103-105": 103} {
		if got, ok := models.LeadingNumber(in); !ok || got != want {
			t.Errorf("%q: got %d %v, want %d", in, got, ok, want)
		}
	}
	if _, ok := models.LeadingNumber("A12"); ok {
		t.Error("a value not starting with a digit has no leading number")
	}
}

func TestANodeIsMatchedByEveryNameItCarries(t *testing.T) {
	tm, ctx := newManagerFor(t, "place")
	labels := models.ExpandLabels([]digitaltwin.Label{
		{Text: "Fő utca", Lang: "hu", Form: models.LabelPreferred},
		{Text: "Main Street", Lang: "en", Form: models.LabelPreferred},
		{Text: "Régi utca", Lang: "hu", Form: models.LabelAlternative},
	}, digitaltwin.Abbreviations{"hu": {"utca": "u."}})
	main, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Fő utca", Kind: "street", Labels: labels})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "Fő utca", Kind: "area"}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Fő utca", "fő u.", "MAIN  street", "Régi u", "régi utca"} {
		got, err := tm.MatchLabel(ctx, "street", text)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != main.ID {
			t.Errorf("%q: got %d matches", text, len(got))
		}
	}
	if got, _ := tm.MatchLabel(ctx, "", "Fő utca"); len(got) != 2 {
		t.Errorf("any kind: got %d, want 2", len(got))
	}
	if got, _ := tm.MatchLabel(ctx, "street", "Nowhere"); len(got) != 0 {
		t.Errorf("unknown name matched %d", len(got))
	}
}

func TestLabelsAreRefusedWhenMalformed(t *testing.T) {
	tm, ctx := newManagerFor(t, "place")
	for name, labels := range map[string][]digitaltwin.Label{
		"two preferred in one language": {{Text: "A", Lang: "hu", Form: models.LabelPreferred}, {Text: "B", Lang: "hu", Form: models.LabelPreferred}},
		"unknown form":                  {{Text: "A", Form: "nickname"}},
		"empty text":                    {{Text: " ", Form: models.LabelAlternative}},
	} {
		if _, err := tm.CreateNode(ctx, digitaltwin.Node{Name: "X", Kind: "street", Labels: labels}); !errors.Is(err, digitaltwin.ErrInvalid) {
			t.Errorf("%s: got %v, want ErrInvalid", name, err)
		}
	}
}
