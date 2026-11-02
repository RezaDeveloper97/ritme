package vitals

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/vitals/store"
)

// TestCopy_NoHardCodedThresholds, database half (T-M10-19b): the seeded message_contents vitals_alert rows (goose
// 00035, rewritten by 00116) never spell a threshold either, and the API fills their placeholders from the reading's
// version.
func TestCopy_DBRowsNoHardCodedThresholds(t *testing.T) {
	clinicalconfig.Reset()
	db := testdb.New(t)
	cp, err := LoadAlertCopy(context.Background(), store.New(db))
	require.NoError(t, err)
	for _, rule := range []string{RuleBPCrisis, RuleGlucoseLow} {
		for _, locale := range []string{"fa", "en"} {
			m, ok := cp[rule][locale]
			require.True(t, ok, "vitals_alert %s %s seeded", rule, locale)
			var texts []string
			for _, k := range []string{"title", "what_we_saw", "advice", "contact"} {
				collectStrings(m[k], &texts)
			}
			for _, s := range texts {
				assert.Empty(t, hardCoded(t, s, locale), "vitals_alert %s %s: %q", rule, locale, s)
			}
		}
	}

	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{
		{Key: clinicalconfig.KeyBPCrisis, Version: 2, Value: clinicalconfig.Value{"systolic": 170, "diastolic": 110},
			EffectiveFrom: civildate.MustParse("2026-10-01")},
	})
	bp := Reading{Type: TypeBP, Date: civildate.MustParse("2026-10-02"), Systolic: 175, Diastolic: 100}
	a := decode(t, fillAlert(cp.AlertJSON(bp, "fa", "fa"), bp, "fa"))
	assert.Contains(t, a["what_we_saw"], "بالاتر از ۱۷۰/۱۱۰")
	assert.Contains(t, a["advice"], "بالاتر از ۱۷۰/۱۱۰")
	low := Reading{Type: TypeGlucose, Date: civildate.MustParse("2026-10-02"), MgDl: 50, Unit: UnitMgDl,
		Context: ContextFasting}
	a = decode(t, fillAlert(cp.AlertJSON(low, "en", "fa"), low, "en"))
	assert.Contains(t, a["what_we_saw"], "below 54 mg/dL (3.0 mmol/L)")
	assert.NotContains(t, a["what_we_saw"], "{")
}
