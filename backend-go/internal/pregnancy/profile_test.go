package pregnancy

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

func attrString(t *testing.T, m *jsonx.OrderedMap, key string) any {
	t.Helper()
	v, _ := m.Get(key)
	return v
}

// D-169: a saved scan replaces the LMP dating only when the discrepancy is more than the band's limit.
func TestApplyScanRule(t *testing.T) {
	clinicalconfig.Reset()
	lmp := &store.PregnancyProfile{
		AgeSource: sql.NullString{String: "lmp", Valid: true},
		LmpDate:   civildate.NullDate{Date: civildate.MustParse("2026-07-01"), Valid: true},
	}
	scan := func(w, d int) *jsonx.OrderedMap {
		return jsonx.Obj("ultrasound_date", "2026-09-01", "ultrasound_weeks", int64(w), "ultrasound_days", int64(d))
	}

	assert.Nil(t, ApplyScanRule(lmp, jsonx.Obj("blood_type", "A"), false), "no scan, no decision")

	// GA by LMP on 2026-09-01 = 62 (8+6, limit 5): 9+5 = 68 → 6 days → replace.
	a := scan(9, 5)
	r := ApplyScanRule(lmp, a, false)
	require.NotNil(t, r)
	assert.True(t, r.Replaced)
	assert.Equal(t, calc.BasisUltrasound, r.Basis)
	assert.Equal(t, "ultrasound", attrString(t, a, "age_source"))
	assert.Equal(t, "high", attrString(t, a, "confidence_level"))
	assert.Equal(t, int64(1), attrString(t, a, "uncertainty_days"))

	// 9+4 = 67 → 5 days → the LMP stays.
	a = scan(9, 4)
	r = ApplyScanRule(lmp, a, false)
	assert.False(t, r.Replaced)
	assert.Equal(t, "lmp", attrString(t, a, "age_source"))
	assert.Equal(t, "medium", attrString(t, a, "confidence_level"))
	assert.Equal(t, 5, *r.DiscrepancyDays)
	assert.Equal(t, 62, *r.BandUntilDays)
	assert.Equal(t, 5, *r.LimitDays)

	// A user-chosen basis stays; the comparison is still reported.
	a = scan(9, 4)
	a.Set("age_source", "ultrasound")
	r = ApplyScanRule(lmp, a, true)
	assert.Equal(t, "ultrasound", r.Basis)
	assert.False(t, r.Replaced)
	assert.Equal(t, "ultrasound", attrString(t, a, "age_source"))
	_, touched := a.Get("confidence_level")
	assert.False(t, touched)

	// Without an LMP (none stored, or cleared in the same save) the scan dates the pregnancy.
	a = scan(9, 4)
	r = ApplyScanRule(&store.PregnancyProfile{}, a, false)
	assert.Equal(t, calc.Redating{Basis: calc.BasisUltrasound}, *r)
	assert.Equal(t, "ultrasound", attrString(t, a, "age_source"))
	a = scan(9, 4)
	a.Set("lmp_date", nil)
	assert.Equal(t, calc.BasisUltrasound, ApplyScanRule(lmp, a, false).Basis)

	// Only the weeks change: the stored scan date and days complete it.
	stored := *lmp
	stored.UltrasoundDate = civildate.NullDate{Date: civildate.MustParse("2026-09-01"), Valid: true}
	stored.UltrasoundWeeks = sql.NullInt32{Int32: 8, Valid: true}
	stored.UltrasoundDays = sql.NullInt32{Int32: 6, Valid: true}
	r = ApplyScanRule(&stored, jsonx.Obj("ultrasound_weeks", int64(10)), false)
	assert.True(t, r.Replaced, "10+6 = 76 vs 62")
	assert.Nil(t, ApplyScanRule(&store.PregnancyProfile{}, jsonx.Obj("ultrasound_weeks", int64(10)), false),
		"an incomplete scan decides nothing")
}
