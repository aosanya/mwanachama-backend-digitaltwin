package digitaltwin_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

var domainWords = []string{
	"pipeline", "transmission", "valve", "station", "compressor", "pump",
	"sensor", "vehicle", "fleet", "depot", "driver", "asset", "utility",
	"telemetry",
}

func TestNoDomainWordsInIdentifiers(t *testing.T) {
	for _, dir := range []string{"models", "routes", "."} {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		seen := map[string]bool{}
		for _, pkg := range pkgs {
			for path, file := range pkg.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					w, bad := carriesDomainWord(id.Name)
					if !bad {
						return true
					}
					key := filepath.Base(path) + ":" + id.Name
					if seen[key] {
						return true
					}
					seen[key] = true
					t.Errorf("%s: identifier %q carries the domain word %q — it belongs in the node document or in a domain's own spec",
						filepath.Base(path), id.Name, w)
					return true
				})
			}
		}
	}
}

func TestNoDomainWordsInStoredVocabularies(t *testing.T) {
	for _, v := range []string{
		string(models.StatusPlanned), string(models.StatusOperational),
		string(models.StatusMaintenance), string(models.StatusFault),
		string(models.StatusDecommissioned),
		string(models.RelationConnectsTo), string(models.RelationPartOf),
		string(models.RelationMonitors),
	} {
		if w, bad := carriesDomainWord(v); bad {
			t.Errorf("stored value %q carries the domain word %q — a value outlives a rename", v, w)
		}
	}
}

func TestNoDomainWordsInTheBlueprintsRolesFieldsAndStoredValues(t *testing.T) {
	b, err := digitaltwin.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	report := func(kind, name string) {
		if w, bad := carriesDomainWord(name); bad {
			t.Errorf("blueprint %s %q carries the domain word %q — it belongs in the document or in a domain's own spec", kind, name, w)
		}
	}
	for _, o := range b.Objects {
		report("role", o.Role)
		if o.Name != "" || o.Table != "" {
			t.Errorf("blueprint role %q names a table, which is the domain's to choose", o.Role)
		}
		for _, f := range o.Fields {
			report("field", o.Role+"."+f.Name)
			for _, v := range f.Values {
				report("stored value", o.Role+"."+f.Name+"="+v)
			}
		}
		for _, idx := range o.Indexes {
			report("index", o.Role+"."+idx.Name)
			if idx.Path != nil {
				report("index path", idx.Path.Path)
			}
		}
	}
}

func TestNoDomainWordsInTheDeclaredRouteTable(t *testing.T) {
	raw := string(digitaltwin.Operations())
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, `"path"`) && !strings.HasPrefix(trimmed, `"action"`) &&
			!strings.Contains(trimmed, `"path":`) && !strings.Contains(trimmed, `"action":`) {
			continue
		}
		for _, field := range []string{`"path":`, `"action":`} {
			at := strings.Index(trimmed, field)
			if at < 0 {
				continue
			}
			rest := trimmed[at+len(field):]
			open := strings.Index(rest, `"`)
			if open < 0 {
				continue
			}
			rest = rest[open+1:]
			end := strings.Index(rest, `"`)
			if end < 0 {
				continue
			}
			if w, bad := carriesDomainWord(rest[:end]); bad {
				t.Errorf("the route table's %s %q carries the domain word %q — an address outlives a rename too", field, rest[:end], w)
			}
		}
	}
}

func carriesDomainWord(s string) (string, bool) {
	low := strings.ToLower(s)
	for _, w := range domainWords {
		if strings.Contains(low, w) {
			return w, true
		}
	}
	return "", false
}
