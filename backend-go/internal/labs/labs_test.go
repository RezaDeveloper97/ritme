package labs

import (
	"encoding/json"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func f(v float64) *float64 { return &v }

func TestClassify(t *testing.T) {
	r := Range{Low: f(12), High: f(15.5)}
	cases := []struct {
		v    *float64
		r    Range
		want string
	}{
		{f(11.2), r, StateBorderlineLow}, // nbl_Lab_Result «پایین مرزی»
		{f(9), Range{Low: f(15), High: f(150)}, StateLow},
		{f(13), r, StateNormal},
		{f(12), r, StateNormal},
		{f(16.5), r, StateBorderlineHigh},
		{f(20), r, StateHigh},
		{f(210), Range{High: f(200)}, StateBorderlineHigh}, // one-sided «< 200»
		{f(18), Range{Low: f(30)}, StateLow},
		{nil, r, StateUnknown},
		{f(5), Range{}, StateUnknown},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, Classify(c.v, c.r), "%v %+v", c.v, c.r)
	}
	assert.True(t, Attention(StateBorderlineLow))
	assert.False(t, Attention(StateUnknown))
}

func TestDirection(t *testing.T) {
	assert.Equal(t, TrendFalling, Direction([]float64{32, 18, 9})) // nbl_An_Hub «۳۲ ← ۱۸ ← ۹»
	assert.Equal(t, TrendRising, Direction([]float64{1, 2}))
	assert.Equal(t, TrendStable, Direction([]float64{10, 100, 101, 102})) // last 3 only
	assert.Empty(t, Direction([]float64{5}))
}

func TestUnitsAndNumbers(t *testing.T) {
	assert.True(t, UnitsMatch("10^3/µL", "×10³/uL"))
	assert.True(t, UnitsMatch("mg/dl", "mg/dL"))
	assert.True(t, UnitsMatch("ug/dL", "µg/dL"))
	assert.False(t, UnitsMatch("mmol/L", "mg/dL"))
	assert.False(t, UnitsMatch("", ""))
	n, ok := parseNumber("۱۱٫۲")
	require.True(t, ok)
	assert.InDelta(t, 11.2, n, 1e-9)
	assert.Equal(t, "vitamind25oh", normalizeName("Vitamin D (25-OH)"))
	assert.Equal(t, "ویتامیند", normalizeName("ويتامين د"))
}

// seedCatalog reads the lab_markers seed from migration 00034 (the rows tests and the API see).
func seedCatalog(t *testing.T) *Catalog {
	t.Helper()
	raw, err := fs.ReadFile(os.DirFS("../../db/migrations"), "00034_labs.sql")
	require.NoError(t, err)
	var items []catalog.Item
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "('lab_markers', '") {
			code := strings.SplitN(strings.TrimPrefix(line, "('lab_markers', '"), "'", 2)[0]
			items = append(items, catalog.Item{Code: code})
			continue
		}
		if len(items) == 0 || !strings.HasPrefix(line, "'{") {
			continue
		}
		js := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(line, ", 1,"), ","), "'")
		js = strings.ReplaceAll(strings.TrimPrefix(js, "'"), "''", "'")
		it := &items[len(items)-1]
		switch {
		case it.Title == nil:
			it.Title = json.RawMessage(js)
		case it.Body == nil:
			it.Body = json.RawMessage(js)
		default:
			it.Meta = json.RawMessage(js)
		}
	}
	require.Len(t, items, 26)
	c, bad := NewCatalog(items)
	require.Empty(t, bad, "every seeded meta decodes")
	return c
}

func TestCatalogSeedAndMatch(t *testing.T) {
	c := seedCatalog(t)
	for _, code := range []string{"hemoglobin", "ferritin", "tsh", "vitamin_d", "b12", "glucose_fasting", "hba1c", "ldl", "hdl",
		"triglycerides", "fsh", "lh", "estradiol", "progesterone", "prolactin", "amh"} {
		m, ok := c.ByCode(code)
		require.True(t, ok, code)
		assert.True(t, m.NeedsReview == false || m.NeedsReview, code) // needs_review is carried, value from the row
		require.NotNil(t, m.Meta.Low, code)
		require.NotNil(t, m.Meta.High, code)
		assert.NotEmpty(t, m.Meta.Unit, code)
	}
	match := map[string]string{
		"Hemoglobin": "hemoglobin", "Hb": "hemoglobin", "HbA1c": "hba1c", "Hemoglobin A1c": "hba1c",
		"Vitamin D (25-OH)": "vitamin_d", "FBS": "glucose_fasting", "Ferritin": "ferritin", "TSH": "tsh",
		"هموگلوبین": "hemoglobin", "فریتین": "ferritin", "Free T4": "free_t4", "LDL-C": "ldl", "Prolactin": "prolactin",
	}
	for name, want := range match {
		m, ok := c.Match(name)
		require.True(t, ok, name)
		assert.Equal(t, want, m.Code, name)
	}
	_, ok := c.Match("Urine colour")
	assert.False(t, ok)
	fsh, _ := c.ByCode("fsh")
	assert.True(t, fsh.TypicalRange().Empty(), "phase-dependent ranges never classify")
}

