package digitaltwin

import "github.com/aosanya/mwanachama-backend-shared/spec"

func isName(s string) bool { return spec.NamePattern.MatchString(s) }

var patterns = map[string]func(string) bool{
	"name": isName,
}
