package insurance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestLangFilesLoad(t *testing.T) {
	assert.Equal(t, "Claim saved", T("messages.claim_saved", "en"))
	assert.Equal(t, "خسارت ذخیره شد", T("messages.claim_saved", "fa"))
	assert.NotEmpty(t, attributes("fa"))
	assert.NotEmpty(t, attributes("en"))
}

func TestLast4(t *testing.T) {
	assert.Equal(t, "4821", Last4("PN-99-۴۸۲۱"))
	assert.Equal(t, "1234", Last4("IR12 0000 0000 0000 1234"))
	assert.Equal(t, "AB1", Last4("ab-1"))
	assert.Empty(t, Last4("  -- "))
}

func TestCategoryOf(t *testing.T) {
	assert.Equal(t, CatVisitDrug, CategoryOf(CostVisit))
	assert.Equal(t, CatVisitDrug, CategoryOf(CostDrug))
	assert.Equal(t, CatParaclinic, CategoryOf(CostParaclinic))
	assert.Equal(t, CatOther, CategoryOf("nope"))
	for _, c := range CostTypes {
		assert.Contains(t, Categories, CategoryOf(c))
	}
}

func i64(v int64) *int64 { return &v }

func TestUsageAndPayable(t *testing.T) {
	from := civildate.New(2026, 3, 21)
	p := Period{From: civildate.NullDate{Date: from, Valid: true}, To: civildate.NullDate{Date: civildate.New(2027, 3, 20), Valid: true}}
	cov := Coverage{Category: CatParaclinic, Status: CoverageCovered, Cap: i64(20_000_000), DeductiblePct: 30, WaitingDays: 270}
	claims := []ClaimFact{
		{ID: 1, CostType: CostParaclinic, ServiceDate: civildate.New(2026, 9, 1), Status: ClaimPaid, Paid: 6_000_000},
		{ID: 2, CostType: CostParaclinic, ServiceDate: civildate.New(2026, 9, 2), Status: ClaimReviewing},
		{ID: 3, CostType: CostParaclinic, ServiceDate: civildate.New(2025, 9, 2), Status: ClaimPaid, Paid: 9},    // before the period
		{ID: 4, CostType: CostVisit, ServiceDate: civildate.New(2026, 9, 2), Status: ClaimPaid, Paid: 1_000_000}, // other category
	}
	today := civildate.New(2026, 9, 23)
	u := UsageOf(cov, p, claims, 0, today)
	assert.Equal(t, int64(6_000_000), u.Used)
	assert.Equal(t, int64(14_000_000), *u.Remaining)
	assert.Equal(t, "2026-12-16", u.WaitingEndsOn.Date.String())
	assert.False(t, u.WaitingOver)
	assert.Equal(t, int64(0), UsageOf(cov, p, claims, 1, today).Used, "the claim itself is excluded")

	pay, ok := ExpectedPayable(1_800_000, 0, &cov, u.Remaining)
	require.True(t, ok)
	assert.Equal(t, int64(540_000), pay.Deductible)
	assert.Equal(t, int64(1_260_000), pay.Payable)
	assert.Equal(t, int64(540_000), pay.UserShare)

	pay, _ = ExpectedPayable(1_000, 200, &cov, i64(100))
	assert.Equal(t, int64(100), pay.Payable)
	assert.True(t, pay.Capped)
	assert.Equal(t, int64(900), pay.UserShare)

	pay, ok = ExpectedPayable(500, 0, &Coverage{Status: CoverageNotCovered}, nil)
	assert.True(t, ok)
	assert.Equal(t, int64(0), pay.Payable)
	assert.Equal(t, int64(500), pay.UserShare)
	_, ok = ExpectedPayable(500, 0, nil, nil)
	assert.False(t, ok)
}

func TestStateOf(t *testing.T) {
	assert.Equal(t, ClaimSubmitted, StateOf(nil).Status)
	d := func(day int) civildate.Date { return civildate.New(2026, 9, day) }
	st := StateOf([]EventFact{
		{ID: 1, Status: ClaimSubmitted, OccurredOn: d(3)},
		{ID: 3, Status: ClaimDocsRequested, OccurredOn: d(6), Deadline: civildate.NullDate{Date: d(16), Valid: true}},
		{ID: 2, Status: ClaimReviewing, OccurredOn: d(5)},
	})
	assert.Equal(t, ClaimDocsRequested, st.Status)
	assert.True(t, st.Deadline.Valid)
	assert.Nil(t, st.Paid)
	st = StateOf([]EventFact{{ID: 1, Status: ClaimSubmitted, OccurredOn: d(3)}, {ID: 2, Status: ClaimPaid, OccurredOn: d(3), Amount: i64(7)}})
	assert.Equal(t, ClaimPaid, st.Status)
	assert.Equal(t, int64(7), *st.Paid)
	assert.False(t, st.Deadline.Valid)

	assert.True(t, MatchesFilter(ClaimAppealed, FilterOpen))
	assert.False(t, MatchesFilter(ClaimPaid, FilterOpen))
	assert.True(t, MatchesFilter(ClaimDocsRequested, FilterNeedsAction))
	assert.True(t, MatchesQuery("ULTRA", "Pregnancy ultrasound"))
	assert.False(t, MatchesQuery("x", "abc"))
}

func answerOf(as []Answer, code string) Answer {
	for _, a := range as {
		if a.Code == code {
			return a
		}
	}
	return Answer{}
}

func TestPrefill(t *testing.T) {
	today := civildate.New(2026, 9, 23)
	empty := Prefill(RecordFacts{}, today)
	require.Len(t, empty, len(Questions))
	for _, a := range empty {
		assert.Nil(t, a.Answer, a.Code+": nothing known → she answers")
	}

	tr := true
	f := RecordFacts{HasLifeProfile: true, LifeMode: "cycle", Chronic: []string{"thyroid"}, Medications: []string{},
		Surgeries: []Surgery{{Title: "Appendix", Date: "2019-01-01"}}, FamilyHistory: []string{"diabetes"},
		Cesareans: []civildate.Date{civildate.New(2024, 5, 1)}, PregnancyMode: &tr}
	as := Prefill(f, today)
	assert.Equal(t, AnswerYes, *answerOf(as, "chronic_disease").Answer)
	assert.Equal(t, []string{"thyroid"}, answerOf(as, "chronic_disease").Evidence)
	assert.Equal(t, AnswerNo, *answerOf(as, "regular_medication").Answer)
	h := answerOf(as, "hospitalized_5y")
	assert.Equal(t, AnswerYes, *h.Answer)
	assert.Equal(t, []string{"cesarean · 2024-05-01"}, h.Evidence, "the 2019 surgery is older than five years")
	assert.Equal(t, AnswerYes, *answerOf(as, "pregnant").Answer)
	assert.Equal(t, AnswerYes, *answerOf(as, "gyn_surgery").Answer)
	assert.Equal(t, AnswerNo, *answerOf(as, "infertility_treatment").Answer)
	assert.Equal(t, AnswerYes, *answerOf(as, "family_history").Answer)
	assert.True(t, answerOf(as, "family_history").FromRecord())
	assert.False(t, Complete(as))

	raw, err := json.Marshal(as)
	require.NoError(t, err)
	back := DecodeAnswers(raw)
	assert.Equal(t, as, back)
	no := AnswerNo
	back[0].Answer = &no
	assert.False(t, back[0].FromRecord())
	assert.Equal(t, "history", SectionOf("pregnant"))
	assert.Empty(t, DecodeTimeline(nil))
}