func row(id uint64, code, name string, value, low, high string, unit string) store.LabMarker {
	m := store.LabMarker{ID: id, Name: name, Source: MarkerExtracted}
	if code != "" {
		m.Code.String, m.Code.Valid = code, true
	}
	m.Value.String, m.Value.Valid = value, value != ""
	m.RefLow.String, m.RefLow.Valid = low, low != ""
	m.RefHigh.String, m.RefHigh.Valid = high, high != ""
	m.Unit.String, m.Unit.Valid = unit, unit != ""
	return m
}

func TestEvaluateRedFlagsQuestionsSummary(t *testing.T) {
	c := seedCatalog(t)
	evals := []Evaluated{
		evaluate(row(1, "ferritin", "Ferritin", "9", "15", "150", "ng/mL"), c),
		evaluate(row(2, "hemoglobin", "Hb", "6.5", "12", "15.5", "g/dL"), c),
		evaluate(row(3, "tsh", "TSH", "2.1", "0.4", "4", "mIU/L"), c),
		evaluate(row(4, "glucose_fasting", "FBS", "5.2", "", "", "mmol/L"), c), // other unit: no typical range, no flag
		evaluate(row(5, "vitamin_d", "Vit D", "18", "", "", "ng/mL"), c),       // no sheet range: typical (units match)
	}
	assert.Equal(t, StateLow, evals[0].State)
	assert.Equal(t, StateLow, evals[1].State)
	assert.Equal(t, StateNormal, evals[2].State)
	assert.Equal(t, StateUnknown, evals[3].State)
	assert.Equal(t, StateLow, evals[4].State)
	assert.Equal(t, RangeTypical, evals[4].Range.Source)

	l := loc{Locale: "fa", Default: "fa"}
	flags := redFlagsOf(evals, l)
	require.Len(t, flags, 1)
	assert.Equal(t, SeverityUrgent, flags[0].Severity)
	assert.Equal(t, "هموگلوبین", flags[0].Name)

	counts := countStates(evals)
	assert.Equal(t, Counts{Total: 5, Normal: 1, Attention: 3, Unknown: 1, Low: 3}, counts)

	qs := questions(evals, l)
	assert.LessOrEqual(t, len(qs), MaxQuestions)
	assert.Contains(t, qs, "آیا برای ذخیره آهن کم باید مکمل آهن بگیرم؟ چه مقدار و تا کی؟")

	sum := rulesSummary(evals, l)
	assert.Contains(t, sum, "فریتین (پایین)")
	assert.Contains(t, sum, "پزشک")
	en := rulesSummary(evals[2:3], loc{Locale: "en", Default: "fa"})
	assert.Contains(t, en, "All 1 markers")
}

func TestPromptCarriesNoIdentity(t *testing.T) {
	c := seedCatalog(t)
	evals := []Evaluated{evaluate(row(1, "ferritin", "Ferritin", "9", "15", "150", "ng/mL"), c)}
	evals[0].Range.Text = "15–150"
	uc := UserContext{Mode: enums.LifeModeTTC, Age: 31, Phase: "luteal", Medications: []string{"Levothyroxine"}, Names: []string{"Sara Ahmadi"}}
	req := buildPrompt(evals, uc, "fa", nil)
	text := req.System + req.Messages[0].Text
	assert.NotContains(t, text, "Sara")
	assert.NotContains(t, text, "31") // age band only
	assert.Contains(t, text, "30–34")
	assert.Contains(t, text, "trying to conceive")
	assert.Contains(t, text, "Levothyroxine")
	assert.Contains(t, text, "Ferritin (ferritin): 9 ng/mL [lab range 15–150] → low")
	assert.Contains(t, req.System, "not a doctor")
	assert.Contains(t, req.System, "ISO code is fa")
	assert.Equal(t, summaryMaxTokens, req.MaxOutputTokens)
}

