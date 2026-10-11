// Package insurance is the insurance self-tracking domain (canvas CB-INS-01, roadmap DECISIONS #12): the user keeps her
// own supplementary health insurance — one policy (insurer from the admin catalog or typed, masked number, validity,
// members = self + spouse + her bloom children, claim deadline, coverage caps / deductibles / waiting periods per
// category), the claims she files with the insurer herself (member, cost type, centre, day, amount, record documents,
// a status timeline she advances, expected payable) and health questionnaire drafts prefilled from her record, which
// she confirms and exports as a PDF drawn on the device. The contracted centres list is admin data.
//
// Privacy. No insurer API: nothing is ever sent anywhere. The policy number and the payout IBAN are kept only as their
// last four characters. Free-text notes (claim note, timeline notes, questionnaire explanation) are sealed at rest
// (PRIVATE_NOTE_KEY, bound to the owner). Every query is scoped by the caller's user id; another user's row answers
// the same 404 as a missing one. Nothing here is logged.
//
// The read-only «how it works» page stays bloom B-N10-06 (info_sections group `insurance`, internal/content).
package insurance

import (
	"embed"
	"errors"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Claim cost types («نوع هزینه»).
const (
	CostParaclinic  = "paraclinic"
	CostVisit       = "visit"
	CostDrug        = "drug"
	CostInpatient   = "inpatient"
	CostDelivery    = "delivery"
	CostDental      = "dental"
	CostOptical     = "optical"
	CostInfertility = "infertility"
	CostOther       = "other"
)

// Coverage categories («سقف‌ها و مصرف»).
const (
	CatInpatient   = "inpatient"
	CatDelivery    = "delivery"
	CatParaclinic  = "paraclinic"
	CatVisitDrug   = "visit_drug"
	CatInfertility = "infertility"
	CatDental      = "dental"
	CatOptical     = "optical"
	CatOther       = "other"
)

// Coverage statuses.
const (
	CoverageCovered     = "covered"
	CoverageConditional = "conditional" // «اگر در قرارداد باشد»
	CoverageNotCovered  = "not_covered"
)

// Member kinds.
const (
	MemberSelf   = "self"
	MemberSpouse = "spouse"
	MemberChild  = "child"
	MemberOther  = "other"
)

// Claim statuses (the latest timeline event).
const (
	ClaimSubmitted     = "submitted"
	ClaimReviewing     = "reviewing"
	ClaimDocsRequested = "docs_requested"
	ClaimPaid          = "paid"
	ClaimRejected      = "rejected"
	ClaimAppealed      = "appealed"
)

// Claim list filters.
const (
	FilterOpen        = "open"
	FilterNeedsAction = "needs_action"
	FilterPaid        = "paid"
	FilterRejected    = "rejected"
)

// Questionnaire purposes and statuses.
const (
	PurposeIssue   = "issue"
	PurposeRenewal = "renewal"

	QDraft        = "draft"
	QConfirmed    = "confirmed"
	QSubmitted    = "submitted" // she sent it to the insurer herself
	QReviewing    = "reviewing"
	QDoctorReview = "doctor_review"
	QApproved     = "approved"
	QConditional  = "conditional"
	QRejected     = "rejected"
)

// Questionnaire answers.
const (
	AnswerYes = "yes"
	AnswerNo  = "no"
)

// Limits.
const (
	MaxMembers        = 10
	MaxCoverages      = 12
	MaxNote           = 1000
	MaxExplanation    = 2000
	MaxQuestionnaires = 20
	MaxAmount         = 1_000_000_000_000 // toman
	MaxDeadlineDays   = 365
	MaxWaitingDays    = 1095
	MaxDocsPerClaim   = 20
	MaxEventsPerClaim = 30
)

// Accepted values.
var (
	CostTypes          = []string{CostParaclinic, CostVisit, CostDrug, CostInpatient, CostDelivery, CostDental, CostOptical, CostInfertility, CostOther}
	Categories         = []string{CatInpatient, CatDelivery, CatParaclinic, CatVisitDrug, CatInfertility, CatDental, CatOptical, CatOther}
	CoverageStatuses   = []string{CoverageCovered, CoverageConditional, CoverageNotCovered}
	MemberKinds        = []string{MemberSelf, MemberSpouse, MemberChild, MemberOther}
	ClaimStatuses      = []string{ClaimSubmitted, ClaimReviewing, ClaimDocsRequested, ClaimPaid, ClaimRejected, ClaimAppealed}
	ClaimFilters       = []string{FilterOpen, FilterNeedsAction, FilterPaid, FilterRejected}
	DocumentLabels     = []string{"prescription", "invoice", "report", "other"}
	Purposes           = []string{PurposeIssue, PurposeRenewal}
	QuestionnaireSteps = []string{QSubmitted, QReviewing, QDoctorReview, QApproved, QConditional, QRejected}
	CentreTypes        = []string{"gyn", "lab", "imaging", "hospital", "pharmacy", "pediatric", "other"}
)

// Error codes (the `error_code` extra of a failed response).
const (
	CodeNotFound         = "insurance_not_found"
	CodeNoPolicy         = "insurance_no_policy"
	CodeStorage          = "insurance_storage_unavailable"
	CodeLocked           = "insurance_questionnaire_locked"
	CodeIncomplete       = "insurance_questionnaire_incomplete"
	CodeNotConfirmed     = "insurance_questionnaire_not_confirmed"
	CodeTooMany          = "insurance_too_many"
	CodeInUse            = "insurance_in_use"
	CodeEventLast        = "insurance_event_last"
	CodeValidationFailed = "insurance_validation_failed"
)

// Errors of the service.
var (
	ErrNotFound     = errors.New("insurance: not found")
	ErrNoPolicy     = errors.New("insurance: no policy")
	ErrStorage      = errors.New("insurance: sealed storage unavailable")
	ErrLocked       = errors.New("insurance: questionnaire already sent")
	ErrIncomplete   = errors.New("insurance: questionnaire has unanswered questions")
	ErrNotConfirmed = errors.New("insurance: questionnaire not confirmed")
	ErrTooMany      = errors.New("insurance: too many items")
	ErrEventLast    = errors.New("insurance: the first event cannot be removed")
)

// Controller messages, validation lines and attribute names are data: lang/<code>/insurance.json (English fallback).
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the insurance line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("insurance."+key, nil, locale) }

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	v, ok := translator().Get("insurance.attributes", locale)
	m, isMap := v.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if s, ok := m.Get(k); ok {
			if str, ok := s.(string); ok {
				kv = append(kv, k, str)
			}
		}
	}
	return kv
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}
