package labs_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// T-M9-37 (DECISIONS.md I-10): under the v6 engine the stored interpretation carries the engine's phase of the lab's
// own date, labelled with that date — not an estimate of the day the interpretation ran.
func TestInterpretation_PhaseOfTheLabDate(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120000971', NOW(), NOW())`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id) //nolint:gosec // test id
	// 28-day cycles; the cycle of 2026-06-20 closes on 2026-07-17 (ovulation ≈ 07-04 → 07-10 is luteal); today
	// 2026-10-06 is day 4 of the 2026-10-03 period (menstruation).
	for _, s := range []string{"2026-06-20", "2026-07-18", "2026-08-15", "2026-09-12", "2026-10-03"} {
		d := civildate.MustParse(s)
		_, err := db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed,
			is_estimated, source, created_at, updated_at) VALUES (?, ?, ?, 5, 1, 0, 'user_logged', NOW(), NOW())`, uid, d, d.AddDays(4))
		require.NoError(t, err)
	}
	q := cyclestore.New(db)
	p, err := state.New(q, nil, clock.Real{}, state.DefaultOptions())
	require.NoError(t, err)
	svc := labs.NewService(labs.Options{DB: db, Cycle: state.Source{P: p, Revs: q}})

	now := time.Date(2026, 10, 6, 10, 0, 0, 0, civildate.Tehran)
	v := 13.2
	labID, err := svc.CreateManual(ctx, uid, labs.Meta{Category: "blood",
		TakenOn: civildate.NullDate{Date: civildate.MustParse("2026-07-10"), Valid: true}},
		[]labs.MarkerInput{{Name: "Hemoglobin", Value: &v, Unit: "g/dL"}}, "en", "fa", now)
	require.NoError(t, err)

	var raw []byte
	require.NoError(t, db.QueryRow(`SELECT interpretation FROM lab_reports WHERE id = ?`, labID).Scan(&raw))
	var in labs.Interpretation
	require.NoError(t, json.Unmarshal(raw, &in))
	assert.Equal(t, "cycle", in.Context.Mode)
	assert.Equal(t, "luteal", in.Context.Phase, "the phase of 2026-07-10, not today's menstruation")
	assert.Equal(t, "2026-07-10", in.Context.PhaseOn)

	// A lab dated before any cycle data has no phase (and no label).
	labID, err = svc.CreateManual(ctx, uid, labs.Meta{Category: "blood",
		TakenOn: civildate.NullDate{Date: civildate.MustParse("2025-01-10"), Valid: true}},
		[]labs.MarkerInput{{Name: "Hemoglobin", Value: &v, Unit: "g/dL"}}, "en", "fa", now)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT interpretation FROM lab_reports WHERE id = ?`, labID).Scan(&raw))
	in = labs.Interpretation{}
	require.NoError(t, json.Unmarshal(raw, &in))
	assert.Empty(t, in.Context.Phase)
	assert.Empty(t, in.Context.PhaseOn)
}
