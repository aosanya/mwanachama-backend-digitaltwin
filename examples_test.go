package digitaltwin_test

import (
	"path/filepath"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
	"github.com/aosanya/mwanachama-backend-digitaltwin/routes"
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func shippedExamples(t *testing.T) []string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(".", "spec", "examples", "*.digitaltwin.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(found) < 2 {
		t.Fatalf("the module ships %d domain specs, and domain-neutrality is only exercised by at least two", len(found))
	}
	return found
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, path := range shippedExamples(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			s, err := digitaltwin.LoadSpec(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			db := newDB(t)
			if err := spec.Migrate(db, s); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			if _, err := digitaltwin.NewTwinManager(db, s); err != nil {
				t.Fatalf("the spec and the Go types disagree: %v", err)
			}
		})
	}
}

func TestTwoDomainsCoexist(t *testing.T) {
	db := newDB(t)
	for _, path := range shippedExamples(t) {
		s, err := digitaltwin.LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		if err := spec.Migrate(db, s); err != nil {
			t.Fatalf("migrate %s into a database another domain already occupies: %v", path, err)
		}
	}

	utility, err := digitaltwin.LoadSpec(examplePath("utility"))
	if err != nil {
		t.Fatalf("load utility: %v", err)
	}
	fleet, err := digitaltwin.LoadSpec(examplePath("fleet"))
	if err != nil {
		t.Fatalf("load fleet: %v", err)
	}

	utilityNode, _ := utility.ByRole("node")
	fleetNode, _ := fleet.ByRole("node")
	if utility.TableFor(utilityNode) == fleet.TableFor(fleetNode) {
		t.Fatalf("two domains landed their node in one table: %s", utility.TableFor(utilityNode))
	}

	utilityManager, err := digitaltwin.NewTwinManager(db, utility)
	if err != nil {
		t.Fatalf("utility manager: %v", err)
	}
	fleetManager, err := digitaltwin.NewTwinManager(db, fleet)
	if err != nil {
		t.Fatalf("fleet manager: %v", err)
	}

	ctx := t.Context()
	if _, err := utilityManager.CreateNode(ctx, digitaltwin.Node{Name: "Line 4", Kind: "pipeline"}); err != nil {
		t.Fatalf("utility write: %v", err)
	}
	if _, err := fleetManager.CreateNode(ctx, digitaltwin.Node{Name: "KDA 001A", Kind: "van"}); err != nil {
		t.Fatalf("fleet write: %v", err)
	}

	held, err := fleetManager.ListNodes(ctx, digitaltwin.NodeFilter{})
	if err != nil {
		t.Fatalf("fleet read: %v", err)
	}
	if len(held) != 1 || held[0].Kind != "van" {
		t.Fatalf("the fleet sees %d nodes, and one domain is reading another's table: %+v", len(held), held)
	}
}

func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	b, err := digitaltwin.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if f.Required && f.Default != "" {
				t.Errorf("%s.%s is required and defaulted to %q — the default is exactly what lets an omitted value pass unnoticed",
					o.Role, f.Name, f.Default)
			}
		}
	}

	for _, path := range shippedExamples(t) {
		s, err := digitaltwin.LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		for _, o := range s.Objects {
			for _, f := range o.Fields {
				if f.Required && f.Default != "" {
					t.Errorf("%s: %s.%s is required and defaulted to %q", filepath.Base(path), o.Name, f.Name, f.Default)
				}
			}
		}
	}
}

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	b, err := digitaltwin.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}

	held := map[string][]string{
		"node.status":   {string(models.StatusPlanned), string(models.StatusOperational), string(models.StatusMaintenance), string(models.StatusFault), string(models.StatusDecommissioned)},
		"link.relation": {string(models.RelationConnectsTo), string(models.RelationPartOf), string(models.RelationMonitors)},
	}

	declared := map[string][]string{}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if len(f.Values) > 0 {
				declared[o.Role+"."+f.Name] = f.Values
			}
		}
	}

	for at, values := range declared {
		if _, ok := held[at]; !ok {
			t.Errorf("the blueprint declares values for %s and models/vocabulary.go holds none", at)
			continue
		}
		for _, v := range values {
			if !contains(held[at], v) {
				t.Errorf("the blueprint declares %s = %q and Go holds no constant for it", at, v)
			}
		}
	}
	for at, values := range held {
		if _, ok := declared[at]; !ok {
			t.Errorf("models/vocabulary.go holds constants for %s and the blueprint declares no values there", at)
			continue
		}
		for _, v := range values {
			if !contains(declared[at], v) {
				t.Errorf("Go holds %s = %q and the blueprint does not declare it", at, v)
			}
		}
	}
}

func TestEverySentinelIsMappedToAStatus(t *testing.T) {
	s, err := dispatch.Parse(digitaltwin.Operations())
	if err != nil {
		t.Fatalf("parse operations: %v", err)
	}
	for _, problem := range dispatch.UnmappedSentinels(s, routes.Sentinels, ".") {
		t.Error(problem)
	}
}

func contains(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
