package voicelog

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/conditions"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/pelvic"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var allTargets = map[string]bool{TargetHotFlash: true, TargetPainDiary: true, TargetPill: true, TargetBladder: true}

// canvasVocabFor is the vocabulary of mode with every canvas target on.
func canvasVocabFor(t *testing.T, mode, locale string) *vocabulary {
	t.Helper()
	v := vocabFor(t, mode, locale)
	v.addCanvas(allTargets, locale)
	return v
}

// understood runs the fake parser over a fixture and validates it like Process does: "category.param.item" → label.
func understood(t *testing.T, v *vocabulary, fixture, mode, locale string) (map[string]string, []Suggestion) {
	t.Helper()
	cands, _, err := ai.NewFake().ParseLog(context.Background(), ai.LogParseRequest{
		Text: ai.FakeTranscripts[fixture][locale], Language: locale, Vocabulary: v.entries})
	require.NoError(t, err)
	got := v.validate(cands, mode, locale)
	out := map[string]string{}
	for _, s := range got {
		out[s.Target+":"+s.Category+"."+s.Param+"."+s.Item] = s.Label
	}
	return out, got
}

func TestCanvasVocabulary(t *testing.T) {
	v := canvasVocabFor(t, "menopause", "fa")
	byKey := map[string]ai.VocabEntry{}
	for _, e := range v.entries {
		byKey[e.Key] = e
	}
	for _, f := range canvasFields {
		require.Contains(t, byKey, f.key())
	}
	assert.Equal(t, ai.VocabEntry{Key: "pain_diary.score", Type: "integer", Label: "دفتر درد › شدت درد از ۰ تا ۱۰", Min: 0, Max: 10},
		byKey["pain_diary.score"])
	assert.Equal(t, 100, byKey["pain_diary.analgesic"].MaxLen)
	assert.Equal(t, []ai.VocabValue{{Code: "taken", Label: "خورده شد"}, {Code: "missed", Label: "جا ماند"}}, byKey["pill.status"].Values)
	assert.Len(t, byKey["bladder.leak"].Values, 4)

	off := vocabFor(t, "cycle", "fa")
	off.addCanvas(map[string]bool{TargetBladder: true}, "fa")
	assert.Contains(t, off.canvas, "bladder.leak")
	assert.NotContains(t, off.canvas, "hot_flash.count", "only the eligible targets are offered")
	assert.NotContains(t, off.canvas, "pill.status")
}

