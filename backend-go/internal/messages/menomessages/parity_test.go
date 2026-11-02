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
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
)

// legacyDetect is the retired D-43 engine (CB-MENO-12) verbatim, kept as the parity oracle of T-M10-09: its hard-coded
// thresholds (score +4, review within 14 days) are now conditions of the seeded meno.* rules.
func legacyDetect(s menopause.Signals) []Hit {
	const scoreWorseBy, hrtReviewDays = 4, 14
	if s.Mode != enums.LifeModeMenopause {
		return nil
	}
	var hits []Hit
	if s.Bleeding.Alert {
		hits = append(hits, Hit{Rule: RuleBleeding, Kind: KindAlert, Priority: PriorityHigh, Link: LinkBleeding,
			BleedingLastOn: s.Bleeding.LastOn, Item: s.Bleeding.Item})
	}
	overdue, overdueN := firstWithStatus(s.Checkups, engine.StatusOverdue)
	if overdue != nil {
		hits = append(hits, Hit{Rule: RuleCheckupOverdue, Kind: KindReminder, Priority: PriorityMedium,
			Link: checkupLink(overdue), Checkup: overdue, CheckupCount: overdueN})
	}
	if sc := s.Score; sc != nil {
		if d := sc.Delta(); d != nil && *d >= scoreWorseBy && fresh(sc, s.Date) {
			hits = append(hits, Hit{Rule: RuleScoreWorsened, Kind: KindAlert, Priority: PriorityMedium, Link: LinkScore, Score: sc})
		}
	}
	for i, it := range s.HRT {
		if !it.ReviewOn.Valid {
			continue
		}
		if days := s.Date.DiffDays(it.ReviewOn.Date); days >= 0 && days <= hrtReviewDays {
			hits = append(hits, Hit{Rule: RuleHRTReview, Kind: KindReminder, Priority: PriorityMedium,
				Link: LinkTreatment, Treatment: &s.HRT[i], DaysLeft: days})
		}
	}
	if overdue == nil {
		if due, n := firstWithStatus(s.Checkups, engine.StatusDue); due != nil {
			hits = append(hits, Hit{Rule: RuleCheckupDue, Kind: KindReminder, Priority: PriorityLow,
				Link: checkupLink(due), Checkup: due, CheckupCount: n})
		}
	}
	for i := range s.Tips {
		hits = append(hits, Hit{Rule: s.Tips[i].Code, Kind: KindTip, Priority: PriorityLow, Item: &s.Tips[i]})
	}
	return hits
}

func stageTip(stage string) catalog.Item {
	return catalog.Item{Code: "stage_" + stage, Title: json.RawMessage(`{"en":"T"}`), Meta: json.RawMessage(`{"placement":"home"}`)}
}

// parityFixtures are the rules_test fixtures (plus edges), each with the home tip HomeTips would pick for its stage.
func parityFixtures() map[string]menopause.Signals {
	last := d("2026-09-28")
	with := func(f func(*menopause.Signals)) menopause.Signals {
		s := base()
		f(&s)
		st := s.Stage.Stage
		if st == "" {
			st = menopause.StageUnsure
		}
		s.Tips = []catalog.Item{stageTip(st)}
		return s
	}
	out := map[string]menopause.Signals{
		"nothing":              with(func(*menopause.Signals) {}),
		"bleeding":             with(func(s *menopause.Signals) { s.Bleeding = menopause.Bleeding{Alert: true, LastOn: &last} }),
		"bleeding day no flag": with(func(s *menopause.Signals) { s.Bleeding = menopause.Bleeding{LastOn: &last} }),
		"overdue and due": with(func(s *menopause.Signals) {
			s.Checkups = []engine.Item{{TypeID: 3, Status: engine.StatusOverdue}, {TypeID: 5, Status: engine.StatusDue}}
		}),
		"due only":  with(func(s *menopause.Signals) { s.Checkups = []engine.Item{{TypeID: 5, Status: engine.StatusDue}} }),
		"soon only": with(func(s *menopause.Signals) { s.Checkups = []engine.Item{{TypeID: 6, Status: engine.StatusSoon}} }),
		"hrt window": with(func(s *menopause.Signals) {
			s.HRT = []store.TreatmentItem{hrt(1, "2026-10-01"), hrt(2, "2026-10-15"), hrt(3, "2026-10-16"),
				hrt(4, "2026-09-30"), hrt(5, "")}
		}),
		"no stage": with(func(s *menopause.Signals) { s.Stage = menopause.Stage{} }),
		"peri":     with(func(s *menopause.Signals) { s.Stage = menopause.Stage{Stage: menopause.StagePeri} }),
		"post everything": with(func(s *menopause.Signals) {
			s.Stage = menopause.Stage{Stage: menopause.StagePost}
			s.Bleeding = menopause.Bleeding{Alert: true, LastOn: &last}
			s.Checkups = []engine.Item{{TypeID: 1, Status: engine.StatusOverdue}}
			s.Score = score("2026-09-23", 20, ip(10))
			s.HRT = []store.TreatmentItem{hrt(9, "2026-10-05")}
		}),
	}
	for name, sc := range map[string]*menopause.ScoreEntry{
		"score +4": score("2026-09-23", 14, ip(10)), "score +3": score("2026-09-23", 13, ip(10)),
		"score better": score("2026-09-23", 6, ip(10)), "score no previous": score("2026-09-23", 30, nil),
		"score previous month": score("2026-08-23", 20, ip(10)), "score stale": score("2026-07-23", 20, ip(10)),
	} {
		out[name] = with(func(s *menopause.Signals) { s.Score = sc })
	}
	for _, mode := range []enums.LifeMode{enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModePregnancy, ""} {
		out["mode "+string(mode)] = with(func(s *menopause.Signals) {
			s.Mode = mode
			s.Bleeding = menopause.Bleeding{Alert: true}
		})
	}
	return out
}

