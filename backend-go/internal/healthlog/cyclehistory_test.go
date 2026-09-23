package healthlog

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestSpottingWarningWindow(t *testing.T) {
	profile := &store.UserProfile{
		LastPeriodStart: civildate.NullDate{Date: civildate.MustParse("2026-09-01"), Valid: true},
		CycleDuration:   sql.NullInt16{Int16: 30, Valid: true},
	}
	logOn := func(date string, spotting bool, bleeding string) *model.DailyHealthLog {
		l := model.FromRow(store.DailyHealthLog{LogDate: civildate.MustParse(date)})
		if spotting {
			require.NoError(t, l.Set("spotting", true))
		}
		if bleeding != "" {
			require.NoError(t, l.Set("bleeding_intensity", bleeding))
		}
		return l
	}
	cases := []struct {
		date     string
		spotting bool
		bleeding string
		want     bool
	}{
		{"2026-09-14", true, "", false},    // day 14
		{"2026-09-15", true, "", true},     // day 15
		{"2026-09-28", false, "low", true}, // day 28 = 30 − 2
		{"2026-09-29", true, "", false},    // day 29
		{"2026-09-20", false, "medium", false},
		{"2026-08-20", true, "", false}, // before the LMP (negative day)
	}
	for _, c := range cases {
		got := SpottingWarning(profile, logOn(c.date, c.spotting, c.bleeding), "en")
		assert.Equal(t, c.want, got != nil, c.date)
	}
	assert.Nil(t, SpottingWarning(nil, logOn("2026-09-20", true, ""), "en"))

	profile.CycleDuration = sql.NullInt16{} // ?? 28 → window 15..26
	assert.Nil(t, SpottingWarning(profile, logOn("2026-09-27", true, ""), "fa"))
	w := SpottingWarning(profile, logOn("2026-09-26", true, ""), "fa")
	require.NotNil(t, w)
	msg, _ := w.Get("message")
	assert.Equal(t, spottingMessageFa, msg)
	w = SpottingWarning(profile, logOn("2026-09-26", true, ""), "ar")
	msg, _ = w.Get("message")
	assert.Equal(t, spottingMessageEn, msg)
}

func TestCopyFieldsFillsEveryParam(t *testing.T) {
	row := store.DailyHealthLog{ID: 9, UserID: 1, LogDate: civildate.MustParse("2026-09-23"),
		Notes: sql.NullString{String: "n", Valid: true}}
	var ins store.InsertDailyHealthLogParams
	var upd store.UpdateDailyHealthLogParams
	assert.NotPanics(t, func() { copyFields(&ins, row) })
	assert.NotPanics(t, func() { copyFields(&upd, row) })
	assert.Equal(t, "n", ins.Notes.String)
	assert.Equal(t, uint64(9), upd.ID)
	assert.Equal(t, row.LogDate, upd.LogDate)
}
