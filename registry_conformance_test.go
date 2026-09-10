package digitaltwin

import (
	"context"
	"errors"
	"testing"
)

// RunRegistryConformance exercises the rules that matter against any
// RegistryRepository implementation — memory or Postgres (once W4 lands).
// Both backends must agree, per this platform's "test both stores" rule —
// see mwanachama-backend-accounting's conformance_test.go for the precedent.
func RunRegistryConformance(t *testing.T, newRepo func() RegistryRepository) {
	t.Helper()
	ctx := context.Background()

	createPipeline := func(t *testing.T, r RegistryRepository, name string) Asset {
		t.Helper()
		a, err := r.CreateAsset(ctx, Asset{Name: name, AssetType: AssetTypePipeline})
		if err != nil {
			t.Fatalf("CreateAsset(%s): %v", name, err)
		}
		return a
	}

	t.Run("CreateAssetDefaultsStatusToPlanned", func(t *testing.T) {
		r := newRepo()
		a := createPipeline(t, r, "Line 1")
		if a.Status != AssetStatusPlanned {
			t.Fatalf("status = %q, want %q", a.Status, AssetStatusPlanned)
		}
		if a.ID == "" || a.CreatedAt == "" || a.UpdatedAt == "" {
			t.Fatalf("CreateAsset did not fill in ID/CreatedAt/UpdatedAt: %+v", a)
		}
	})

	t.Run("CreateAssetRejectsUnknownType", func(t *testing.T) {
		r := newRepo()
		_, err := r.CreateAsset(ctx, Asset{Name: "x", AssetType: "bogus"})
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("want ErrInvalid, got %v", err)
		}
	})

	t.Run("CreateAssetRejectsStationTypeMismatch", func(t *testing.T) {
		r := newRepo()
		if _, err := r.CreateAsset(ctx, Asset{Name: "Station A", AssetType: AssetTypeStation}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("station with no station_type: want ErrInvalid, got %v", err)
		}
		if _, err := r.CreateAsset(ctx, Asset{Name: "Line 1", AssetType: AssetTypePipeline, StationType: StationTypePump}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("non-station with station_type: want ErrInvalid, got %v", err)
		}
	})

	t.Run("ListAssetsFiltersByTypeAndStatus", func(t *testing.T) {
		r := newRepo()
		pipeline := createPipeline(t, r, "Line 1")
		if _, err := r.CreateAsset(ctx, Asset{Name: "Valve 1", AssetType: AssetTypeValve}); err != nil {
			t.Fatal(err)
		}
		if _, err := r.UpdateAssetStatus(ctx, pipeline.ID, AssetStatusOperational); err != nil {
			t.Fatal(err)
		}

		pipelines, err := r.ListAssets(ctx, AssetFilter{AssetType: AssetTypePipeline}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(pipelines) != 1 || pipelines[0].AssetType != AssetTypePipeline {
			t.Fatalf("want 1 pipeline, got %+v", pipelines)
		}

		operational, err := r.ListAssets(ctx, AssetFilter{Status: AssetStatusOperational}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(operational) != 1 || operational[0].ID != pipeline.ID {
			t.Fatalf("want 1 operational asset (the pipeline), got %+v", operational)
		}
	})

	t.Run("UpdateAssetStatusEnforcesStateMachine", func(t *testing.T) {
		r := newRepo()
		a := createPipeline(t, r, "Line 1")

		a, err := r.UpdateAssetStatus(ctx, a.ID, AssetStatusOperational)
		if err != nil {
			t.Fatalf("planned -> operational should succeed: %v", err)
		}
		a, err = r.UpdateAssetStatus(ctx, a.ID, AssetStatusDecommissioned)
		if err != nil {
			t.Fatalf("operational -> decommissioned should succeed: %v", err)
		}
		if _, err := r.UpdateAssetStatus(ctx, a.ID, AssetStatusOperational); !errors.Is(err, ErrInvalid) {
			t.Fatalf("decommissioned is terminal: want ErrInvalid, got %v", err)
		}
	})

	t.Run("UpdateAssetLeavesTypeAndStatusUnchanged", func(t *testing.T) {
		r := newRepo()
		a := createPipeline(t, r, "Line 1")

		updated, err := r.UpdateAsset(ctx, Asset{ID: a.ID, Name: "Line 1 (renamed)", Notes: "recoated 2026"})
		if err != nil {
			t.Fatal(err)
		}
		if updated.AssetType != AssetTypePipeline || updated.Status != AssetStatusPlanned {
			t.Fatalf("UpdateAsset changed immutable fields: %+v", updated)
		}
		if updated.Name != "Line 1 (renamed)" || updated.Notes != "recoated 2026" {
			t.Fatalf("UpdateAsset did not apply mutable fields: %+v", updated)
		}
	})

	t.Run("CreateConnectionIsIdempotentAndValidatesEndpoints", func(t *testing.T) {
		r := newRepo()
		pipeline := createPipeline(t, r, "Line 1")
		valve, err := r.CreateAsset(ctx, Asset{Name: "Valve 1", AssetType: AssetTypeValve})
		if err != nil {
			t.Fatal(err)
		}

		c1, err := r.CreateConnection(ctx, Connection{Kind: ConnectionPartOf, FromAssetID: valve.ID, ToAssetID: pipeline.ID})
		if err != nil {
			t.Fatalf("CreateConnection: %v", err)
		}
		c2, err := r.CreateConnection(ctx, Connection{Kind: ConnectionPartOf, FromAssetID: valve.ID, ToAssetID: pipeline.ID})
		if err != nil {
			t.Fatalf("re-creating the same connection should be idempotent: %v", err)
		}
		if c1.ID != c2.ID {
			t.Fatalf("re-creating the same connection minted a second edge: %s != %s", c1.ID, c2.ID)
		}
	})

	t.Run("CreateConnectionRejectsUnknownAsset", func(t *testing.T) {
		r := newRepo()
		pipeline := createPipeline(t, r, "Line 1")
		_, err := r.CreateConnection(ctx, Connection{Kind: ConnectionPartOf, FromAssetID: "does-not-exist", ToAssetID: pipeline.ID})
		if !errors.Is(err, ErrInvalidConnection) {
			t.Fatalf("want ErrInvalidConnection, got %v", err)
		}
	})

	t.Run("DeleteConnectionRemovesEdge", func(t *testing.T) {
		r := newRepo()
		pipeline := createPipeline(t, r, "Line 1")
		valve, err := r.CreateAsset(ctx, Asset{Name: "Valve 1", AssetType: AssetTypeValve})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.CreateConnection(ctx, Connection{Kind: ConnectionPartOf, FromAssetID: valve.ID, ToAssetID: pipeline.ID}); err != nil {
			t.Fatal(err)
		}
		if err := r.DeleteConnection(ctx, valve.ID, pipeline.ID, ConnectionPartOf); err != nil {
			t.Fatalf("DeleteConnection: %v", err)
		}
		if err := r.DeleteConnection(ctx, valve.ID, pipeline.ID, ConnectionPartOf); !errors.Is(err, ErrConnectionNotFound) {
			t.Fatalf("deleting again: want ErrConnectionNotFound, got %v", err)
		}
	})

	t.Run("ListConnectionsRespectsDirection", func(t *testing.T) {
		r := newRepo()
		pipeline := createPipeline(t, r, "Line 1")
		valve, err := r.CreateAsset(ctx, Asset{Name: "Valve 1", AssetType: AssetTypeValve})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.CreateConnection(ctx, Connection{Kind: ConnectionPartOf, FromAssetID: valve.ID, ToAssetID: pipeline.ID}); err != nil {
			t.Fatal(err)
		}

		outbound, err := r.ListConnections(ctx, valve.ID, "", DirectionOutbound)
		if err != nil {
			t.Fatal(err)
		}
		if len(outbound) != 1 {
			t.Fatalf("valve outbound: want 1, got %d", len(outbound))
		}

		inbound, err := r.ListConnections(ctx, pipeline.ID, "", DirectionInbound)
		if err != nil {
			t.Fatal(err)
		}
		if len(inbound) != 1 || inbound[0].FromAssetID != valve.ID {
			t.Fatalf("pipeline inbound: want 1 edge from the valve, got %+v", inbound)
		}
	})

	t.Run("UpsertMetricDefinitionFindsOrUpdates", func(t *testing.T) {
		r := newRepo()
		first, err := r.UpsertMetricDefinition(ctx, MetricDefinition{
			Name: "pressure_psi", Unit: "psi", AssetType: AssetTypePipeline,
		})
		if err != nil {
			t.Fatal(err)
		}
		second, err := r.UpsertMetricDefinition(ctx, MetricDefinition{
			Name: "pressure_psi", Unit: "psi", AssetType: AssetTypePipeline, Description: "line pressure",
		})
		if err != nil {
			t.Fatal(err)
		}
		if first.ID != second.ID {
			t.Fatalf("upserting the same (asset_type, name) minted a second definition: %s != %s", first.ID, second.ID)
		}
		if second.Description != "line pressure" {
			t.Fatalf("upsert did not apply the update: %+v", second)
		}

		defs, err := r.ListMetricDefinitions(ctx, AssetTypePipeline)
		if err != nil {
			t.Fatal(err)
		}
		if len(defs) != 1 {
			t.Fatalf("want 1 definition, got %d", len(defs))
		}
	})

	t.Run("SetRetentionPolicyUpserts", func(t *testing.T) {
		r := newRepo()
		first, err := r.SetRetentionPolicy(ctx, RetentionPolicy{AssetType: AssetTypePipeline, MetricName: "pressure_psi", RetentionDays: 90})
		if err != nil {
			t.Fatal(err)
		}
		second, err := r.SetRetentionPolicy(ctx, RetentionPolicy{AssetType: AssetTypePipeline, MetricName: "pressure_psi", RetentionDays: 365})
		if err != nil {
			t.Fatal(err)
		}
		if first.ID != second.ID || second.RetentionDays != 365 {
			t.Fatalf("SetRetentionPolicy did not upsert in place: %+v -> %+v", first, second)
		}
	})
}
