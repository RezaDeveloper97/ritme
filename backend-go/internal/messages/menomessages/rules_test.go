package menomessages

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func d(s string) civildate.Date {
	v, err := civildate.Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// today is 2026-10-01 = 9 Mehr 1405.
func base() menopause.Signals {
	return menopause.Signals{Date: d("2026-10-01"), Mode: enums.LifeModeMenopause, Stage: menopause.Stage{Stage: menopause.StageMeno}}
}

func rulesOf(hits []Hit) []string {
	out := []string{}
	for _, h := range hits {
		out = append(out, h.Rule)
	}
	return out
}

func score(month string, total int, prev *int) *menopause.ScoreEntry {
	e := &menopause.ScoreEntry{Score: menopause.Score{Month: d(month), Total: total}}
	if prev != nil {
		e.Previous = &menopause.Score{Month: menopause.PrevMonth(d(month)), Total: *prev}
	}
	return e
}

func ip(n int) *int { return &n }

func hrt(id uint64, review string) store.TreatmentItem {
	it := store.TreatmentItem{ID: id, Kind: menopause.KindHRT, Name: "Estradiol gel"}
	if review != "" {
		it.ReviewOn = civildate.NullDate{Date: d(review), Valid: true}
	}
	return it
}

func tip(code string) catalog.Item {
	return catalog.Item{Code: code, Title: json.RawMessage(`{"en":"T"}`), Meta: json.RawMessage(`{"placement":"home"}`)}
}

func TestNonMenopauseUsersGetNothing(t *testing.T) {
	for _, mode := range []enums.LifeMode{enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModeTeen,
		enums.LifeModePregnancy, enums.LifeModePostpartum, ""} {
		s := base()
		s.Mode = mode
		s.Bleeding = menopause.Bleeding{Alert: true}
		s.Checkups = []engine.Item{{TypeID: 1, Status: engine.StatusOverdue}}
		s.Score = score("2026-09-23", 20, ip(10))
		s.HRT = []store.TreatmentItem{hrt(1, "2026-10-02")}
		s.Tips = []catalog.Item{tip("stage_meno")}
		assert.Empty(t, Detect(s), mode)
	}
}

func TestBleeding_HighPriorityFirst(t *testing.T) {
	s := base()
	last := d("2026-09-28")
	s.Bleeding = menopause.Bleeding{Alert: true, LastOn: &last}
	s.Checkups = []engine.Item{{TypeID: 7, Status: engine.StatusOverdue}}
	hits := Detect(s)
	require.Len(t, hits, 2)
	assert.Equal(t, RuleBleeding, hits[0].Rule)
	assert.Equal(t, PriorityHigh, hits[0].Priority)
	assert.Equal(t, KindAlert, hits[0].Kind)
	assert.Equal(t, LinkBleeding, hits[0].Link)
	assert.Equal(t, &last, hits[0].BleedingLastOn)

	// The flag is the menopause API's (stage meno/post + 30 days): a bleeding day without the flag raises nothing.
	s = base()
	s.Bleeding = menopause.Bleeding{LastOn: &last}
	assert.Empty(t, Detect(s))
}

func TestCheckups_OverdueBeatsDue(t *testing.T) {
	s := base()
	s.Checkups = []engine.Item{
		{TypeID: 3, Status: engine.StatusOverdue},
		{TypeID: 4, Status: engine.StatusOverdue},
		{TypeID: 5, Status: engine.StatusDue},
		{TypeID: 6, Status: engine.StatusSoon},
	}
	hits := Detect(s)
	require.Equal(t, []string{RuleCheckupOverdue}, rulesOf(hits))
	assert.Equal(t, uint64(3), hits[0].Checkup.TypeID)
	assert.Equal(t, 2, hits[0].CheckupCount)
	assert.Equal(t, "/checkups/3", hits[0].Link)
	assert.Equal(t, PriorityMedium, hits[0].Priority)

	s.Checkups = s.Checkups[2:]
	hits = Detect(s)
	require.Equal(t, []string{RuleCheckupDue}, rulesOf(hits))
	assert.Equal(t, uint64(5), hits[0].Checkup.TypeID)
	assert.Equal(t, 1, hits[0].CheckupCount)
	assert.Equal(t, PriorityLow, hits[0].Priority)

	// Soon is not messaged.
	s.Checkups = s.Checkups[1:]
	assert.Empty(t, Detect(s))
}

func TestScoreWorsened_Threshold(t *testing.T) {
	cases := []struct {
		name string
		sc   *menopause.ScoreEntry
		want bool
	}{
		{"exactly +4", score("2026-09-23", 14, ip(10)), true},
		{"+3", score("2026-09-23", 13, ip(10)), false},
		{"better", score("2026-09-23", 6, ip(10)), false},
		{"no previous", score("2026-09-23", 30, nil), false},
		{"previous Jalali month counts", score("2026-08-23", 20, ip(10)), true},
		{"two months ago is stale", score("2026-07-23", 20, ip(10)), false},
		{"none", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := base()
			s.Score = c.sc
			hits := Detect(s)
			if !c.want {
				assert.Empty(t, hits)
				return
			}
			require.Equal(t, []string{RuleScoreWorsened}, rulesOf(hits))
			assert.Equal(t, LinkScore, hits[0].Link)
		})
	}
}

