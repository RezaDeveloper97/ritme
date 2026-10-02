package service_test

// B-N3-14b (N3 stage smoke B-2, deviations.md D-55): a late user who logs bleeding today through the log
// sheet (PUT /logs/days → the period-start reconciliation inserts an unconfirmed row) sees «روز ۱» on the
// home hero (cycle_view) too, not «تأخیر پریود ۱۰ روز».

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/validation"
)

func TestCycleView_BleedingLoggedTodayAnchorsTheHero(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))
	svc := service.New(db, cache.New(nil, false, nil))
	ctx := context.Background()
	const uid = 1010 // overdue_10: last period 2026-08-16, 28-day cycles
	today := civildate.MustParse("2026-09-23")
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

	type view struct {
		CycleView struct {
			CycleDay  *int   `json:"cycle_day"`
			MainPhase string `json:"main_phase"`
			DaysLate  *int   `json:"days_late"`
		} `json:"cycle_view"`
	}
	read := func() view {
		p, err := svc.DayJSON(ctx, uid, today, today, "fa")
		require.NoError(t, err)
		var v view
		require.NoError(t, json.Unmarshal(p.JSON, &v), string(p.JSON))
		return v
	}

	before := read()
	require.NotNil(t, before.CycleView.DaysLate, "overdue before the log")
	assert.Equal(t, 10, *before.CycleView.DaysLate)

	changes, err := taxonomy.Parse(validation.DecodeBody([]byte(`{"categories":{"bleeding":{"flow":"medium"}}}`)), "cycle", "fa", nil, nil)
	require.NoError(t, err)
	_, err = healthlog.NewService(db).SaveDay(ctx, uid, today, changes, "fa", now)
	require.NoError(t, err)

	after := read()
	require.NotNil(t, after.CycleView.CycleDay)
	assert.Equal(t, 1, *after.CycleView.CycleDay, "the logged start is day 1")
	assert.Equal(t, "menstrual", after.CycleView.MainPhase)
	if after.CycleView.DaysLate != nil {
		assert.Zero(t, *after.CycleView.DaysLate, "no longer late")
	}
}