// Fixture tests per new item kind (CB-VOICE-01 acceptance), fa and en.
func TestValidate_CanvasFixtures(t *testing.T) {
	cases := []struct {
		fixture, mode string
		want          map[string]map[string]string // locale → key → label
	}{
		{"menopause", "menopause", map[string]map[string]string{
			"fa": {
				"hot_flash:hot_flash.count.":        "گرگرفتگی · ۳ بار",
				"hot_flash:hot_flash.night.":        "گرگرفتگی شبانه",
				"log:symptoms.general.night_sweats": "تعریق شبانه",
				"log:symptoms.general.brain_fog":    "مه مغزی",
				"log:menopause.triggers.caffeine":   "قهوه یا چای پررنگ",
			},
			"en": {
				"hot_flash:hot_flash.count.":        "Hot flashes · 3",
				"hot_flash:hot_flash.night.":        "Night-time hot flashes",
				"log:symptoms.general.night_sweats": "Night sweats",
				"log:symptoms.general.brain_fog":    "Brain fog",
				"log:menopause.triggers.caffeine":   "Coffee or strong tea",
			},
		}},
		{"pain_diary", "cycle", map[string]map[string]string{
			"fa": {
				"log:pain.location.abdomen":               "درد شکم · متوسط",
				"log:pain.relief.painkiller":              "مسکن",
				"pain_diary:pain_diary.score.":            "شدت درد · ۶ از ۱۰",
				"pain_diary:pain_diary.analgesic.":        "مسکن · ایبوپروفن",
				"pain_diary:pain_diary.analgesic_time.":   "ساعت مسکن · ۱۰:۰۰",
				"pain_diary:pain_diary.analgesic_effect.": "اثر مسکن · کمی کمک کرد",
				"pain_diary:pain_diary.missed_activity.":  "سر کار یا درس نرفتم",
			},
			"en": {
				"log:pain.location.abdomen":               "Abdomen pain · Moderate",
				"log:pain.relief.painkiller":              "Painkiller",
				"pain_diary:pain_diary.score.":            "Pain score · 6 of 10",
				"pain_diary:pain_diary.analgesic.":        "Painkiller · ibuprofen",
				"pain_diary:pain_diary.analgesic_time.":   "Painkiller at 10:00",
				"pain_diary:pain_diary.analgesic_effect.": "Painkiller effect · Helped a little",
				"pain_diary:pain_diary.missed_activity.":  "Missed work or school",
			},
		}},
		{"pill", "cycle", map[string]map[string]string{
			"fa": {"pill:pill.status.": "قرص امروز · خورده شد"},
			"en": {"pill:pill.status.": "Today's pill · Taken"},
		}},
		{"pelvic", "cycle", map[string]map[string]string{
			"fa": {"bladder:bladder.leak.": "نشت ادرار · با سرفه یا عطسه", "bladder:bladder.night_voids.": "بیدار شدن شبانه برای ادرار · ۲ بار"},
			"en": {"bladder:bladder.leak.": "Leak · With a cough or sneeze", "bladder:bladder.night_voids.": "Times up at night to pee · 2"},
		}},
	}
	for _, tc := range cases {
		for locale, want := range tc.want {
			t.Run(tc.fixture+"/"+locale, func(t *testing.T) {
				got, _ := understood(t, canvasVocabFor(t, tc.mode, locale), tc.fixture, tc.mode, locale)
				assert.Equal(t, want, got)
			})
		}
	}
}

func TestValidate_CanvasValues(t *testing.T) {
	v := canvasVocabFor(t, "menopause", "fa")
	got := v.validate([]ai.Candidate{
		{Key: "hot_flash.count", Value: 2.0, Confidence: 0.9},
		{Key: "hot_flash.severity", Value: "very_severe", Confidence: 0.9},
		{Key: "pain_diary.analgesic_time", Value: "9:05", Confidence: 0.9},
		{Key: "pain_diary.analgesic", Value: "  ژلوفن ", Confidence: 0.9},
		// dropped:
		{Key: "hot_flash.count", Value: 0.0, Confidence: 0.9},     // dup anyway
		{Key: "bladder.night_voids", Value: 2.5, Confidence: 0.9}, // not whole
		{Key: "bladder.leak", Value: "flood", Confidence: 0.9},    // unknown option
		{Key: "pill.status", Value: true, Confidence: 0.9},        // wrong type
		{Key: "pain_diary.score", Value: 11.0, Confidence: 0.9},   // out of range
		{Key: "pain_diary.missed_activity", Value: "yes", Confidence: 0.9},
		{Key: "hot_flash.duration", Value: 3.0, Confidence: 0.2}, // unsure
		{Key: "pain_diary.analgesic_effect", Value: "great", Confidence: 0.9},
	}, "menopause", "fa")
	require.Len(t, got, 4)
	assert.Equal(t, "hot_flash", got[0].Target)
	assert.Equal(t, "شدت گرگرفتگی · خیلی شدید", got[1].Label)
	assert.Equal(t, "09:05", got[2].Value)
	assert.Equal(t, "ژلوفن", got[3].Value)

	raw, err := json.Marshal(got[0].JSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"target":"hot_flash","category":"hot_flash","param":"count","item":null,"value":2,"confidence":0.9,"label":"گرگرفتگی · ۲ بار","options":[]}`, string(raw))

	// a canvas field the user may not log is dropped like an unknown slot
	plain := vocabFor(t, "menopause", "fa")
	assert.Empty(t, plain.validate([]ai.Candidate{{Key: "hot_flash.count", Value: 2.0, Confidence: 0.9}}, "menopause", "fa"))
}

func TestValidate_AmbiguityOptions(t *testing.T) {
	_, got := understood(t, canvasVocabFor(t, "cycle", "fa"), "default", "cycle", "fa")
	var bored Suggestion
	for _, s := range got {
		if s.Item == "bored" {
			bored = s
		}
	}
	require.Len(t, bored.Options, 2)
	raw, err := json.Marshal(bored.JSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"target":"log","category":"mood","param":"moods","item":"bored","value":true,"confidence":0.9,"label":"بی\u200cحوصله",
		"options":[{"category":"mood","param":"moods","item":"sad","value":true,"label":"غمگین"},
		{"category":"symptoms","param":"general","item":"fatigue","value":"yes","label":"خستگی"}]}`, string(raw))

	// invalid / unknown / duplicate alternatives are dropped
	v := vocabFor(t, "cycle", "en")
	s := v.validate([]ai.Candidate{{Key: "mood.moods.bored", Value: true, Confidence: 0.8,
		Alternatives: []string{"mood.moods.bored", "pain.location.leg", "sleep.quality", "nope", "mood.moods.sad", "mood.moods.sad"}}}, "cycle", "en")
	require.Len(t, s, 1)
	require.Len(t, s[0].Options, 1)
	assert.Equal(t, "sad", s[0].Options[0].Item)
}

