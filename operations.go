package digitaltwin

import _ "embed"

//go:embed digitaltwin.operations.json
var operationsJSON []byte

func Operations() []byte { return operationsJSON }