func TestHRTReview_Window(t *testing.T) {
	s := base()
	s.HRT = []store.TreatmentItem{
		hrt(1, "2026-10-01"), // today
		hrt(2, "2026-10-15"), // 14 days
		hrt(3, "2026-10-16"), // 15 days: too far
		hrt(4, "2026-09-30"), // passed
		hrt(5, ""),           // no review date
	}
	hits := Detect(s)
	require.Equal(t, []string{RuleHRTReview, RuleHRTReview}, rulesOf(hits))
	assert.Equal(t, uint64(1), hits[0].Treatment.ID)
	assert.Equal(t, 0, hits[0].DaysLeft)
	assert.Equal(t, uint64(2), hits[1].Treatment.ID)
	assert.Equal(t, 14, hits[1].DaysLeft)
	assert.Equal(t, LinkTreatment, hits[1].Link)
}

func TestTips_LastAndAllRulesInOrder(t *testing.T) {
	s := base()
	s.Bleeding = menopause.Bleeding{Alert: true}
	s.Checkups = []engine.Item{{TypeID: 1, Status: engine.StatusOverdue}}
	s.Score = score("2026-09-23", 20, ip(10))
	s.HRT = []store.TreatmentItem{hrt(9, "2026-10-05")}
	s.Tips = []catalog.Item{tip("stage_meno")}
	hits := Detect(s)
	assert.Equal(t, []string{RuleBleeding, RuleCheckupOverdue, RuleScoreWorsened, RuleHRTReview, "stage_meno"}, rulesOf(hits))
	last := hits[len(hits)-1]
	assert.Equal(t, KindTip, last.Kind)
	assert.Equal(t, PriorityLow, last.Priority)
	assert.Empty(t, last.Link)

	s.Checkups[0].Status = engine.StatusDue
	assert.Equal(t, []string{RuleBleeding, RuleScoreWorsened, RuleHRTReview, RuleCheckupDue, "stage_meno"}, rulesOf(Detect(s)))
}

func TestEmbeddedCopyHasEveryRuleInEveryLanguage(t *testing.T) {
	for _, loc := range []string{"fa", "en"} { // the shipped embedded copies, not a language list
		for _, r := range Rules {
			for _, k := range TextKeys {
				assert.NotEmpty(t, fallbackText(r, k, loc), "%s %s.%s", loc, r, k)
			}
		}
	}
	assert.Contains(t, fallbackText(RuleScoreWorsened, "body", "en"), "{delta}")
	assert.Contains(t, fallbackText(RuleHRTReview, "body", "fa"), "{date}")
}
