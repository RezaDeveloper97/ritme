package accountexport

// The one table the export reads without sqlc: user_profiles_dedupe_backup (00096) is created through PREPARE on
// purpose — "an operational record, not app schema" — so sqlc never sees it and cannot generate a query for it. The
// statement is a constant with an explicit column list (nothing built from input; the owner's id is bound).

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

const dedupeBackupTable = "user_profiles_dedupe_backup"

var dedupeBackupColumns = []string{"id", "user_id", "birthday", "weight", "height", "period_duration", "cycle_duration",
	"last_period_start", "user_goal", "pregnancy_intention", "chronic_conditions", "subscription_type", "calculation_status",
	"calculation_started_at", "calculation_completed_at", "calculation_version", "created_at", "updated_at", "removed_at"}

const dedupeBackupSQL = "SELECT `id`, `user_id`, `birthday`, `weight`, `height`, `period_duration`, `cycle_duration`, " +
	"`last_period_start`, `user_goal`, `pregnancy_intention`, `chronic_conditions`, `subscription_type`, `calculation_status`, " +
	"`calculation_started_at`, `calculation_completed_at`, `calculation_version`, `created_at`, `updated_at`, `removed_at` " +
	"FROM `user_profiles_dedupe_backup` WHERE user_id = ? ORDER BY id"

type dedupeBackupRow struct {
	ID                   uint64
	UserID               uint64
	Birthday             civildate.NullDate
	Weight               sql.NullString
	Height               sql.NullInt64
	PeriodDuration       sql.NullInt64
	CycleDuration        sql.NullInt64
	LastPeriodStart      civildate.NullDate
	UserGoal             string
	PregnancyIntention   sql.NullString
	ChronicConditions    sql.Null[json.RawMessage]
	SubscriptionType     string
	CalculationStatus    string
	CalculationStartedAt sql.NullTime
	CalculationCompleted sql.NullTime
	CalculationVersion   int64
	CreatedAt            sql.NullTime
	UpdatedAt            sql.NullTime
	RemovedAt            sql.NullTime
}

func dedupeBackupRows(ctx context.Context, db *sql.DB, uid uint64) ([]string, []any, error) {
	rs, err := db.QueryContext(ctx, dedupeBackupSQL, uid)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rs.Close() }()
	out := []any{}
	for rs.Next() {
		var r dedupeBackupRow
		if err := rs.Scan(&r.ID, &r.UserID, &r.Birthday, &r.Weight, &r.Height, &r.PeriodDuration, &r.CycleDuration,
			&r.LastPeriodStart, &r.UserGoal, &r.PregnancyIntention, &r.ChronicConditions, &r.SubscriptionType,
			&r.CalculationStatus, &r.CalculationStartedAt, &r.CalculationCompleted, &r.CalculationVersion,
			&r.CreatedAt, &r.UpdatedAt, &r.RemovedAt); err != nil {
			return nil, nil, err
		}
		out = append(out, r)
	}
	return dedupeBackupColumns, out, rs.Err()
}
