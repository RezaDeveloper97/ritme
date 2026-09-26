package calendar

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

func dating(t *testing.T, p *store.PregnancyProfile, today string) v2.Dating {
	t.Helper()
	d, ok := v2.Resolve(p, civildate.MustParse(today))
	require.True(t, ok)
	return d
}

func lmp(date string) *store.PregnancyProfile {
	return &store.PregnancyProfile{PregnancyMode: true, AgeSource: sql.NullString{String: "lmp", Valid: true},
		LmpDate: civildate.NullDate{Date: civildate.MustParse(date), Valid: true}}
}

func ultrasound(date string, w, dd int32) *store.PregnancyProfile {
	return &store.PregnancyProfile{PregnancyMode: true, AgeSource: sql.NullString{String: "ultrasound", Valid: true},
		UltrasoundDate:  civildate.NullDate{Date: civildate.MustParse(date), Valid: true},
		UltrasoundWeeks: sql.NullInt32{Int32: w, Valid: true}, UltrasoundDays: sql.NullInt32{Int32: dd, Valid: true}}
}

func visit(id uint64, date, key, stage string) Visit {
	d := civildate.MustParse(date)
	return Visit{ID: id, Date: d, At: d.TehranMidnight().Add(10 * time.Hour), CareItemKey: key, Stage: stage}
}

func TestResolveItem_States(t *testing.T) {
	d := dating(t, lmp("2026-07-01"), "2026-09-23") // week 13
	// NT scan window weeks 11–14: 2026-09-09 .. 2026-10-06.
	st := ResolveItem(11, 14, "nt", nil, d)
	assert.Equal(t, StateToBook, st.State)
	assert.Equal(t, "2026-09-09", st.From.String())
	assert.Equal(t, "2026-10-06", st.To.String())
	require.NotNil(t, st.Suggested)
	assert.Equal(t, "2026-09-23", st.Suggested.String(), "open window → today")

	future := ResolveItem(20, 22, "anomaly", nil, d)
	assert.Equal(t, "2026-11-11", future.Suggested.String(), "future window → window start")

	past := ResolveItem(6, 8, "first", nil, d)
	assert.Nil(t, past.Suggested, "ended window → no suggestion")

	vs := []Visit{visit(1, "2026-09-01", "other", "done"), visit(2, "2026-09-30", "nt", "")}
	st = ResolveItem(11, 14, "nt", vs, d)
	assert.Equal(t, StateBooked, st.State)
	assert.EqualValues(t, 2, st.Visit.ID)
	assert.Nil(t, st.Suggested)

	vs = append(vs, visit(3, "2026-09-20", "nt", "result"))
	st = ResolveItem(11, 14, "nt", vs, d)
	assert.Equal(t, StateDone, st.State)
	assert.EqualValues(t, 3, st.Visit.ID)

	// A past linked visit without a stage still counts as booked.
	st = ResolveItem(6, 8, "first", []Visit{visit(4, "2026-08-20", "first", "")}, d)
	assert.Equal(t, StateBooked, st.State)
}

func TestResolveItem_WindowFollowsDating(t *testing.T) {
	// LMP says 12w0d on 2026-09-23; the ultrasound says 10w2d → the window moves ~12 days later.
	a := ResolveItem(11, 14, "nt", nil, dating(t, lmp("2026-07-01"), "2026-09-23"))
	b := ResolveItem(11, 14, "nt", nil, dating(t, ultrasound("2026-09-09", 8, 2), "2026-09-23"))
	assert.Equal(t, "2026-09-09", a.From.String())
	assert.Equal(t, "2026-09-21", b.From.String())
	assert.Equal(t, "2026-10-18", b.To.String())
}

func TestMonthOf_Boundaries(t *testing.T) {
	m := MonthOf(1405, 7, "fa") // Mehr 1405
	assert.Equal(t, "2026-09-23", m.First.String())
	assert.Equal(t, "2026-10-22", m.Last.String())
	m = MonthOf(1405, 12, "fa") // Esfand 1405 (29 days)
	assert.Equal(t, "2027-02-20", m.First.String())
	assert.Equal(t, "2027-03-20", m.Last.String())
	m = MonthOf(2026, 12, "en")
	assert.Equal(t, "2026-12-01", m.First.String())
	assert.Equal(t, "2026-12-31", m.Last.String())
	m = MonthOf(2028, 2, "en")
	assert.Equal(t, "2028-02-29", m.Last.String())
	assert.Equal(t, MonthOf(1405, 6, "fa"), MonthContaining(civildate.MustParse("2026-09-22"), "fa"))
	assert.Equal(t, MonthOf(1405, 7, "fa"), MonthContaining(civildate.MustParse("2026-09-23"), "fa"))
}

func TestBuild_MonthMarkersAndVisits(t *testing.T) {
	d := dating(t, lmp("2026-07-01"), "2026-09-23")
	vs := []Visit{visit(1, "2026-09-22", "nt", "booked"), visit(2, "2026-09-23", "", ""), visit(3, "2026-10-22", "", "")}
	out := build(d, MonthOf(1405, 7, "fa"), vs, nil, nil, v2.Lang{Locale: "fa", Default: "fa"})
	days, _ := out.Get("days")
	list := days.([]any)
	assert.Len(t, list, 30)
	visits, _ := out.Get("visits")
	assert.Len(t, visits.([]any), 2, "2026-09-22 is the last day of Shahrivar")
	month, _ := out.Get("month")
	assert.Equal(t, "1405-07", month)
	first := list[0].(interface{ Get(string) (any, bool) })
	ws, _ := first.Get("week_start")
	assert.Equal(t, 13, ws) // LMP 2026-07-01 is a Wednesday → week 13 starts 2026-09-23
	it, _ := first.Get("is_today")
	assert.Equal(t, true, it)
}
