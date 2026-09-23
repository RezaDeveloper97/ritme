package legacy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Edge cases the goldens don't reach, with expected values computed by the PHP classes themselves:
// testdata/php_cases.json is the output of testdata/php_cases.php (run it with `php` to regenerate).

type phpCases struct {
	SymptomScore []struct {
		Log            map[string]any `json:"log"`
		Pms            bool           `json:"pms"`
		LutealSpotting bool           `json:"luteal_spotting"`
		Score          float64        `json:"score"`
		Rounded        float64        `json:"rounded"`
	} `json:"symptom_score"`
	CycleScore []struct {
		Variability string         `json:"variability"`
		Log         map[string]any `json:"log"`
		Score       float64        `json:"score"`
		Rounded     float64        `json:"rounded"`
	} `json:"cycle_score"`
	AgeFactor []struct {
		Birthday *string `json:"birthday"`
		Today    string  `json:"today"`
		Factor   float64 `json:"factor"`
	} `json:"age_factor"`
	// [day, age, cycle, symptom, final, round(final*100, 2), "{round(final*100, 1)}"]
	FinalProbability [][7]any `json:"final_probability"`
	TextFlags        []struct {
		Phase          string          `json:"phase"`
		Subphase       string          `json:"subphase"`
		Fertile        bool            `json:"fertile"`
		Pms            bool            `json:"pms"`
		PeriodTomorrow bool            `json:"period_tomorrow"`
		Probability    float64         `json:"probability"`
		Variability    string          `json:"variability"`
		Flags          json.RawMessage `json:"flags"`
	} `json:"text_flags"`
	FallbackTips []struct {
		Phase    string          `json:"phase"`
		Subphase string          `json:"subphase"`
		Log      map[string]any  `json:"log"`
		Tips     json.RawMessage `json:"tips"`
	} `json:"fallback_tips"`
	CycleDay []struct {
		LMP      string `json:"lmp"`
		Length   int    `json:"length"`
		Date     string `json:"date"`
		CycleDay int    `json:"cycle_day"`
	} `json:"cycle_day"`
	LocalizeFallback map[string]json.RawMessage `json:"localize_fallback"`
}

func loadPHPCases(t *testing.T) phpCases {
	t.Helper()
	raw, err := os.ReadFile("testdata/php_cases.json")
	require.NoError(t, err)
	var c phpCases
	require.NoError(t, json.Unmarshal(raw, &c))
	return c
}

// logFromAttrs builds a DailyLog from raw column values the way the model casts read them.
// A nil map is "no log".
func logFromAttrs(t *testing.T, attrs map[string]any) *DailyLog {
	t.Helper()
	if attrs == nil {
		return nil
	}
	str := func(k string) *string {
		if v, ok := attrs[k]; ok {
			s := fmt.Sprint(v)
			return &s
		}
		return nil
	}
	boolean := func(k string) *bool {
		if v, ok := attrs[k]; ok {
			b := v.(float64) != 0
			return &b
		}
		return nil
	}
	list := func(k string) []string {
		v, ok := attrs[k]
		if !ok {
			return nil
		}
		var out []string
		if json.Unmarshal([]byte(v.(string)), &out) != nil {
			return nil // decoded to a non-array: is_array() false
		}
		return out
	}
	return &DailyLog{
		Spotting:             boolean("spotting"),
		VaginalDryness:       boolean("vaginal_dryness"),
		Fatigue:              boolean("fatigue"),
		DischargeTexture:     str("discharge_texture"),
		OvarianPainIntensity: str("ovarian_pain_intensity"),
		BloatingIntensity:    str("bloating_intensity"),
		HeadacheIntensity:    str("headache_intensity"),
		PelvicPainIntensity:  str("pelvic_pain_intensity"),
		StomachAcheIntensity: str("stomach_ache_intensity"),
		SleepQuality:         str("sleep_quality"),
		SexualDesire:         str("sexual_desire"),
		SexualActivities:     list("sexual_activities"),
		Moods:                list("moods"),
	}
}

func TestSymptomScore_MatchesPHP(t *testing.T) {
	c := loadPHPCases(t)
	require.NotEmpty(t, c.SymptomScore)
	for _, tc := range c.SymptomScore {
		got := SymptomScore(logFromAttrs(t, tc.Log), tc.Pms, tc.LutealSpotting)
		require.Equal(t, tc.Score, got, "%v pms=%v spotting=%v", tc.Log, tc.Pms, tc.LutealSpotting)
		require.Equal(t, tc.Rounded, phpround.Round(got, 4))
	}
}

