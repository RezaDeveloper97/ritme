package labs

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// T-M10-20 / D-171: the general ranges, bands, borderline config and per-value confirmation.

func general(t *testing.T, c *Catalog, code, value, unit string, f sheetFacts) Evaluated {
	t.Helper()
	e := evaluate(row(1, code, code, value, "", "", unit), c, f)
	require.Equal(t, RangeTypical, e.Range.Source, code)
	return e
}

func TestHemoglobinByPregnancyAndTrimester(t *testing.T) {
	c := seedCatalog(t)
	notPregnant := sheetFacts{}
	assert.Equal(t, StateBorderlineLow, general(t, c, codeHemoglobin, "11.9", "g/dL", notPregnant).State)
	assert.Equal(t, StateNormal, general(t, c, codeHemoglobin, "12.0", "g/dL", notPregnant).State)
	t1 := sheetFacts{Pregnant: true, Trimester: 1}
	e := general(t, c, codeHemoglobin, "10.9", "g/dL", t1)
	assert.True(t, lowSide(e.State))
	assert.Equal(t, BasisPregnancy, e.Basis)
	assert.Equal(t, StateNormal, general(t, c, codeHemoglobin, "11.0", "g/dL", t1).State)
	t2 := sheetFacts{Pregnant: true, Trimester: 2}
	e = general(t, c, codeHemoglobin, "10.4", "g/dL", t2)
	assert.True(t, lowSide(e.State))
	assert.Equal(t, BasisPregnancyT2, e.Basis)
	assert.Equal(t, StateNormal, general(t, c, codeHemoglobin, "10.5", "g/dL", t2).State)
	// The sheet's own range always wins.
	sheet := evaluate(row(1, codeHemoglobin, "Hb", "11.5", "11", "15", "g/dL"), c, t2)
	assert.Equal(t, RangeSheet, sheet.Range.Source)
	assert.Equal(t, StateNormal, sheet.State)
}

func TestFerritinWithInflammation(t *testing.T) {
	c := seedCatalog(t)
	assert.Equal(t, StateLow, general(t, c, codeFerritin, "13", "ng/mL", sheetFacts{}).State)
	assert.True(t, Attention(general(t, c, codeFerritin, "14", "ng/mL", sheetFacts{}).State))
	assert.Equal(t, StateNormal, general(t, c, codeFerritin, "15", "ng/mL", sheetFacts{}).State)
	inf := sheetFacts{Inflamed: true}
	e := general(t, c, codeFerritin, "69", "ng/mL", inf)
	assert.True(t, lowSide(e.State))
	assert.Equal(t, BasisInflamed, e.Basis)
	assert.Equal(t, StateNormal, general(t, c, codeFerritin, "70", "ng/mL", inf).State)

	// Inflammation is read from the same sheet: CRP above its printed range, confirmed.
	rows := []store.LabMarker{
		row(1, codeFerritin, "Ferritin", "40", "", "", "ng/mL"),
		row(2, "", "hs-CRP", "12", "0", "5", "mg/L"),
	}
	evals := evaluateSheet(rows, c, sheetFacts{})
	assert.Equal(t, StateLow, evals[0].State, "40 < 70 with inflammation")
	rows[1].Value.String = "3"
	assert.Equal(t, StateNormal, evaluateSheet(rows, c, sheetFacts{})[0].State)
	rows[1].Value.String = "12"
	rows[1].Confidence = sql.NullString{String: "0.5", Valid: true}
	assert.Equal(t, StateNormal, evaluateSheet(rows, c, sheetFacts{})[0].State, "an unconfirmed CRP counts nowhere")
	assert.True(t, isCRP("C-Reactive Protein"))
	assert.True(t, isCRP("CRP (quantitative)"))
	assert.False(t, isCRP("Creatinine"))
}

