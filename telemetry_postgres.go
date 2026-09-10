// telemetry_postgres.go — PostgresTelemetryRepository (W5): a dedicated,
// service-owned Postgres table for TelemetryReading, deliberately separate
// from mwanachama-backend-shared's entity-graph store — see telemetry.go's
// doc comment for why. Uses database/sql via the pgx stdlib driver
// directly, the same convention mwanachama-backend-shared/postgres uses,
// rather than depending on that package (this table has nothing to do with
// the entity-graph engine).
package digitaltwin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DefaultTelemetryTable is the table name a PostgresTelemetryRepository
// uses when none is given explicitly.
const DefaultTelemetryTable = "digitaltwin_telemetry_readings"

// TelemetryDDL returns the CREATE TABLE / CREATE INDEX statements for the
// dedicated telemetry table, named table. Each deployment embeds this into
// its own golang-migrate up-migration — mirroring how
// mwanachama-backend-shared/postgres.DDL works for the entity-graph tables,
// but this table has no relationship to that schema at all.
//
// No time-based partitioning yet — open question 2 in
// documentation/2. design/README.md: the ingestion frequency ceiling was
// never quantified, so this starts as a single indexed table. Partition by
// RecordedAt if/when real volume shows it's needed (W5 follow-up).
func TelemetryDDL(table string) string {
	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %[1]s (
    id           TEXT PRIMARY KEY,
    asset_id     TEXT NOT NULL,
    asset_type   TEXT NOT NULL,
    metric_name  TEXT NOT NULL,
    value        DOUBLE PRECISION NOT NULL,
    unit         TEXT NOT NULL DEFAULT '',
    recorded_at  TIMESTAMPTZ NOT NULL,
    ingested_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    quality      TEXT NOT NULL DEFAULT ''
);

-- Backs Query/Latest: filter by (asset, metric), ordered by time.
CREATE INDEX IF NOT EXISTS %[1]s_asset_metric_time_idx
    ON %[1]s (asset_id, metric_name, recorded_at);

-- Backs LatestByAsset: filter by asset, ordered by time.
CREATE INDEX IF NOT EXISTS %[1]s_asset_time_idx
    ON %[1]s (asset_id, recorded_at);

-- Backs PurgeBefore: scoped by (asset_type, metric), bounded by time.
CREATE INDEX IF NOT EXISTS %[1]s_type_metric_time_idx
    ON %[1]s (asset_type, metric_name, recorded_at);
`, table)
}

// TelemetryDropDDL returns the DROP TABLE statement undoing [TelemetryDDL].
func TelemetryDropDDL(table string) string {
	return fmt.Sprintf(`DROP TABLE IF EXISTS %[1]s;`, table)
}

// PostgresTelemetryRepository is the dedicated-table-backed
// TelemetryRepository. Construct with an open *sql.DB (pgx stdlib driver)
// and a table already created via [TelemetryDDL].
type PostgresTelemetryRepository struct {
	db    *sql.DB
	table string
}

// NewPostgresTelemetryRepository constructs a TelemetryRepository over db,
// reading and writing table. An empty table defaults to
// [DefaultTelemetryTable]. Does not verify the table exists — run a
// migration built from [TelemetryDDL] first.
func NewPostgresTelemetryRepository(db *sql.DB, table string) *PostgresTelemetryRepository {
	if table == "" {
		table = DefaultTelemetryTable
	}
	return &PostgresTelemetryRepository{db: db, table: table}
}

// Record implements TelemetryRepository.
func (r *PostgresTelemetryRepository) Record(ctx context.Context, reading TelemetryReading) (TelemetryReading, error) {
	out, err := r.RecordBatch(ctx, []TelemetryReading{reading})
	if err != nil {
		return TelemetryReading{}, err
	}
	return out[0], nil
}

// RecordBatch implements TelemetryRepository. All-or-nothing: every reading
// is validated before any is inserted, and the inserts themselves run in
// one transaction.
func (r *PostgresTelemetryRepository) RecordBatch(ctx context.Context, readings []TelemetryReading) ([]TelemetryReading, error) {
	for _, reading := range readings {
		if err := reading.Validate(); err != nil {
			return nil, err
		}
	}
	if len(readings) == 0 {
		return nil, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("RecordBatch: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, asset_id, asset_type, metric_name, value, unit, recorded_at, ingested_at, quality)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, r.table))
	if err != nil {
		return nil, fmt.Errorf("RecordBatch: prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	out := make([]TelemetryReading, len(readings))
	for i, reading := range readings {
		reading.ID = uuid.NewString()
		reading.IngestedAt = now
		if _, err := stmt.ExecContext(ctx, reading.ID, reading.AssetID, string(reading.AssetType),
			reading.MetricName, reading.Value, reading.Unit, reading.RecordedAt, reading.IngestedAt, reading.Quality); err != nil {
			return nil, fmt.Errorf("RecordBatch: insert: %w", err)
		}
		out[i] = reading
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("RecordBatch: commit: %w", err)
	}
	return out, nil
}

func scanReading(row interface{ Scan(...any) error }) (TelemetryReading, error) {
	var r TelemetryReading
	var assetType string
	err := row.Scan(&r.ID, &r.AssetID, &assetType, &r.MetricName,
		&r.Value, &r.Unit, &r.RecordedAt, &r.IngestedAt, &r.Quality)
	r.AssetType = AssetType(assetType)
	return r, err
}

// Query implements TelemetryRepository.
func (r *PostgresTelemetryRepository) Query(ctx context.Context, filter TelemetryFilter) ([]TelemetryReading, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultPage
	}

	var conds []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.AssetID != "" {
		conds = append(conds, "asset_id = "+arg(filter.AssetID))
	}
	if filter.MetricName != "" {
		conds = append(conds, "metric_name = "+arg(filter.MetricName))
	}
	if !filter.From.IsZero() {
		conds = append(conds, "recorded_at >= "+arg(filter.From))
	}
	if !filter.To.IsZero() {
		conds = append(conds, "recorded_at <= "+arg(filter.To))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	query := fmt.Sprintf(
		`SELECT id, asset_id, asset_type, metric_name, value, unit, recorded_at, ingested_at, quality
		 FROM %s %s ORDER BY recorded_at ASC LIMIT %s`, r.table, where, arg(limit))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Query: %w", err)
	}
	defer rows.Close()

	out := make([]TelemetryReading, 0)
	for rows.Next() {
		reading, err := scanReading(rows)
		if err != nil {
			return nil, fmt.Errorf("Query: scan: %w", err)
		}
		out = append(out, reading)
	}
	return out, rows.Err()
}

// Latest implements TelemetryRepository.
func (r *PostgresTelemetryRepository) Latest(ctx context.Context, assetID, metricName string) (TelemetryReading, error) {
	query := fmt.Sprintf(
		`SELECT id, asset_id, asset_type, metric_name, value, unit, recorded_at, ingested_at, quality
		 FROM %s WHERE asset_id = $1 AND metric_name = $2
		 ORDER BY recorded_at DESC LIMIT 1`, r.table)
	reading, err := scanReading(r.db.QueryRowContext(ctx, query, assetID, metricName))
	if err != nil {
		if err == sql.ErrNoRows {
			return TelemetryReading{}, ErrReadingNotFound
		}
		return TelemetryReading{}, fmt.Errorf("Latest: %w", err)
	}
	return reading, nil
}

// LatestByAsset implements TelemetryRepository. Uses Postgres's
// SELECT DISTINCT ON to pick the newest row per metric_name in one query.
func (r *PostgresTelemetryRepository) LatestByAsset(ctx context.Context, assetID string) ([]TelemetryReading, error) {
	query := fmt.Sprintf(
		`SELECT DISTINCT ON (metric_name) id, asset_id, asset_type, metric_name, value, unit, recorded_at, ingested_at, quality
		 FROM %s WHERE asset_id = $1
		 ORDER BY metric_name, recorded_at DESC`, r.table)
	rows, err := r.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, fmt.Errorf("LatestByAsset: %w", err)
	}
	defer rows.Close()

	out := make([]TelemetryReading, 0)
	for rows.Next() {
		reading, err := scanReading(rows)
		if err != nil {
			return nil, fmt.Errorf("LatestByAsset: scan: %w", err)
		}
		out = append(out, reading)
	}
	return out, rows.Err()
}

// PurgeBefore implements TelemetryRepository.
func (r *PostgresTelemetryRepository) PurgeBefore(ctx context.Context, assetType AssetType, metricName string, cutoff time.Time) (int, error) {
	conds := []string{"recorded_at < $1"}
	args := []any{cutoff}
	if assetType != "" {
		args = append(args, string(assetType))
		conds = append(conds, fmt.Sprintf("asset_type = $%d", len(args)))
	}
	if metricName != "" {
		args = append(args, metricName)
		conds = append(conds, fmt.Sprintf("metric_name = $%d", len(args)))
	}
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s`, r.table, strings.Join(conds, " AND "))
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("PurgeBefore: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("PurgeBefore: rows affected: %w", err)
	}
	return int(n), nil
}

var _ TelemetryRepository = (*PostgresTelemetryRepository)(nil)
