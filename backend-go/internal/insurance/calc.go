package insurance

import (
	"sort"
	"strings"
	"unicode"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// CategoryOf maps a claim cost type to the coverage category whose cap it uses.
func CategoryOf(costType string) string {
	switch costType {
	case CostVisit, CostDrug:
		return CatVisitDrug
	case CostParaclinic, CostInpatient, CostDelivery, CostDental, CostOptical, CostInfertility:
		return costType
	}
	return CatOther
}

// Last4 keeps only the last four letters / digits of a policy number or IBAN (Persian / Arabic digits → ASCII); ""
// when nothing is left. The full value is never stored.
func Last4(s string) string {
	var b []rune
	for _, r := range s {
		switch {
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		}
		if r < unicode.MaxASCII && (unicode.IsDigit(r) || unicode.IsLetter(r)) {
			b = append(b, unicode.ToUpper(r))
		}
	}
	if len(b) > 4 {
		b = b[len(b)-4:]
	}
	return string(b)
}

// Period is the policy validity window (open ends are unbounded).
type Period struct{ From, To civildate.NullDate }

// Contains reports whether d is inside the window.
func (p Period) Contains(d civildate.Date) bool {
	if p.From.Valid && d.Before(p.From.Date) {
		return false
	}
	if p.To.Valid && d.After(p.To.Date) {
		return false
	}
	return true
}

// ClaimFact is what the coverage maths needs of one claim.
type ClaimFact struct {
	ID          uint64
	CostType    string
	ServiceDate civildate.Date
	Status      string
	Paid        int64 // paid amount, 0 when not paid
}

// Coverage is one category's terms.
type Coverage struct {
	Category      string
	Status        string
	Cap           *int64
	DeductiblePct int64
	WaitingDays   int64
}

// Usage is the computed state of one category in the policy period.
type Usage struct {
	Used          int64
	Remaining     *int64
	WaitingEndsOn civildate.NullDate
	WaitingOver   bool
}

// UsageOf sums the paid claims of the category inside the period (skipping claim `except`) and derives the remaining
// cap and the waiting period state on `today`.
func UsageOf(c Coverage, p Period, claims []ClaimFact, except uint64, today civildate.Date) Usage {
	var u Usage
	for _, cl := range claims {
		if cl.ID == except || cl.Status != ClaimPaid || CategoryOf(cl.CostType) != c.Category || !p.Contains(cl.ServiceDate) {
			continue
		}
		u.Used += cl.Paid
	}
	if c.Cap != nil {
		r := max(*c.Cap-u.Used, 0)
		u.Remaining = &r
	}
	u.WaitingOver = true
	if c.WaitingDays > 0 && p.From.Valid {
		end := p.From.Date.AddDays(int(c.WaitingDays))
		u.WaitingEndsOn = civildate.NullDate{Date: end, Valid: true}
		u.WaitingOver = !today.Before(end)
	}
	return u
}

// Payable is the expected payout breakdown of a claim.
type Payable struct {
	Invoice    int64
	OverTariff int64
	Deductible int64
	Payable    int64
	UserShare  int64
	Pct        int64
	Capped     bool
}

// ExpectedPayable estimates what the insurer pays: invoice − over-tariff − deductible %, limited by the remaining
// cap of the category (other claims' payouts) and zero when not covered. ok=false when the policy has no terms for
// the category (unknown).
func ExpectedPayable(amount, overTariff int64, cov *Coverage, remaining *int64) (Payable, bool) {
	p := Payable{Invoice: amount, OverTariff: min(overTariff, amount)}
	if cov == nil {
		return p, false
	}
	base := amount - p.OverTariff
	if cov.Status == CoverageNotCovered {
		p.UserShare = amount
		return p, true
	}
	p.Pct = cov.DeductiblePct
	p.Deductible = (base*p.Pct + 50) / 100
	p.Payable = base - p.Deductible
	if remaining != nil && p.Payable > *remaining {
		p.Payable, p.Capped = *remaining, true
	}
	p.UserShare = amount - p.Payable
	return p, true
}

// EventFact is one timeline step.
type EventFact struct {
	ID         uint64
	Status     string
	OccurredOn civildate.Date
	Deadline   civildate.NullDate
	Amount     *int64
}

// ClaimState is what the claim row mirrors from its timeline.
type ClaimState struct {
	Status   string
	Paid     *int64
	Deadline civildate.NullDate
}

// StateOf derives the claim state: the latest event (by day, then id) gives the status, its deadline when documents
// are requested, and the latest paid event's amount while the claim is paid.
func StateOf(events []EventFact) ClaimState {
	evs := append([]EventFact(nil), events...)
	sort.SliceStable(evs, func(i, j int) bool {
		if c := evs[i].OccurredOn.Compare(evs[j].OccurredOn); c != 0 {
			return c < 0
		}
		return evs[i].ID < evs[j].ID
	})
	st := ClaimState{Status: ClaimSubmitted}
	if len(evs) == 0 {
		return st
	}
	last := evs[len(evs)-1]
	st.Status = last.Status
	if last.Status == ClaimDocsRequested {
		st.Deadline = last.Deadline
	}
	if last.Status == ClaimPaid {
		st.Paid = last.Amount
	}
	return st
}

// Open reports whether a claim is still in progress.
func Open(status string) bool {
	return status == ClaimSubmitted || status == ClaimReviewing || status == ClaimDocsRequested || status == ClaimAppealed
}

// MatchesFilter applies the list filter chip.
func MatchesFilter(status, filter string) bool {
	switch filter {
	case FilterOpen:
		return Open(status)
	case FilterNeedsAction:
		return status == ClaimDocsRequested
	case FilterPaid:
		return status == ClaimPaid
	case FilterRejected:
		return status == ClaimRejected
	}
	return true
}

// PolicyFact is what the history rules need of one policy.
type PolicyFact struct {
	ID     uint64
	Period Period
}

// newer orders the history: the later start first (an open start is the oldest), then the higher id.
func newer(a, b PolicyFact) bool {
	switch {
	case a.Period.From.Valid != b.Period.From.Valid:
		return a.Period.From.Valid
	case a.Period.From.Valid && a.Period.From.Date.Compare(b.Period.From.Date) != 0:
		return a.Period.From.Date.After(b.Period.From.Date)
	}
	return a.ID > b.ID
}

func sortedHistory(ps []PolicyFact) []PolicyFact {
	out := append([]PolicyFact(nil), ps...)
	sort.SliceStable(out, func(i, j int) bool { return newer(out[i], out[j]) })
	return out
}

// CurrentOf is the id of her single current policy on `today`: the latest period containing today, else the newest
// policy (0 when she has none).
func CurrentOf(ps []PolicyFact, today civildate.Date) uint64 {
	sorted := sortedHistory(ps)
	for _, p := range sorted {
		if p.Period.Contains(today) {
			return p.ID
		}
	}
	return NewestOf(sorted)
}

// NewestOf is the id of the newest policy — the only one that can be renewed (0 when none).
func NewestOf(ps []PolicyFact) uint64 {
	sorted := sortedHistory(ps)
	if len(sorted) == 0 {
		return 0
	}
	return sorted[0].ID
}

// StatusOf is a policy's place in the history: current, upcoming (starts after today, e.g. renewed early) or past.
// A past policy and its claims are read-only.
func StatusOf(p PolicyFact, currentID uint64, today civildate.Date) string {
	switch {
	case p.ID == currentID:
		return PolicyCurrent
	case p.Period.From.Valid && p.Period.From.Date.After(today):
		return PolicyUpcoming
	}
	return PolicyPast
}

// RenewalPeriod is the default period of a renewal: from the day after the old period ends (today when it has no
// end) for the same length (one year when the old period is open).
func RenewalPeriod(old Period, today civildate.Date) Period {
	from := today
	if old.To.Valid {
		from = old.To.Date.AddDays(1)
	}
	to := civildate.New(from.Year+1, from.Month, from.Day).AddDays(-1)
	if old.From.Valid && old.To.Valid {
		to = from.AddDays(old.From.Date.DiffDays(old.To.Date))
	}
	return Period{From: civildate.NullDate{Date: from, Valid: true}, To: civildate.NullDate{Date: to, Valid: true}}
}

// MatchesQuery is the list search over the claim's own visible text.
func MatchesQuery(q string, fields ...string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}