// ---- commit ----

type stubWriters struct {
	enrolled   bool
	locations  []string
	method     *contraception.Method
	flashes    []menopause.FlashInput
	pain       []conditions.PainInput
	pills      []string
	diaries    []pelvic.DiaryInput
	painErr    error
	pillLogErr error
}

func (s *stubWriters) StartFlash(_ context.Context, _ uint64, in menopause.FlashInput, _ time.Time) (menopause.Flash, bool, error) {
	s.flashes = append(s.flashes, in)
	return menopause.Flash{}, true, nil
}

func (s *stubWriters) Enrolments(context.Context, uint64) ([]conditions.Enrolment, error) {
	return []conditions.Enrolment{{Program: conditions.ProgramEndo, Enrolled: s.enrolled}}, nil
}

func (s *stubWriters) Pain(_ context.Context, _ uint64, d civildate.Date) (conditions.PainDay, error) {
	return conditions.PainDay{Date: d, Locations: s.locations}, nil
}

func (s *stubWriters) SavePain(_ context.Context, _ uint64, in conditions.PainInput, _ string, _ time.Time) (conditions.PainDay, error) {
	s.pain = append(s.pain, in)
	return conditions.PainDay{}, s.painErr
}

func (s *stubWriters) Overview(context.Context, uint64, civildate.Date) (contraception.Overview, error) {
	return contraception.Overview{Method: s.method}, nil
}

func (s *stubWriters) LogPill(_ context.Context, _ uint64, d civildate.Date, status string, _ time.Time) error {
	s.pills = append(s.pills, d.String()+":"+status)
	return s.pillLogErr
}

func (s *stubWriters) SaveDiary(_ context.Context, _ uint64, in pelvic.DiaryInput, _ time.Time) (pelvic.Diary, error) {
	s.diaries = append(s.diaries, in)
	return pelvic.Diary{}, nil
}

func commitSvc(mode string, w *stubWriters) *Service {
	return NewService(nil, stubLogs{mode: mode}).WithWriters(Writers{Flashes: w, Pain: w, Pills: w, Bladder: w})
}

func pillMethod(start string) *contraception.Method {
	d, _ := civildate.Parse(start)
	return &contraception.Method{Method: contraception.MethodCombinedPill, PackType: "21_7", PackStartedOn: d}
}

var commitNow = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func day(s string) civildate.Date {
	d, _ := civildate.Parse(s)
	return d
}

