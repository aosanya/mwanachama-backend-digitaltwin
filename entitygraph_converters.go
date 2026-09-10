// entitygraph_converters.go — entity <-> domain converters for the asset
// registry (W4). Property helpers (StringProp, Float64Prop, …) live in
// [github.com/aosanya/mwanachama-backend-shared/entitygraph] and are used
// directly, the same convention mwanachama-backend-taskmanager's
// task_impl_converters.go follows.
package digitaltwin

import (
	"encoding/json"
	"time"

	"github.com/aosanya/mwanachama-backend-shared/entitygraph"
)

// ── Asset ────────────────────────────────────────────────────────────────

// assetToProperties serialises an Asset into the property map stored on its
// entitygraph Entity. Attributes is JSON-encoded into a single string
// property — entitygraph has no generic string-map property type, the same
// reason taskmanager JSON-encodes DirectionHistory.
func assetToProperties(a Asset) map[string]any {
	props := map[string]any{
		"name":         a.Name,
		"asset_type":   string(a.AssetType),
		"station_type": string(a.StationType),
		"status":       string(a.Status),
		"installed_at": a.InstalledAt,
		"notes":        a.Notes,
		"created_at":   a.CreatedAt,
		"updated_at":   a.UpdatedAt,
	}
	if a.Location != nil {
		props["location_lat"] = a.Location.Lat
		props["location_lng"] = a.Location.Lng
	}
	if len(a.Attributes) > 0 {
		if b, err := json.Marshal(a.Attributes); err == nil {
			props["attributes"] = string(b)
		}
	}
	return props
}

// assetFromEntity reconstructs an Asset from an entitygraph Entity.
func assetFromEntity(e entitygraph.Entity) Asset {
	a := Asset{
		ID:          e.ID,
		Name:        entitygraph.StringProp(e.Properties, "name"),
		AssetType:   AssetType(entitygraph.StringProp(e.Properties, "asset_type")),
		StationType: StationType(entitygraph.StringProp(e.Properties, "station_type")),
		Status:      AssetStatus(entitygraph.StringProp(e.Properties, "status")),
		InstalledAt: entitygraph.StringProp(e.Properties, "installed_at"),
		Notes:       entitygraph.StringProp(e.Properties, "notes"),
		CreatedAt:   entitygraph.StringProp(e.Properties, "created_at"),
		UpdatedAt:   entitygraph.StringProp(e.Properties, "updated_at"),
	}
	if _, ok := e.Properties["location_lat"]; ok {
		a.Location = &Location{
			Lat: entitygraph.Float64Prop(e.Properties, "location_lat"),
			Lng: entitygraph.Float64Prop(e.Properties, "location_lng"),
		}
	}
	if raw := entitygraph.StringProp(e.Properties, "attributes"); raw != "" {
		var attrs map[string]string
		if err := json.Unmarshal([]byte(raw), &attrs); err == nil {
			a.Attributes = attrs
		}
	}
	return a
}

// ── MetricDefinition ─────────────────────────────────────────────────────

func metricDefinitionToProperties(m MetricDefinition) map[string]any {
	props := map[string]any{
		"name":        m.Name,
		"unit":        m.Unit,
		"asset_type":  string(m.AssetType),
		"description": m.Description,
		"created_at":  m.CreatedAt,
		"updated_at":  m.UpdatedAt,
	}
	if m.MinValue != nil {
		props["min_value"] = *m.MinValue
	}
	if m.MaxValue != nil {
		props["max_value"] = *m.MaxValue
	}
	return props
}

func metricDefinitionFromEntity(e entitygraph.Entity) MetricDefinition {
	m := MetricDefinition{
		ID:          e.ID,
		Name:        entitygraph.StringProp(e.Properties, "name"),
		Unit:        entitygraph.StringProp(e.Properties, "unit"),
		AssetType:   AssetType(entitygraph.StringProp(e.Properties, "asset_type")),
		Description: entitygraph.StringProp(e.Properties, "description"),
		CreatedAt:   entitygraph.StringProp(e.Properties, "created_at"),
		UpdatedAt:   entitygraph.StringProp(e.Properties, "updated_at"),
	}
	if _, ok := e.Properties["min_value"]; ok {
		v := entitygraph.Float64Prop(e.Properties, "min_value")
		m.MinValue = &v
	}
	if _, ok := e.Properties["max_value"]; ok {
		v := entitygraph.Float64Prop(e.Properties, "max_value")
		m.MaxValue = &v
	}
	return m
}

// ── RetentionPolicy ──────────────────────────────────────────────────────

func retentionPolicyToProperties(p RetentionPolicy) map[string]any {
	return map[string]any{
		"asset_type":     string(p.AssetType),
		"metric_name":    p.MetricName,
		"retention_days": p.RetentionDays,
		"created_at":     p.CreatedAt,
		"updated_at":     p.UpdatedAt,
	}
}

func retentionPolicyFromEntity(e entitygraph.Entity) RetentionPolicy {
	return RetentionPolicy{
		ID:            e.ID,
		AssetType:     AssetType(entitygraph.StringProp(e.Properties, "asset_type")),
		MetricName:    entitygraph.StringProp(e.Properties, "metric_name"),
		RetentionDays: int(entitygraph.Int64Prop(e.Properties, "retention_days")),
		CreatedAt:     entitygraph.StringProp(e.Properties, "created_at"),
		UpdatedAt:     entitygraph.StringProp(e.Properties, "updated_at"),
	}
}

// ── Connection ───────────────────────────────────────────────────────────

func connectionFromEntitygraph(r entitygraph.Relationship) Connection {
	props := r.Properties
	if props != nil {
		dup := make(map[string]any, len(props))
		for k, v := range props {
			dup[k] = v
		}
		props = dup
	}
	return Connection{
		ID:          r.ID,
		Kind:        ConnectionKind(r.Name),
		FromAssetID: r.FromID,
		ToAssetID:   r.ToID,
		Properties:  props,
		CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
	}
}
