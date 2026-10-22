package pregnancy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// D-169: a scan saved by the rule keeps the LMP dating when the discrepancy is within the band's limit and replaces it
// when it is more; the explicit path always dates by the scan.
func TestApplyUltrasoundDatingByRule(t *testing.T) {
	_, q, uid := setup(t)
	clinicalconfig.Reset()
	ctx := context.Background()
	require.NoError(t, saveProfile(ctx, q, uid, nil, jsonx.Obj("pregnancy_mode", true, "cycle_mode", false,
		"age_source", "lmp", "confidence_level", "medium", "uncertainty_days", int64(3), "lmp_date", "2026-07-01",
		"estimated_due_date", "2027-04-07"), t0))
	scan := civildate.MustParse("2026-09-01") // GA by LMP 62 (8+6) → limit 5 days

	p, r, err := ApplyUltrasoundDatingByRule(ctx, q, uid, UltrasoundDating{ScanDate: scan, Weeks: 9, Days: 4}, t0)
	require.NoError(t, err)
	assert.False(t, r.Replaced)
	assert.Equal(t, "lmp", p.AgeSource.String)
	assert.Equal(t, "2026-09-01", p.UltrasoundDate.Date.String(), "the scan is stored anyway")
	assert.Equal(t, "2027-04-07", p.EstimatedDueDate.Date.String())

	p, r, err = ApplyUltrasoundDatingByRule(ctx, q, uid, UltrasoundDating{ScanDate: scan, Weeks: 9, Days: 5}, t0)
	require.NoError(t, err)
	assert.True(t, r.Replaced)
	assert.Equal(t, 6, *r.DiscrepancyDays)
	assert.Equal(t, "ultrasound", p.AgeSource.String)
	assert.Equal(t, "high", p.ConfidenceLevel.String)
	assert.Equal(t, "2027-04-01", p.EstimatedDueDate.Date.String(), "2026-09-01 − 68 + 280")

	// The explicit path (a basis the user chose) dates by the scan whatever the discrepancy.
	p, err = ApplyUltrasoundDating(ctx, q, uid, UltrasoundDating{ScanDate: scan, Weeks: 8, Days: 6}, t0)
	require.NoError(t, err)
	assert.Equal(t, "ultrasound", p.AgeSource.String)
	assert.Equal(t, "2027-04-07", p.EstimatedDueDate.Date.String())
}