func TestVitaminDBands(t *testing.T) {
	c := seedCatalog(t)
	cases := []struct {
		v     string
		band  string
		state string
	}{
		{"11.9", BandVitDDeficient, StateLow},
		{"12", BandVitDInsufficient, StateLow},
		{"19.9", BandVitDInsufficient, StateLow},
		{"20", BandVitDSufficient, StateNormal},
		{"120", BandVitDSufficient, StateHigh},
	}
	for _, cs := range cases {
		e := general(t, c, codeVitaminD, cs.v, "ng/mL", sheetFacts{})
		assert.Equal(t, cs.band, e.Band, cs.v)
		assert.Equal(t, cs.state, e.State, cs.v)
	}
	l := loc{Locale: "fa", Default: "fa"}
	m := l.markerJSON(general(t, c, codeVitaminD, "15", "ng/mL", sheetFacts{}))
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "کمتر از حد کافی")
	assert.Contains(t, string(raw), "۲۰۲۴")
	assert.Contains(t, string(raw), "محدوده عمومی")
}

func TestTSHInPregnancy(t *testing.T) {
	c := seedCatalog(t)
	p := sheetFacts{Pregnant: true, Trimester: 1}
	assert.Equal(t, StateNormal, general(t, c, codeTSH, "4.0", "mIU/L", p).State)
	e := general(t, c, codeTSH, "4.1", "mIU/L", p)
	assert.True(t, Attention(e.State))
	assert.Equal(t, BasisPregnancy, e.Basis)
}

func TestHbA1cBands(t *testing.T) {
	c := seedCatalog(t)
	for _, cs := range []struct{ v, band, state string }{
		{"5.6", BandHbA1cTypical, StateNormal},
		{"5.7", BandHbA1cRaised, StateHigh},
		{"6.4", BandHbA1cRaised, StateHigh},
		{"6.5", BandHbA1cHigh, StateHigh},
	} {
		e := general(t, c, codeHbA1c, cs.v, "%", sheetFacts{})
		assert.Equal(t, cs.band, e.Band, cs.v)
		assert.Equal(t, cs.state, e.State, cs.v)
	}
	l := loc{Locale: "fa", Default: "fa"}
	label := l.stateLabel(general(t, c, codeHbA1c, "6.0", "%", sheetFacts{}))
	assert.Contains(t, label, "معمولاً پزشک بررسی بیشتری می\u200cخواهد")
	assert.NoError(t, guardOK(label), "a band label never names a condition")
}

func guardOK(s string) error {
	if r := guardText(s); r != "" {
		return &guardErr{r}
	}
	return nil
}

type guardErr struct{ reason string }

func (g *guardErr) Error() string { return g.reason }

func TestConfidenceNeedsConfirmation(t *testing.T) {
	c := seedCatalog(t)
	m := row(1, codeFerritin, "Ferritin", "9", "15", "150", "ng/mL")
	m.Confidence = sql.NullString{String: "0.89", Valid: true}
	e := evaluate(m, c, sheetFacts{})
	assert.True(t, e.Unconfirmed)
	assert.Equal(t, StateUnknown, e.State, "excluded from the status rules")
	_, flag := redFlag(e, "x")
	assert.False(t, flag)
	assert.Empty(t, promptEvals([]Evaluated{e}), "never in the AI payload")

	m.Confidence.String = "0.9"
	e = evaluate(m, c, sheetFacts{})
	assert.False(t, e.Unconfirmed)
	assert.Equal(t, StateLow, e.State)

	m.Confidence.String = "0.5"
	m.ConfirmedAt = sql.NullTime{Time: time.Now(), Valid: true}
	e = evaluate(m, c, sheetFacts{})
	assert.False(t, e.Unconfirmed, "confirmed → included")
	assert.True(t, e.LowConfidence())
	assert.Equal(t, StateLow, e.State)
}