func TestCommit_WritesThroughServices(t *testing.T) {
	w := &stubWriters{enrolled: true, locations: []string{"abdomen"}, method: pillMethod("2026-09-16")}
	saved, err := commitSvc("menopause", w).Commit(context.Background(), 1, day("2026-09-23"), []CommitItem{
		{Category: "hot_flash", Param: "count", Value: 3.0},
		{Category: "hot_flash", Param: "night", Value: true},
		{Category: "pain_diary", Param: "score", Value: 6.0},
		{Category: "pain_diary", Param: "analgesic", Value: "ایبوپروفن"},
		{Category: "pain_diary", Param: "analgesic_time", Value: "10:00"},
		{Category: "pill", Param: "status", Value: "taken"},
		{Category: "bladder", Param: "leak", Value: "cough"},
		{Category: "bladder", Param: "night_voids", Value: 2.0},
		{Category: "hot_flash", Param: "count", Value: 9.0}, // duplicate: the first wins
	}, "fa", commitNow)
	require.NoError(t, err)
	require.Len(t, saved, 8)
	assert.Equal(t, "گرگرفتگی · ۳ بار", saved[0].Label)

	// three finished night flashes, 03:00 and the two before it, 3 minutes each
	require.Len(t, w.flashes, 3)
	for i, f := range w.flashes {
		require.NotNil(t, f.Duration)
		assert.Equal(t, DefaultFlashSeconds, *f.Duration)
		assert.True(t, *f.Night)
		want := time.Date(2026, 9, 23, 3, 0, 0, 0, civildate.Tehran).Add(-time.Duration(2-i) * 4 * time.Minute)
		assert.True(t, want.Equal(*f.StartedAt), "%d: %s", i, f.StartedAt)
	}
	require.Len(t, w.pain, 1)
	assert.Equal(t, map[string]bool{"score": true, "analgesic": true, "analgesic_time": true}, w.pain[0].Set)
	assert.Equal(t, 6, *w.pain[0].Score)
	assert.Equal(t, "10:00", *w.pain[0].AnalgesicTime)
	assert.Equal(t, []string{"2026-09-23:taken"}, w.pills)
	require.Len(t, w.diaries, 1)
	assert.Equal(t, "cough", *w.diaries[0].Leak)
	assert.Equal(t, 2, *w.diaries[0].NightVoids)
	assert.Equal(t, map[string]bool{"leak": true, "night_voids": true}, w.diaries[0].Set)
}

func TestCommit_DayFlashBeforeNoon(t *testing.T) {
	w := &stubWriters{}
	_, err := commitSvc("menopause", w).Commit(context.Background(), 1, day("2026-09-23"), []CommitItem{
		{Category: "hot_flash", Param: "severity", Value: "mild"},
		{Category: "hot_flash", Param: "duration", Value: 5.0},
	}, "en", commitNow)
	require.NoError(t, err)
	require.Len(t, w.flashes, 1, "no count → one flash")
	assert.Equal(t, 300, *w.flashes[0].Duration)
	assert.Nil(t, w.flashes[0].Night, "night left to the diary's own rule")
	assert.Equal(t, "mild", w.flashes[0].Severity)
	assert.True(t, commitNow.Add(-5*time.Minute).Equal(*w.flashes[0].StartedAt), "noon is still ahead → it ended now")
}