func compact(hits []Hit) [][3]any {
	out := [][3]any{}
	for _, h := range hits {
		var extra any
		switch {
		case h.Treatment != nil:
			extra = h.Treatment.ID
		case h.Checkup != nil:
			extra = [2]any{h.Checkup.TypeID, h.CheckupCount}
		}
		out = append(out, [3]any{h.Rule + "|" + h.Kind + "|" + h.Priority + "|" + h.Link, extra, h.DaysLeft})
	}
	return out
}

// T-M10-09 parity table: the retired engine's messages equal the meno.* rule matches over every fixture — through the
// code-held rules and through the seeded rules compiled as the engine compiles them.
func TestParity_LegacyEngineEqualsRules(t *testing.T) {
	var seeded []rules.Rule
	for i, d := range Defs {
		seeded = append(seeded, d.Rule(uint64(i+1))) //nolint:gosec // small
	}
	comp := compiled.CompileRules(seeded)
	byKey := map[string]*rules.Rule{}
	for _, r := range comp.BySlot[rules.SlotAlertsInbox] {
		byKey[r.Src.RuleKey] = r.Src
	}
	for _, r := range comp.BySlot[SlotDailyAdvice] {
		byKey[r.Src.RuleKey] = r.Src
	}
	require.Len(t, byKey, len(Defs))
	compiledHolds := func(d *Def, f rules.Facts) bool { return rules.MatchRule(byKey[d.Key], f) }

	for name, s := range parityFixtures() {
		want := compact(legacyDetect(s))
		assert.Equal(t, want, compact(Detect(s)), "%s: code-held", name)
		assert.Equal(t, want, compact(DetectWith(s, compiledHolds)), "%s: seeded", name)
	}
}

// The alerts inbox selects exactly the alerts / reminders of the parity table (one winner per rule; an HRT rule wins
// when any item is near its review).
func TestParity_InboxSelection(t *testing.T) {
	var box []rules.Rule
	for _, d := range Defs {
		if d.Slot == rules.SlotAlertsInbox {
			box = append(box, d.Rule(CodeRuleID(d.Key)))
		}
	}
	comp := compiled.CompileRules(box)
	for name, s := range parityFixtures() {
		want := map[string]bool{}
		for _, h := range legacyDetect(s) {
			if h.Kind != KindTip {
				want[KeyPrefix+h.Rule] = true
			}
		}
		got := map[string]bool{}
		f := rules.Facts(facts.MenopauseValues(Input(s, nil)))
		for _, ch := range comp.Select(rules.SlotAlertsInbox, f, nil, 1, s.Date.TehranMidnight().Add(10*3600e9)) {
			got[ch.Rule.RuleKey] = true
		}
		assert.Equal(t, want, got, name)
	}
}

func TestInput_NothingOutsideMenopause(t *testing.T) {
	s := base()
	s.Mode = enums.LifeModeCycle
	v := facts.MenopauseValues(Input(s, nil))
	assert.NotContains(t, v, "meno.stage")
	assert.Equal(t, false, v["meno.postmenopausal_bleeding"])
	assert.InDelta(t, -1, v["meno.hrt_review_in_days"], 0)
}

func TestSeedCopy_EveryRuleInEveryLanguage(t *testing.T) {
	for _, code := range CopyLanguages() {
		for _, d := range Defs {
			cp, ok := embedded()[code][d.Key]
			require.True(t, ok, "%s %s", code, d.Key)
			assert.NotEmpty(t, cp.Title, "%s %s", code, d.Key)
			assert.NotEmpty(t, cp.Body, "%s %s", code, d.Key)
			if d.Level == rules.LevelWorthFollowUp || d.Level == rules.LevelFollowSoon {
				assert.NotEmpty(t, cp.Seen, "%s %s: follow_up needs «what we saw»", code, d.Key)
				assert.NotEmpty(t, cp.Confidence, "%s %s", code, d.Key)
			}
			for _, v := range rules.Placeholders(cp.Title, cp.Body, cp.CTA, cp.Seen, cp.Confidence) {
				def, ok := facts.Lookup(v)
				assert.True(t, ok && def.Variable, "%s %s: {%s} is a registered text variable", code, d.Key, v)
			}
		}
	}
	for _, d := range Defs {
		assert.True(t, rules.LevelAllowed(d.Level, d.Kind), d.Key)
		assert.True(t, rules.KindAllowedIn(d.Slot, d.Kind), d.Key)
		for _, g := range d.Groups {
			for _, it := range g.Items {
				_, ok := facts.Lookup(it.Fact)
				assert.True(t, ok, "%s: %s", d.Key, it.Fact)
			}
		}
	}
}
