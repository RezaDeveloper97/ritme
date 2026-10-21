package postpartum

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func visitsOf(vs []Visit) map[string]Visit {
	out := map[string]Visit{}
	for _, v := range vs {
		out[v.Kind] = v
	}
	return out
}

// T-M9-06: delivery 2026-10-01 → contact by 10-22, reminder 11-12, comprehensive by 12-24 (version 1: 21 / 42 / 84).
func TestVisitsOn(t *testing.T) {
	clinicalconfig.Reset()
	d := civildate.MustParse
	birth := d("2026-10-01")
	vs := VisitsOn(birth, d("2026-10-10"), nil)
	assert.Equal(t, VisitKinds, []string{vs[0].Kind, vs[1].Kind, vs[2].Kind})
	m := visitsOf(vs)
	assert.Equal(t, "2026-10-22", m[VisitContact].DueOn.String())
	assert.Equal(t, "2026-11-12", m[VisitReminder].DueOn.String())
	assert.Equal(t, "2026-12-24", m[VisitComprehensive].DueOn.String())
	for _, v := range vs {
		assert.Equal(t, VisitUpcoming, v.Status, v.Kind)
		assert.False(t, v.Done)
	}
	assert.Empty(t, NoticeKinds(vs))

	// day 42 without a visit: contact overdue, the reminder is due → one notice kind
	m = visitsOf(VisitsOn(birth, d("2026-11-12"), nil))
	assert.Equal(t, VisitOverdue, m[VisitContact].Status)
	assert.Equal(t, VisitDue, m[VisitReminder].Status)
	assert.Equal(t, VisitDue, m[VisitComprehensive].Status)
	assert.Equal(t, []string{VisitReminder}, NoticeKinds(VisitsOn(birth, d("2026-11-12"), nil)))
	assert.Empty(t, NoticeKinds(VisitsOn(birth, d("2026-11-11"), nil)), "not before reminder_day")
	assert.Equal(t, []string{VisitReminder, VisitComprehensive}, NoticeKinds(VisitsOn(birth, d("2026-12-25"), nil)))

	// a logged visit in the window marks it done
	vs = VisitsOn(birth, d("2026-11-20"), []civildate.Date{d("2026-09-30"), d("2026-10-15"), d("2026-11-15")})
	m = visitsOf(vs)
	assert.True(t, m[VisitContact].Done)
	assert.Equal(t, "2026-10-15", m[VisitContact].DoneOn.String())
	assert.True(t, m[VisitReminder].Done)
	assert.Equal(t, VisitDone, m[VisitComprehensive].Status)
	assert.Equal(t, "2026-11-15", m[VisitComprehensive].DoneOn.String())
	assert.Empty(t, NoticeKinds(vs))

	// a contact visit alone does not count for the 6-week / comprehensive visit
	m = visitsOf(VisitsOn(birth, d("2026-11-20"), []civildate.Date{d("2026-10-22")}))
	assert.True(t, m[VisitContact].Done)
	assert.False(t, m[VisitReminder].Done)

	// the days are config
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPostpartumVisits, Version: 2,
		Value:         clinicalconfig.Value{"contact_by_day": 14, "reminder_day": 35, "comprehensive_by_day": 84},
		EffectiveFrom: d("2026-10-01")}})
	m = visitsOf(VisitsOn(birth, d("2026-10-10"), nil))
	assert.Equal(t, "2026-10-15", m[VisitContact].DueOn.String())
	assert.Equal(t, "2026-11-05", m[VisitReminder].DueOn.String())
}
