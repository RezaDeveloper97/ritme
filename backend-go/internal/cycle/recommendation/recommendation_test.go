package recommendation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
)

func strp(s string) *string { return &s }

// TestToTipAndLocalize_MatchesPHP replays Recommendation::toTip + DailyTipLocalizer::localize outputs
// computed by PHP (generated with the legacy engine cases: ../legacy/testdata/php_cases.php).
func TestToTipAndLocalize_MatchesPHP(t *testing.T) {
	raw, err := os.ReadFile("../legacy/testdata/php_cases.json")
	require.NoError(t, err)
	var cases struct {
		ToTip []struct {
			Row struct {
				Type  string  `json:"type"`
				Title *string `json:"title"`
				Text  string  `json:"text"`
			} `json:"row"`
			Tip       json.RawMessage            `json:"tip"`
			Localized map[string]json.RawMessage `json:"localized"`
		} `json:"to_tip"`
	}
	require.NoError(t, json.Unmarshal(raw, &cases))
	require.NotEmpty(t, cases.ToTip)

	for _, tc := range cases.ToTip {
		row := Row{Type: tc.Row.Type, Text: json.RawMessage(tc.Row.Text)}
		if tc.Row.Title != nil {
			row.Title = json.RawMessage(*tc.Row.Title)
		}
		tip := row.ToTip()
		got, err := json.Marshal(tip)
		require.NoError(t, err)
		assert.JSONEq(t, string(tc.Tip), string(got), "%+v", tc.Row)

		require.NotEmpty(t, tc.Localized)
		for locale, want := range tc.Localized {
			got, err := json.Marshal(Localize([]Tip{tip}, locale))
			require.NoError(t, err)
			assert.JSONEq(t, string(want), string(got), "%+v %s", tc.Row, locale)
		}
	}
}

func TestAppliesTo(t *testing.T) {
	menstruation, follicular := "menstruation", "follicular"
	early, mid := "early_follicular", "mid_follicular"
	headache := "headache"

	cases := []struct {
		name     string
		row      Row
		phase    *string
		subphase *string
		triggers []string
		want     bool
	}{
		{"blank row applies everywhere", Row{}, &follicular, &early, nil, true},
		{"phase mismatch", Row{CyclePhase: &menstruation}, &follicular, &early, nil, false},
		{"phase set, day has none", Row{CyclePhase: &menstruation}, nil, nil, nil, false},
		{"trigger not active", Row{SymptomTrigger: &headache}, &follicular, &early, []string{"cramps"}, false},
		{"trigger active", Row{SymptomTrigger: &headache}, &follicular, &early, []string{"cramps", "headache"}, true},
		{"subphase listed", Row{CyclePhase: &follicular, CycleSubphases: json.RawMessage(`["early_follicular"]`)}, &follicular, &early, nil, true},
		{"subphase not listed", Row{CyclePhase: &follicular, CycleSubphases: json.RawMessage(`["early_follicular"]`)}, &follicular, &mid, nil, false},
		{"empty list = every subphase", Row{CycleSubphases: json.RawMessage(`[]`)}, &follicular, &mid, nil, true},
		{"null subphase with list", Row{CycleSubphases: json.RawMessage(`["mid_follicular"]`)}, &follicular, nil, nil, false},
		{"strict in_array: non-string entries never match", Row{CycleSubphases: json.RawMessage(`[1]`)}, &follicular, strp("1"), nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.row.AppliesTo(tc.phase, tc.subphase, tc.triggers))
		})
	}
}

type fakeSource struct {
	rows       []Row
	exists     bool
	loads      int
	existsHits int
	err        error
}

func (f *fakeSource) ActiveRows(context.Context) ([]Row, error) {
	f.loads++
	return f.rows, f.err
}

func (f *fakeSource) AnyExists(context.Context) (bool, error) {
	f.existsHits++
	return f.exists, nil
}

type log struct{ headache *string }

func (l log) HeadacheIntensity() *string    { return l.headache }
func (l log) PelvicPainIntensity() *string  { return nil }
func (l log) StomachAcheIntensity() *string { return nil }
func (l log) SleepQuality() *string         { return nil }
func (l log) Moods() []string               { return nil }
func (l log) BloatingIntensity() *string    { return nil }
func (l log) Fatigue() *bool                { return nil }

func text(en string) json.RawMessage { return json.RawMessage(`{"fa":"` + en + `","en":"` + en + `"}`) }

func TestRepository_ForDayOrdersTriggeredFirstAndMemoises(t *testing.T) {
	headache, luteal := "headache", "luteal"
	src := &fakeSource{rows: []Row{
		{ID: 1, Type: "nutrition", Text: text("plain-1"), CyclePhase: &luteal},
		{ID: 2, Type: "pain_relief", Text: text("headache"), SymptomTrigger: &headache},
		{ID: 3, Type: "sleep", Text: text("plain-2"), CycleSubphases: json.RawMessage(`["mid_luteal"]`)},
		{ID: 4, Type: "mood", Text: text("other-subphase"), CycleSubphases: json.RawMessage(`["late_luteal"]`)},
	}}
	repo := New(src)
	ctx := context.Background()

	has, err := repo.HasContent(ctx)
	require.NoError(t, err)
	assert.True(t, has)
	assert.Zero(t, src.existsHits, "a loaded set answers hasContent for free")

	tips, err := repo.ForDay(ctx, enums.CyclePhaseLuteal, enums.CycleSubphaseMidLuteal, log{headache: strp("mild")})
	require.NoError(t, err)
	var texts []string
	for _, tip := range tips {
		texts = append(texts, tip.EN)
	}
	assert.Equal(t, []string{"headache", "plain-1", "plain-2"}, texts)

	// No log → no triggers; the aliases collapse onto their canonical sub-phase.
	tips, err = repo.ForDay(ctx, enums.CyclePhaseLuteal, enums.CycleSubphaseMidLuteal, nil)
	require.NoError(t, err)
	assert.Len(t, tips, 2)
	assert.Equal(t, 1, src.loads, "one load per request")
}

func TestRepository_HasContentCountsInactiveRows(t *testing.T) {
	src := &fakeSource{exists: true}
	repo := New(src)
	has, err := repo.HasContent(context.Background())
	require.NoError(t, err)
	assert.True(t, has)

	has, err = New(&fakeSource{}).HasContent(context.Background())
	require.NoError(t, err)
	assert.False(t, has)

	tips, err := repo.ForDay(context.Background(), enums.CyclePhaseLuteal, enums.CycleSubphaseMidLuteal, nil)
	require.NoError(t, err)
	assert.Empty(t, tips)
}

func TestRepository_LoadErrorPropagates(t *testing.T) {
	repo := New(&fakeSource{err: errors.New("db down")})
	_, err := repo.HasContent(context.Background())
	require.ErrorContains(t, err, "db down")
	_, err = repo.ForDay(context.Background(), enums.CyclePhaseLuteal, enums.CycleSubphaseMidLuteal, nil)
	require.ErrorContains(t, err, "db down")
}

func TestRepository_Signature(t *testing.T) {
	a, err := New(&fakeSource{rows: []Row{{ID: 1, Type: "rest", Text: text("a")}}}).Signature(context.Background())
	require.NoError(t, err)
	b, err := New(&fakeSource{rows: []Row{{ID: 1, Type: "rest", Text: text("b")}}}).Signature(context.Background())
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.Len(t, a, 32)
}