func TestCommit_Refusals(t *testing.T) {
	cases := []struct {
		name  string
		mode  string
		w     *stubWriters
		items []CommitItem
		field string
		msg   string
	}{
		{"hot flash outside menopause", "cycle", &stubWriters{}, []CommitItem{{Category: "hot_flash", Param: "count", Value: 1.0}},
			"items.0.category", "Hot flashes are logged in menopause mode only."},
		{"pain diary without the program", "cycle", &stubWriters{}, []CommitItem{{Category: "pain_diary", Param: "score", Value: 1.0}},
			"items.0.category", "Join the endometriosis program to use the pain diary."},
		{"pill without a pill method", "cycle", &stubWriters{method: &contraception.Method{Method: contraception.MethodCopperIUD}},
			[]CommitItem{{Category: "pill", Param: "status", Value: "taken"}}, "items.0.category", "Choose the pill as your birth control method first."},
		{"log slots stay on PUT /logs/days", "cycle", &stubWriters{}, []CommitItem{{Category: "pain", Param: "location", Value: "mild"}},
			"items.0.category", "This item can't be logged by voice."},
		{"unknown param", "cycle", &stubWriters{}, []CommitItem{{Category: "bladder", Param: "color", Value: "x"}},
			"items.0.param", "This item can't be logged by voice."},
		{"bad value", "cycle", &stubWriters{}, []CommitItem{{Category: "bladder", Param: "leak", Value: "cough"}, {Category: "bladder", Param: "night_voids", Value: 21.0}},
			"items.1.value", "This item's value isn't valid."},
		{"score without a location", "cycle", &stubWriters{enrolled: true}, []CommitItem{{Category: "pain_diary", Param: "score", Value: 4.0}},
			"items.0.value", conditions.T("validation.locations_required", "en")},
		{"break day", "cycle", &stubWriters{method: pillMethod("2026-09-01")}, []CommitItem{{Category: "pill", Param: "status", Value: "taken"}},
			"items.0.value", contraception.T("validation.break_day", "en")},
		{"before the pack", "cycle", &stubWriters{method: pillMethod("2026-09-30")}, []CommitItem{{Category: "pill", Param: "status", Value: "missed"}},
			"items.0.value", contraception.T("validation.date_before_pack", "en")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := commitSvc(tc.mode, tc.w).Commit(context.Background(), 1, day("2026-09-23"), tc.items, "en", commitNow)
			var ie *ItemError
			require.ErrorAs(t, err, &ie)
			assert.Equal(t, tc.field, "items."+strconv.Itoa(ie.Index)+"."+ie.Field)
			assert.Equal(t, tc.msg, ie.Message)
			assert.Empty(t, tc.w.flashes, "nothing half-saved")
			assert.Empty(t, tc.w.pain)
			assert.Empty(t, tc.w.pills)
			assert.Empty(t, tc.w.diaries)
		})
	}
	// a service refusal after the pre-checks still maps to the item
	w := &stubWriters{enrolled: true, locations: []string{"head"}, painErr: &conditions.FieldError{Field: "score", Key: "score_required"}}
	_, err := commitSvc("cycle", w).Commit(context.Background(), 1, day("2026-09-23"),
		[]CommitItem{{Category: "pain_diary", Param: "missed_activity", Value: true}, {Category: "pain_diary", Param: "score", Value: 3.0}}, "en", commitNow)
	var ie *ItemError
	require.ErrorAs(t, err, &ie)
	assert.Equal(t, 1, ie.Index)
}

func TestNormalizeCanvas(t *testing.T) {
	f, _ := canvasFieldOf("pain_diary", "analgesic_time")
	for raw, want := range map[string]any{"7:30": "07:30", "23:59": "23:59", "24:00": nil, "7.30": nil, "": nil} {
		got, ok := normalizeCanvas(f, raw)
		if want == nil {
			assert.False(t, ok, raw)
			continue
		}
		assert.Equal(t, want, got, raw)
	}
	n, _ := canvasFieldOf("bladder", "night_voids")
	v, ok := normalizeCanvas(n, int64(3))
	assert.True(t, ok)
	assert.InDelta(t, 3.0, v, 0)
	_, ok = normalizeCanvas(n, "3")
	assert.True(t, ok)
	tx, _ := canvasFieldOf("pain_diary", "analgesic")
	_, ok = normalizeCanvas(tx, string(make([]rune, 101)))
	assert.False(t, ok)
	assert.Nil(t, CommitValue([]any{1}))
	assert.InDelta(t, 2.0, CommitValue(int64(2)), 0)
}