func TestMarkersFromMergesPages(t *testing.T) {
	today := civildate.New(2026, 10, 5)
	page1 := ai.FakeExtractions["lab_panel"]
	page2 := ai.Extraction{
		Fields: []ai.ExtractedField{{Key: "date", Value: "2027-01-01", Confidence: 1}}, // future: ignored
		Items: [][]ai.ExtractedField{
			{{Key: "marker", Value: "Ferritin", Confidence: 1}, {Key: "value", Value: 50.0, Confidence: 1}}, // duplicate name
			{{Key: "marker", Value: "Urine protein", Confidence: 0.8}, {Key: "value_text", Value: "Negative", Confidence: 0.8}},
			{{Key: "marker", Value: "Nothing", Confidence: 0.9}}, // no value: dropped
			{{Key: "marker", Value: "Odd", Confidence: 0.9}, {Key: "value", Value: 3.0, Confidence: 0.4},
				{Key: "ref_low", Value: 9.0, Confidence: 1}, {Key: "ref_high", Value: 1.0, Confidence: 1}},
		},
	}
	rows, date, labName := markersFrom([]ai.Extraction{page1, page2}, today)
	require.Len(t, rows, 7)
	assert.Equal(t, "2026-09-20", date.Date.String())
	assert.Equal(t, "آزمایشگاه نمونه", labName)
	assert.InDelta(t, 9, *rows[1].in.Value, 1e-9, "first page wins")
	assert.Equal(t, "Negative", rows[5].in.ValueText)
	assert.InDelta(t, 0.4, rows[6].confidence, 1e-9, "the lowest of name / value confidence")
	assert.Nil(t, rows[6].in.RefLow, "inverted bounds dropped")
	assert.InDelta(t, 0.62, rows[4].confidence, 1e-9)
}

func TestPhaseAndAge(t *testing.T) {
	start := civildate.New(2026, 9, 10)
	assert.Equal(t, "menstruation", phaseOn(start, 28, 5, civildate.New(2026, 9, 12)))
	assert.Equal(t, "luteal", phaseOn(start, 28, 5, civildate.New(2026, 10, 5)))
	assert.Equal(t, "menstruation", phaseOn(start, 28, 5, civildate.New(2026, 10, 9)), "rolled forward one cycle")
	assert.Empty(t, phaseOn(start, 28, 5, civildate.New(2027, 6, 1)), "too old to roll")
	assert.Equal(t, 31, ageOn(civildate.New(1995, 3, 1), civildate.New(2026, 10, 5)))
	assert.Equal(t, "30–34", ageBand(31))
}

func TestLangFilesHaveTheSameKeys(t *testing.T) {
	read := func(code string) map[string]any {
		raw, err := langFS.ReadFile("lang/" + code + "/labs.json")
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		return m
	}
	var keys func(prefix string, m map[string]any) []string
	keys = func(prefix string, m map[string]any) []string {
		var out []string
		for k, v := range m {
			if sub, ok := v.(map[string]any); ok {
				out = append(out, keys(prefix+k+".", sub)...)
			} else {
				out = append(out, prefix+k)
			}
		}
		return out
	}
	assert.ElementsMatch(t, keys("", read("en")), keys("", read("fa")))
	for _, s := range []string{StateLow, StateBorderlineLow, StateNormal, StateBorderlineHigh, StateHigh, StateUnknown} {
		assert.NotEqual(t, "labs.states."+s, T("states."+s, "fa", nil))
	}
	for _, code := range []string{CodeAIFailed, CodeAIUnavailable, CodeAIBudget, CodeUnreadable, CodeNothingRead, CodeConsent, CodeInvalidFile, CodeInternal} {
		assert.NotEqual(t, "labs.errors."+code, T("errors."+code, "en", nil), code)
	}
	for _, cat := range Categories {
		assert.NotEqual(t, "labs.categories."+cat, T("categories."+cat, "fa", nil), cat)
	}
}

func TestLabSchemaIsAccepted(t *testing.T) {
	// The platform refuses identity keys; the fake answers with the same keys.
	client := ai.NewClientWith(ai.Options{Provider: "fake", Extractor: ai.NewFake()})
	pdf := []byte("%PDF-1.4 RITME-FAKE:lab_panel")
	ex, err := client.Extract(t.Context(), ai.FeatureLabAnalysis, ai.ExtractRequest{Document: ai.Document{Data: pdf, MIME: "application/pdf"}, Schema: LabSchema})
	require.NoError(t, err)
	assert.Len(t, ex.Items, 5)
}