func TestBorderlineConfig(t *testing.T) {
	r := Range{Low: f(12), High: f(15.5)}
	s, near := classifyRange(f(11.2), r, borderline{Enabled: true, Margin: 0.1})
	assert.Equal(t, StateBorderlineLow, s)
	assert.True(t, near)
	s, near = classifyRange(f(11.2), r, borderline{Enabled: false, Margin: 0.1})
	assert.Equal(t, StateLow, s, "no class with enabled 0")
	assert.True(t, near, "only the near-edge hint")
	s, near = classifyRange(f(9), r, borderline{Enabled: false, Margin: 0.1})
	assert.Equal(t, StateLow, s)
	assert.False(t, near)

	// The registry version valid on the lab date is the one used.
	day := civildate.New(2026, 11, 1)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyLabBorderline, Version: 2, EffectiveFrom: day,
		Value: clinicalconfig.Value{"enabled": 0, "margin": 0.1}}})
	t.Cleanup(clinicalconfig.Reset)
	c := seedCatalog(t)
	before := evaluate(row(1, codeHemoglobin, "Hb", "11.2", "12", "15.5", "g/dL"), c, sheetFacts{Date: day.AddDays(-1)})
	after := evaluate(row(1, codeHemoglobin, "Hb", "11.2", "12", "15.5", "g/dL"), c, sheetFacts{Date: day})
	assert.Equal(t, StateBorderlineLow, before.State)
	assert.Equal(t, StateLow, after.State)
	assert.True(t, after.NearEdge)
}

func TestDatingOnTheLabDate(t *testing.T) {
	lmp := civildate.New(2026, 3, 1)
	p := &pregstore.PregnancyProfile{PregnancyMode: true, AgeSource: sql.NullString{String: string(enums.PregnancyAgeSourceLmp), Valid: true},
		LmpDate: civildate.NullDate{Date: lmp, Valid: true}}
	d := dating{calc: calc.New(p, "", civildate.Date{})}
	f := d.facts(lmp.AddDays(13 * 7))
	assert.True(t, f.Pregnant)
	assert.Equal(t, 1, f.Trimester)
	f = d.facts(lmp.AddDays(14 * 7))
	assert.Equal(t, 2, f.Trimester, "trimester by completed days (calc.TrimesterOn)")
	assert.False(t, d.facts(lmp.AddDays(-10)).Pregnant, "a lab before this pregnancy")
	assert.False(t, dating{}.facts(lmp).Pregnant)
}

