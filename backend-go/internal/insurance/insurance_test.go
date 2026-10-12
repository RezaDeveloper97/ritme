package insurance

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/insurance/store"
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

func nd(y int, m time.Month, d int) civildate.NullDate {
	return civildate.NullDate{Date: civildate.New(y, m, d), Valid: true}
}

func TestPolicyHistory(t *testing.T) {
	today := civildate.New(2026, 10, 12)
	old := PolicyFact{ID: 1, Period: Period{From: nd(2025, 3, 21), To: nd(2026, 3, 20)}}
	cur := PolicyFact{ID: 2, Period: Period{From: nd(2026, 3, 21), To: nd(2027, 3, 20)}}
	next := PolicyFact{ID: 3, Period: Period{From: nd(2027, 3, 21), To: nd(2028, 3, 19)}}

	assert.Zero(t, CurrentOf(nil, today))
	ps := []PolicyFact{old, next, cur}
	assert.Equal(t, uint64(2), CurrentOf(ps, today), "the period containing today")
	assert.Equal(t, uint64(3), NewestOf(ps))
	assert.Equal(t, PolicyPast, StatusOf(old, 2, today))
	assert.Equal(t, PolicyCurrent, StatusOf(cur, 2, today))
	assert.Equal(t, PolicyUpcoming, StatusOf(next, 2, today))

	// Nothing contains today (expired, not renewed) → the newest one stays current.
	assert.Equal(t, uint64(1), CurrentOf([]PolicyFact{old}, today))
	// Overlapping periods → the latest start wins; an open start is the oldest.
	open := PolicyFact{ID: 9, Period: Period{}}
	assert.Equal(t, uint64(2), CurrentOf([]PolicyFact{open, cur}, today))
	assert.Equal(t, uint64(9), CurrentOf([]PolicyFact{open}, today))

	// Renewal: the day after the old end, same length; open periods → today, one year.
	r := RenewalPeriod(cur.Period, today)
	assert.Equal(t, "2027-03-21", r.From.Date.String())
	assert.Equal(t, "2028-03-19", r.To.Date.String())
	r = RenewalPeriod(Period{}, today)
	assert.Equal(t, "2026-10-12", r.From.Date.String())
	assert.Equal(t, "2027-10-11", r.To.Date.String())
}

// fakeSeal is a reversible test sealer bound to the additional data.
type fakeSeal struct{ off bool }

func (f fakeSeal) SealWith(plain string, ad []byte) (string, error) {
	return "sealed:" + string(ad) + ":" + hexOf(plain), nil
}
func (f fakeSeal) OpenWith(stored string, ad []byte) (string, error) {
	prefix := "sealed:" + string(ad) + ":"
	if !strings.HasPrefix(stored, prefix) {
		return "", errors.New("wrong owner")
	}
	b, err := hex.DecodeString(strings.TrimPrefix(stored, prefix))
	return string(b), err
}
func (f fakeSeal) Disabled() bool { return f.off }

func hexOf(s string) string { return hex.EncodeToString([]byte(s)) }

func TestQuestionnaireEvidenceSealed(t *testing.T) {
	svc := NewService(nil, fakeSeal{})
	f := RecordFacts{HasLifeProfile: true, Chronic: []string{"thyroid"}, FamilyHistory: []string{}}
	as := Prefill(f, civildate.New(2026, 9, 23))
	stripped, enc, err := svc.sealEvidence(7, as)
	require.NoError(t, err)
	require.True(t, enc.Valid)
	assert.NotContains(t, enc.String, "thyroid")
	assert.Contains(t, enc.String, "insurance:questionnaire:7", "bound to the owner")
	raw, err := json.Marshal(stripped)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "thyroid")
	assert.NotContains(t, string(raw), "evidence")

	row := store.InsuranceQuestionnaire{UserID: 7, Answers: raw, EvidenceEnc: enc}
	assert.Equal(t, []string{"thyroid"}, answerOf(svc.Answers(row), "chronic_disease").Evidence)
	row.UserID = 8 // a seal moved to another account does not open
	assert.Empty(t, answerOf(svc.Answers(row), "chronic_disease").Evidence)

	// A legacy row (plain evidence in answers) stays readable.
	legacy, err := json.Marshal(as)
	require.NoError(t, err)
	assert.Equal(t, []string{"thyroid"}, answerOf(svc.Answers(store.InsuranceQuestionnaire{UserID: 7, Answers: legacy}), "chronic_disease").Evidence)

	// No key: nothing is stored in clear (fail closed); nothing to seal needs no key.
	off := NewService(nil, fakeSeal{off: true})
	_, _, err = off.sealEvidence(7, as)
	require.ErrorIs(t, err, ErrStorage)
	_, enc, err = off.sealEvidence(7, Prefill(RecordFacts{}, civildate.New(2026, 9, 23)))
	require.NoError(t, err)
	assert.False(t, enc.Valid)
	row = store.InsuranceQuestionnaire{UserID: 7, Answers: raw, EvidenceEnc: sql.NullString{String: "x", Valid: true}}
	assert.Empty(t, answerOf(off.Answers(row), "chronic_disease").Evidence)
}
