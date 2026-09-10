// Package digitaltwin — pre-delivered schema definition (W4).
//
// This file exposes [DefaultDigitalTwinSchema], which returns the fixed
// [schema.Schema] for the asset registry — Asset, MetricDefinition,
// RetentionPolicy. Telemetry readings are deliberately NOT part of this
// schema; see telemetry_postgres.go and the file doc comment on
// telemetry.go for why.
//
// Wiring code in mwanachama-backend-api-gateway seeds this schema at
// startup via SchemaManager.SetSchema — the same pattern
// mwanachama-backend-taskmanager's DefaultWorkSchema uses.
package digitaltwin

import "github.com/aosanya/mwanachama-backend-shared/schema"

// assetEntityTypeID is the TypeDefinition.Name used for Asset entities —
// one type covering all six AssetType values (asset_type is a property,
// not six separate TypeDefinitions), matching how Asset is a single Go
// struct in asset.go.
const assetEntityTypeID = "Asset"

// metricDefinitionEntityTypeID is the TypeDefinition.Name used for
// MetricDefinition entities.
const metricDefinitionEntityTypeID = "MetricDefinition"

// retentionPolicyEntityTypeID is the TypeDefinition.Name used for
// RetentionPolicy entities.
const retentionPolicyEntityTypeID = "RetentionPolicy"

// DefaultDigitalTwinSchema returns the pre-delivered [schema.Schema] seeded
// by mwanachama-backend-api-gateway on startup via SchemaManager.SetSchema.
// The operation is idempotent — calling it multiple times with the same
// schema ID is safe.
func DefaultDigitalTwinSchema() schema.Schema {
	return schema.Schema{
		ID:      "digitaltwin-schema-v1",
		Version: 1,
		Tag:     "v1",
		Types: []schema.TypeDefinition{
			{
				Name:        assetEntityTypeID,
				DisplayName: "Asset",
				Properties: []schema.PropertyDefinition{
					{Name: "name", Type: schema.PropertyTypeString, Required: true},
					{Name: "asset_type", Type: schema.PropertyTypeString, Required: true},
					// station_type is required only when asset_type == station;
					// enforced in Go (Asset.Validate), not by the schema layer.
					{Name: "station_type", Type: schema.PropertyTypeString},
					{Name: "status", Type: schema.PropertyTypeString},
					{Name: "location_lat", Type: schema.PropertyTypeFloat},
					{Name: "location_lng", Type: schema.PropertyTypeFloat},
					{Name: "installed_at", Type: schema.PropertyTypeString},
					{Name: "notes", Type: schema.PropertyTypeString},
					// attributes is a JSON object (map[string]string),
					// expert-configured, no fixed per-type schema — see
					// asset.go's doc comment.
					{Name: "attributes", Type: schema.PropertyTypeString},
					{Name: "created_at", Type: schema.PropertyTypeString},
					{Name: "updated_at", Type: schema.PropertyTypeString},
				},
				Relationships: []schema.RelationshipDefinition{
					{
						Name:    string(ConnectionConnectsTo),
						Label:   "Connects to",
						ToType:  assetEntityTypeID,
						ToMany:  true,
						Inverse: "connected_from",
						Properties: []schema.PropertyDefinition{
							{Name: "created_at", Type: schema.PropertyTypeString},
						},
					},
					{
						Name:    "connected_from",
						Label:   "Connected from",
						ToType:  assetEntityTypeID,
						ToMany:  true,
						Inverse: string(ConnectionConnectsTo),
					},
					{
						Name:    string(ConnectionPartOf),
						Label:   "Part of",
						ToType:  assetEntityTypeID,
						ToMany:  false,
						Inverse: "has_part",
						Properties: []schema.PropertyDefinition{
							{Name: "created_at", Type: schema.PropertyTypeString},
						},
					},
					{
						Name:    "has_part",
						Label:   "Has part",
						ToType:  assetEntityTypeID,
						ToMany:  true,
						Inverse: string(ConnectionPartOf),
					},
					{
						Name:    string(ConnectionMonitors),
						Label:   "Monitors",
						ToType:  assetEntityTypeID,
						ToMany:  true,
						Inverse: "monitored_by",
						Properties: []schema.PropertyDefinition{
							{Name: "created_at", Type: schema.PropertyTypeString},
						},
					},
					{
						Name:    "monitored_by",
						Label:   "Monitored by",
						ToType:  assetEntityTypeID,
						ToMany:  true,
						Inverse: string(ConnectionMonitors),
					},
				},
			},
			{
				Name:        metricDefinitionEntityTypeID,
				DisplayName: "Metric Definition",
				// UniqueKey scopes UpsertMetricDefinition's find-or-update to
				// (type_id, [asset_type, name]) — see the DDL's unique index.
				UniqueKey: []string{"asset_type", "name"},
				Properties: []schema.PropertyDefinition{
					{Name: "name", Type: schema.PropertyTypeString, Required: true},
					{Name: "unit", Type: schema.PropertyTypeString, Required: true},
					{Name: "asset_type", Type: schema.PropertyTypeString},
					{Name: "description", Type: schema.PropertyTypeString},
					{Name: "min_value", Type: schema.PropertyTypeFloat},
					{Name: "max_value", Type: schema.PropertyTypeFloat},
					{Name: "created_at", Type: schema.PropertyTypeString},
					{Name: "updated_at", Type: schema.PropertyTypeString},
				},
			},
			{
				Name:        retentionPolicyEntityTypeID,
				DisplayName: "Retention Policy",
				UniqueKey:   []string{"asset_type", "metric_name"},
				Properties: []schema.PropertyDefinition{
					{Name: "asset_type", Type: schema.PropertyTypeString},
					{Name: "metric_name", Type: schema.PropertyTypeString},
					{Name: "retention_days", Type: schema.PropertyTypeInteger, Required: true},
					{Name: "created_at", Type: schema.PropertyTypeString},
					{Name: "updated_at", Type: schema.PropertyTypeString},
				},
			},
		},
	}
}
