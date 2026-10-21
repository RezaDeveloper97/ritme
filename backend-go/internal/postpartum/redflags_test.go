package postpartum

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

func fp(f float64) *float64 { return &f }
func bp(b bool) *bool       { return &b }

// T-M9-06: the red flags are urgent, the lochia alerts stay warnings.
func TestRecoveryAlerts_RedFlags(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.Reset()
	day := civildate.MustParse("2026-10-21")
	assert.Empty(t, Recovery{TemperatureC: fp(37.9)}.AlertsOn(day))
	assert.Equal(t, []string{guide.AlertFever}, Recovery{TemperatureC: fp(38.0)}.AlertsOn(day))
	assert.Equal(t, []string{guide.AlertPadPerHour}, Recovery{PadSoakedWithinHour: bp(true)}.AlertsOn(day))
	assert.Equal(t, []string{guide.AlertSeverePain}, Recovery{SeverePain: bp(true)}.AlertsOn(day))
	assert.Equal(t, []string{guide.AlertBreathless}, Recovery{Breathless: bp(true)}.AlertsOn(day))
	assert.Empty(t, Recovery{PadSoakedWithinHour: bp(false), SeverePain: bp(false), Breathless: bp(false)}.AlertsOn(day))
	assert.Equal(t, []string{guide.AlertHeavyBleeding}, Recovery{LochiaAmount: "heavy"}.AlertsOn(day), "heavy lochia unchanged")
	assert.Equal(t, []string{guide.AlertFever, guide.AlertHeavyBleeding, guide.AlertLargeClots},
		Recovery{TemperatureC: fp(39), LochiaAmount: "heavy", LargeClots: true}.AlertsOn(day))

	// the fever threshold is config
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyPostpartumRedFlags, Version: 2,
		Value: clinicalconfig.Value{"fever_c": 38.5}, EffectiveFrom: civildate.MustParse("2026-10-01")}})
	assert.Empty(t, Recovery{TemperatureC: fp(38.0)}.AlertsOn(day))
	assert.Equal(t, []string{guide.AlertFever}, Recovery{TemperatureC: fp(38.0)}.AlertsOn(civildate.MustParse("2026-09-30")))

	for _, locale := range []string{"fa", "en"} {
		alerts, err := RecoveryAlerts(context.Background(), guide.NewCopy(nil, locale, ""),
			Recovery{Breathless: bp(true), LochiaAmount: "heavy"}, day)
		require.NoError(t, err)
		require.Len(t, alerts, 2)
		urgent := toMap(t, alerts[0])
		assert.Equal(t, LevelUrgent, urgent["level"])
		assert.Equal(t, "115", urgent["action"].(map[string]any)["number"])
		assert.NotEmpty(t, urgent["title"])
		assert.Contains(t, urgent["body"], map[string]string{"fa": "۱۱۵", "en": "115"}[locale])
		assert.Equal(t, LevelWarning, toMap(t, alerts[1])["level"])
	}
	fa := guide.NewCopy(nil, "fa", "").Embedded(guide.AlertGroup, guide.AlertFever)
	assert.Contains(t, fa["body"], "همین حالا با ۱۱۵ تماس بگیر یا به اورژانس برو")
}

// An admin row that empties an urgent alert never silences it: the embedded copy fills in.
func TestUrgentCopyFallsBackWhenEmpty(t *testing.T) {
	rows := emptyRows{}
	alerts, err := RecoveryAlerts(context.Background(), guide.NewCopy(rows, "fa", "fa"), Recovery{SeverePain: bp(true)},
		civildate.MustParse("2026-10-21"))
	require.NoError(t, err)
	m := toMap(t, alerts[0])
	assert.NotEmpty(t, m["title"])
	assert.NotEmpty(t, m["body"])
}

type emptyRows struct{}

func (emptyRows) Payload(context.Context, string, string, string) (any, bool, error) {
	return map[string]any{"title": "", "body": "", "action": ""}, true, nil
}

func TestRecoveryLogRow_PartialMerge(t *testing.T) {
	stored := Recovery{TemperatureC: fp(37.2), SeverePain: bp(true)}
	in := RecoveryInput{Set: map[string]bool{"severe_pain": true, "breathless": true}, SeverePain: nil, Breathless: bp(true)}
	assert.True(t, in.HasLog())
	row := in.LogRow(stored)
	assert.Equal(t, "37.2", row.TemperatureC.String, "a key not sent stays")
	assert.False(t, row.SeverePain.Valid, "null clears")
	assert.True(t, row.Breathless.Valid && row.Breathless.Bool)
	assert.False(t, RecoveryInput{Set: map[string]bool{"lochia_amount": true}}.HasLog())
}

func TestSafetyJSON_LikelyWithoutUrgent(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyEPDSUrgentTotal, Version: 2,
		Value: clinicalconfig.Value{"score": 31}, EffectiveFrom: civildate.MustParse("2026-10-01")}})
	day := civildate.MustParse("2026-10-21")
	r := ScoreOn(KindFull, map[string]int{"q1": 3, "q2": 3, "q3": 3, "q4": 3, "q5": 1}, day)
	require.Equal(t, BandLikely, r.Band)
	require.False(t, r.Urgent)
	v, err := SafetyJSON(context.Background(), guide.NewCopy(nil, "en", ""), r)
	require.NoError(t, err)
	m := toMap(t, v)
	assert.Equal(t, LevelAdvice, m["level"])
	assert.Equal(t, "Talk to your doctor soon", m["title"])

	// Q10 = 1, total 4 → urgent with 115 and 123
	u, err := SafetyJSON(context.Background(), guide.NewCopy(nil, "en", ""),
		ScoreOn(KindFull, map[string]int{"q1": 1, "q2": 1, "q3": 1, "q10": 1}, day))
	require.NoError(t, err)
	um := toMap(t, u)
	assert.Equal(t, LevelUrgent, um["level"])
	var numbers []any
	for _, a := range um["actions"].([]any) {
		numbers = append(numbers, a.(map[string]any)["number"])
	}
	assert.Contains(t, numbers, "115")
	assert.Contains(t, numbers, "123")
}
