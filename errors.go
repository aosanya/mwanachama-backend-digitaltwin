package digitaltwin

import "errors"

// ErrInvalid is returned by Validate and by any write that would violate it.
var ErrInvalid = errors.New("digitaltwin: invalid")

// ErrAssetNotFound is returned when an asset id names nothing.
var ErrAssetNotFound = errors.New("digitaltwin: asset not found")

// ErrConnectionNotFound is returned when a connection lookup matches no edge.
var ErrConnectionNotFound = errors.New("digitaltwin: connection not found")

// ErrInvalidConnection is returned when a connection's endpoints don't
// resolve to real assets in the registry.
var ErrInvalidConnection = errors.New("digitaltwin: invalid connection")

// ErrMetricDefinitionNotFound is returned when a metric definition id or
// (asset type, name) lookup matches nothing.
var ErrMetricDefinitionNotFound = errors.New("digitaltwin: metric definition not found")

// ErrRetentionPolicyNotFound is returned when a retention policy id names
// nothing.
var ErrRetentionPolicyNotFound = errors.New("digitaltwin: retention policy not found")

// ErrReadingNotFound is returned by Latest when no reading has been
// recorded yet for the given asset/metric.
var ErrReadingNotFound = errors.New("digitaltwin: no reading found")
