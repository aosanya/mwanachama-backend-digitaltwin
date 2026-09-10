// retention.go — Sweep (W6): the enforcement mechanism RetentionPolicy
// (telemetry.go) ships configuration for but does not itself act on. Pure
// logic over the two repository interfaces — no new storage — so it works
// against any RegistryRepository/TelemetryRepository pairing, memory or
// Postgres.
//
// Nothing in this package invokes Sweep on a schedule. That's a deployment
// decision (a cron, a periodic goroutine in the gateway process, …) left to
// whoever wires this in — see W9 on the task board.
package digitaltwin

import (
	"context"
	"fmt"
	"time"
)

// Sweep applies every registered RetentionPolicy: for each policy, it
// deletes every reading in the telemetry store older than now minus
// RetentionDays, scoped to that policy's AssetType/MetricName. Returns the
// total number of readings purged across all policies.
//
// A reading not covered by any policy is never purged — "no policy" means
// "keep forever" (RetentionPolicy's doc comment), not "delete on sight".
func Sweep(ctx context.Context, registry RegistryRepository, telemetry TelemetryRepository, now time.Time) (int, error) {
	policies, err := registry.ListRetentionPolicies(ctx)
	if err != nil {
		return 0, fmt.Errorf("Sweep: ListRetentionPolicies: %w", err)
	}

	total := 0
	for _, p := range policies {
		cutoff := now.AddDate(0, 0, -p.RetentionDays)
		purged, err := telemetry.PurgeBefore(ctx, p.AssetType, p.MetricName, cutoff)
		if err != nil {
			return total, fmt.Errorf("Sweep: PurgeBefore(asset_type=%q, metric_name=%q): %w", p.AssetType, p.MetricName, err)
		}
		total += purged
	}
	return total, nil
}