func TestAIReplyValidationAndFilter(t *testing.T) {
	c := seedCatalog(t)
	evals := []Evaluated{
		evaluate(row(1, codeFerritin, "Ferritin", "9", "15", "150", "ng/mL"), c, sheetFacts{}),
		evaluate(row(2, codeTSH, "TSH", "2", "0.4", "4", "mIU/L"), c, sheetFacts{}),
		evaluate(row(3, "", "Urine colour", "1", "2", "4", ""), c, sheetFacts{}),
	}
	good := `{"summary":"ذخیره آهن کمی پایین است. نتیجه را با پزشکت در میان بگذار.","out_of_range":[{"code":"ferritin","explanation":"ممکن است با خونریزی پریود مرتبط باشد."},{"code":"r3","explanation":"ممکن است تکرار لازم باشد."}],"doctor_questions":["چند وقت دیگر تکرار کنم؟"]}`
	res, reason := parseAIReply(good, evals, nil)
	require.Empty(t, reason)
	assert.Contains(t, res.Summary, "پزشکت")
	require.Len(t, res.Explanations, 2)
	assert.Equal(t, uint64(1), res.Explanations[0].MarkerID)
	assert.Equal(t, uint64(3), res.Explanations[1].MarkerID)
	assert.Equal(t, []string{"چند وقت دیگر تکرار کنم؟"}, res.Questions)
	_, reason = parseAIReply("```json\n"+good+"\n```", evals, nil)
	assert.Empty(t, reason, "a fenced JSON answer is accepted")

	for name, cs := range map[string]struct{ reply, reason string }{
		"diagnosis fa":   {`{"summary":"دیابت داری.","out_of_range":[],"doctor_questions":[]}`, FilterDiagnostic},
		"diagnosis en":   {`{"summary":"You may have anaemia.","out_of_range":[],"doctor_questions":[]}`, FilterDiagnostic},
		"dose en":        {`{"summary":"Please take 500 mg of iron daily.","out_of_range":[],"doctor_questions":[]}`, FilterDose},
		"dose fa digits": {`{"summary":"روزی ۵۰۰۰۰ واحد بخور.","out_of_range":[],"doctor_questions":[]}`, FilterDose},
		"tablet fa":      {`{"summary":"خوب است.","out_of_range":[],"doctor_questions":["آیا قرص آهن بخورم؟"]}`, FilterDose},
		"in explanation": {`{"summary":"خوب است.","out_of_range":[{"code":"ferritin","explanation":"نشانه کم خونی است."}],"doctor_questions":[]}`, FilterDiagnostic},
		"category":       {`{"summary":"خوب است.","out_of_range":[{"code":"tsh","explanation":"بالاست."}],"doctor_questions":[]}`, FilterCategory},
		"unknown code":   {`{"summary":"خوب است.","out_of_range":[{"code":"xyz","explanation":"."}],"doctor_questions":[]}`, FilterCategory},
		"not json":       {`ذخیره آهن پایین است.`, FilterInvalid},
		"no summary":     {`{"out_of_range":[],"doctor_questions":[]}`, FilterInvalid},
		"extra key":      {`{"summary":"x","diagnosis":"y"}`, FilterInvalid},
		"two objects":    {`{"summary":"x"}{"summary":"y"}`, FilterInvalid},
	} {
		_, reason := parseAIReply(cs.reply, evals, nil)
		assert.Equal(t, cs.reason, reason, name)
	}
	// A lab unit is not a dose.
	_, reason = parseAIReply(`{"summary":"قند ۹۵ mg/dL است و TSH ۲ mIU/L.","out_of_range":[],"doctor_questions":[]}`, evals, nil)
	assert.Empty(t, reason)
	// Names are redacted and links stripped.
	res, _ = parseAIReply(`{"summary":"Sara, see https://x.example ok","out_of_range":[],"doctor_questions":[]}`, evals, []string{"Sara"})
	assert.NotContains(t, res.Summary, "Sara")
	assert.NotContains(t, res.Summary, "https")
}

func TestGuardWordLists(t *testing.T) {
	for _, list := range [][]string{diagnosticTerms, doseWords, doseUnits} {
		seen := map[string]bool{}
		for _, w := range list {
			assert.NotEmpty(t, w)
			assert.False(t, seen[w], "duplicate %q", w)
			seen[w] = true
		}
	}
	for _, hit := range []string{"دیابت داری", "پیش\u200cدیابت", "کم خونی", "Hypothyroidism", "take 500 mg", "۲ عدد قرص", "1000 IU daily"} {
		assert.NotEmpty(t, guardText(hit), hit)
	}
	for _, clean := range []string{"ذخیره آهن کمی پایین است", "Your ferritin is a little low", "95 mg/dL", "TSH 2.1 mIU/L",
		"با پزشکت در میان بگذار", "kidney-friendly diet is not mentioned"} {
		assert.Empty(t, guardText(clean), clean)
	}
}

func TestPromptLeavesUnconfirmedOut(t *testing.T) {
	c := seedCatalog(t)
	m := row(2, codeVitaminD, "Vitamin D", "14", "30", "100", "ng/mL")
	m.Confidence = sql.NullString{String: "0.88", Valid: true}
	evals := promptEvals([]Evaluated{
		evaluate(row(1, codeFerritin, "Ferritin", "9", "15", "150", "ng/mL"), c, sheetFacts{}),
		evaluate(m, c, sheetFacts{}),
	})
	req := buildPrompt(evals, UserContext{Mode: enums.LifeModeCycle}, "en", nil)
	assert.NotContains(t, req.Messages[0].Text, "Vitamin D")
	assert.Contains(t, req.Messages[0].Text, `"code": "ferritin"`)
	assert.Contains(t, req.System, `"out_of_range"`)
}
