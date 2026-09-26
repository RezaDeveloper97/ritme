package v2

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

func lmpProfile(lmp string) *store.PregnancyProfile {
	return &store.PregnancyProfile{
		PregnancyMode: true,
		AgeSource:     sql.NullString{String: "lmp", Valid: true},
		LmpDate:       civildate.NullDate{Date: civildate.MustParse(lmp), Valid: true},
	}
}

func TestResolve_LMP(t *testing.T) {
	d, ok := Resolve(lmpProfile("2026-07-01"), civildate.MustParse("2026-09-23"))
	assert.True(t, ok)
	assert.Equal(t, 84, d.TotalDays)
	assert.Equal(t, 12, d.Weeks)
	assert.Equal(t, 13, d.CurrentWeek())
	assert.Equal(t, "2027-04-07", d.Due.String())
	from, to := d.WeekRange(13)
	assert.Equal(t, "2026-09-23", from.String())
	assert.Equal(t, "2026-09-29", to.String())
	bf, bt := d.BirthRange()
	assert.Equal(t, "2027-03-21", bf.String())
	assert.Equal(t, "2027-04-24", bt.String())
	assert.Equal(t, 30, d.Percent())
	assert.Equal(t, "past", d.Relation(12))
	assert.Equal(t, "current", d.Relation(13))
}

func TestResolve_Overdue(t *testing.T) {
	d, _ := Resolve(lmpProfile("2026-01-01"), civildate.MustParse("2026-10-15"))
	assert.Equal(t, 41, d.Weeks)
	assert.Equal(t, 42, d.CurrentWeek())
	assert.Equal(t, 100, d.Percent())
	assert.Negative(t, d.DaysLeft())
}

func TestResolve_NoData(t *testing.T) {
	_, ok := Resolve(&store.PregnancyProfile{}, civildate.MustParse("2026-09-23"))
	assert.False(t, ok)
	_, ok = Resolve(nil, civildate.MustParse("2026-09-23"))
	assert.False(t, ok)
}

func TestLabels(t *testing.T) {
	d := civildate.MustParse("2026-10-07")
	assert.Equal(t, "۱۵ مهر ۱۴۰۵", FullDate(d, "fa"))
	assert.Equal(t, "October 7, 2026", FullDate(d, "en"))
	assert.Equal(t, "سه\u200cماههٔ اول · هفتهٔ ۸ از ۴۰", WeekLabel(8, "fa"))
	assert.Equal(t, "Second trimester · Week 14 of 40", WeekLabel(14, "en"))
	for _, loc := range []string{"fa", "en"} {
		assert.NotEqual(t, "pregnancy_v2.messages.not_active", T("messages.not_active", loc))
		assert.NotEmpty(t, attributes(loc))
	}
}
