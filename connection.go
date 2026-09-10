package digitaltwin

import (
	"fmt"
	"strings"
)

// Connection is a directed graph edge between two Assets — DSN-1701
// decision 5. Mirrors how mwanachama-backend-taskmanager models
// depends_on/blocks/subtask_of between Tasks via
// mwanachama-backend-shared's relationship/edge table, once the registry
// is wired onto it (W4).
type Connection struct {
	// ID is the storage-assigned edge identifier.
	ID string `json:"id"`

	// Kind is the edge label — one of the Connection* constants
	// (vocabulary.go).
	Kind ConnectionKind `json:"kind"`

	// FromAssetID is the source asset.
	FromAssetID string `json:"from_asset_id"`

	// ToAssetID is the target asset.
	ToAssetID string `json:"to_asset_id"`

	// Properties are caller-supplied edge metadata.
	Properties map[string]any `json:"properties,omitempty"`

	// CreatedAt is the RFC 3339 timestamp the edge was created.
	CreatedAt string `json:"created_at"`
}

// Validate refuses a malformed connection.
func (c Connection) Validate() error {
	if !IsConnectionKind(c.Kind) {
		return fmt.Errorf("%w: connection kind %q is not one of the current vocabulary", ErrInvalid, c.Kind)
	}
	from, to := strings.TrimSpace(c.FromAssetID), strings.TrimSpace(c.ToAssetID)
	if from == "" || to == "" {
		return fmt.Errorf("%w: a connection names both a from_asset_id and a to_asset_id", ErrInvalid)
	}
	if from == to {
		return fmt.Errorf("%w: an asset cannot connect to itself", ErrInvalid)
	}
	return nil
}