func TestCycleScore_MatchesPHP(t *testing.T) {
	c := loadPHPCases(t)
	require.NotEmpty(t, c.CycleScore)
	for _, tc := range c.CycleScore {
		got := CycleScore(enums.CycleVariability(tc.Variability), logFromAttrs(t, tc.Log))
		require.Equal(t, tc.Score, got, "%s %v", tc.Variability, tc.Log)
		require.Equal(t, tc.Rounded, phpround.Round(got, 4))
	}
}

func TestAgeFactor_MatchesPHP(t *testing.T) {
	c := loadPHPCases(t)
	require.NotEmpty(t, c.AgeFactor)
	for _, tc := range c.AgeFactor {
		p := &model.Profile{LastPeriodStart: civildate.New(2026, 9, 1)}
		if tc.Birthday != nil {
			p.Birthday = civildate.MustParse(*tc.Birthday)
		}
		e := New(Input{Profile: p, Today: civildate.MustParse(tc.Today)})
		assert.Equal(t, tc.Factor, e.ageFactor(), "birthday %v on %s", tc.Birthday, tc.Today)
	}
}

func TestFinalProbabilityAndRounding_MatchesPHP(t *testing.T) {
	c := loadPHPCases(t)
	require.NotEmpty(t, c.FinalProbability)
	for _, row := range c.FinalProbability {
		day := int(row[0].(float64))
		age, cycle, symptom := row[1].(float64), row[2].(float64), row[3].(float64)
		final := FinalProbability(BaseProbability(day), age, cycle, symptom)
		msg := fmt.Sprintf("day %d age %v cycle %v symptom %v", day, age, cycle, symptom)
		require.Equal(t, row[4].(float64), final, msg)
		require.Equal(t, row[5].(float64), phpround.Round(final*100, 2), msg)
		require.Equal(t, row[6].(string), phpround.String(phpround.Round(final*100, 1)), msg)
	}
}

func TestTextFlags_MatchesPHP(t *testing.T) {
	c := loadPHPCases(t)
	require.NotEmpty(t, c.TextFlags)
	for _, tc := range c.TextFlags {
		flags := TextFlags(enums.CyclePhase(tc.Phase), enums.CycleSubphase(tc.Subphase), tc.Fertile, tc.Pms,
			tc.PeriodTomorrow, tc.Probability, enums.CycleVariability(tc.Variability))
		got, err := json.Marshal(Calculation{TextFlags: flags}.textFlagsJSON())
		require.NoError(t, err)
		assert.JSONEq(t, string(tc.Flags), string(got), "%s/%s p=%v", tc.Phase, tc.Subphase, tc.Probability)
	}
}

type noTips struct{}

func (noTips) HasContent(context.Context) (bool, error) { return false, nil }
func (noTips) ForDay(context.Context, enums.CyclePhase, enums.CycleSubphase, enums.TriggerLog) ([]recommendation.Tip, error) {
	panic("ForDay must not be called when the table is empty")
}

func TestFallbackTips_MatchesPHP(t *testing.T) {
	c := loadPHPCases(t)
	require.NotEmpty(t, c.FallbackTips)
	e := New(Input{Tips: noTips{}})
	for _, tc := range c.FallbackTips {
		tips, err := e.dailyTips(context.Background(), enums.CyclePhase(tc.Phase), enums.CycleSubphase(tc.Subphase), logFromAttrs(t, tc.Log))
		require.NoError(t, err)
		got, err := json.Marshal(Calculation{DailyTips: tips}.dailyTipsJSON())
		require.NoError(t, err)
		assert.JSONEq(t, string(tc.Tips), string(got), "%s/%s %v", tc.Phase, tc.Subphase, tc.Log)
	}

	fatigue := true
	tips, err := e.dailyTips(context.Background(), enums.CyclePhaseMenstruation, enums.CycleSubphaseMenstruation, &DailyLog{Fatigue: &fatigue})
	require.NoError(t, err)
	require.Len(t, c.LocalizeFallback, 3)
	for locale, want := range c.LocalizeFallback {
		got, err := json.Marshal(recommendation.Localize(tips, locale))
		require.NoError(t, err)
		assert.JSONEq(t, string(want), string(got), locale)
	}
}

func TestCycleDayFromLMP_MatchesPHP(t *testing.T) {
	c := loadPHPCases(t)
	require.NotEmpty(t, c.CycleDay)
	for _, tc := range c.CycleDay {
		e := New(Input{Profile: &model.Profile{LastPeriodStart: civildate.MustParse(tc.LMP)}})
		date := civildate.MustParse(tc.Date)
		got := cycleDayFor(e.relevantCycleStart(date, tc.Length), date, tc.Length)
		require.Equal(t, tc.CycleDay, got, "lmp %s length %d date %s", tc.LMP, tc.Length, tc.Date)
	}
}
