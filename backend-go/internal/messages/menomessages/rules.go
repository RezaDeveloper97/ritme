// Package menomessages is the menopause part of the message engine (CB-MENO-12, canvas Main / nbl_Meno_Home): the
// alerts and tips of a user in menopause mode, computed per request from the menopause API's own rules
// (internal/menopause.Service.Signals) — nothing re-implemented, nothing stored.
//
// Rules, in display order (thresholds are named constants below, all [needs clinical review]):
//
//   - postmenopausal_bleeding (alert, high): the home's bleeding flag — bleeding or spotting logged in the last
//     menopause.BleedingAlertDays days while the stage is meno or post (docs/canvas-build/menopause.md §2);
//   - checkup_overdue (reminder, medium): a menopause checkup of her M4 plan is overdue;
//   - score_worsened (alert, medium): her newest monthly score (this or the previous Jalali month) is ScoreWorseBy+
//     points above the questionnaire before it;
//   - hrt_review (reminder, medium): an active HRT item's review_on is today or within HRTReviewDays days;
//   - checkup_due (reminder, low): no checkup is overdue but one is due (never done counts as due, like the plan);
//   - stage tips (tip, low): the active catalog meno_tips placed on the home for her stage.
//
// Only users whose effective life mode is menopause get messages; every other user gets none.
//
// Texts: the bleeding alert and the tips are the admin-editable catalog items (meno_alerts postmenopausal_bleeding,
// meno_tips) the menopause screens already show; the other rules are message_contents rows of group
// `menopause_message` (item = rule key, payload {title, body, action}) with a per-field fallback to the embedded copy
// in lang/<code>/menopause_messages.json (also the bleeding alert's fallback when its catalog item is inactive).
// `needs_review` is true while an embedded text is used or the catalog item is flagged for review.
package menomessages

import (
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Group is the message_contents group of the rule texts.
const Group = "menopause_message"

// Rule keys (= message_contents item keys for the TextRules).
const (
	RuleBleeding       = menopause.AlertPostmenopausalBleeding
	RuleCheckupOverdue = "checkup_overdue"
	RuleScoreWorsened  = "score_worsened"
	RuleHRTReview      = "hrt_review"
	RuleCheckupDue     = "checkup_due"
)

// Rules are every rule, in display order (tips come after them).
var Rules = []string{RuleBleeding, RuleCheckupOverdue, RuleScoreWorsened, RuleHRTReview, RuleCheckupDue}

// TextRules are the rules whose texts are message_contents rows (the admin registry group's items).
var TextRules = []string{RuleCheckupOverdue, RuleScoreWorsened, RuleHRTReview, RuleCheckupDue}

// TextKeys are the texts of a message (= the payload keys of its message_contents rows).
var TextKeys = []string{"title", "body", "action"}

// Kinds and priorities.
const (
	KindAlert    = "alert"
	KindReminder = "reminder"
	KindTip      = "tip"

	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Thresholds [needs clinical review].
const (
	// ScoreWorseBy is the rise of the monthly total (out of 44) over the previous questionnaire that raises
	// score_worsened.
	ScoreWorseBy = 4
	// ScoreFreshMonths: only a questionnaire of the current Jalali month or the one before counts (an old rise is
	// stale news).
	ScoreFreshMonths = 2
	// HRTReviewDays: review_on from today up to this many days ahead is "near".
	HRTReviewDays = 14
)

// Links of the menopause screens (frontend CB-MENO-08..10) and the checkups screen.
const (
	LinkBleeding  = "/menopause/alert"
	LinkScore     = "/menopause/score"
	LinkTreatment = "/menopause/treatment"
	LinkCheckups  = "/checkups"
)

// Hit is one raised message before its texts are resolved.
type Hit struct {
	Rule, Kind, Priority string
	Link                 string // "" = none

	BleedingLastOn *civildate.Date       // postmenopausal_bleeding
	Checkup        *engine.Item          // checkup_overdue / checkup_due: the first one of that status
	CheckupCount   int                   // checkup_*: how many checkups have that status
	Score          *menopause.ScoreEntry // score_worsened
	Treatment      *store.TreatmentItem  // hrt_review
	DaysLeft       int                   // hrt_review: days from today to review_on
	Item           *catalog.Item         // the catalog item of the texts (bleeding alert, tip); nil = no item
}

// Detect runs the rules on the signals, in display order.
func Detect(s menopause.Signals) []Hit {
	if s.Mode != enums.LifeModeMenopause {
		return nil
	}
	var hits []Hit
	if s.Bleeding.Alert {
		hits = append(hits, Hit{
			Rule: RuleBleeding, Kind: KindAlert, Priority: PriorityHigh, Link: LinkBleeding,
			BleedingLastOn: s.Bleeding.LastOn, Item: s.Bleeding.Item,
		})
	}
	overdue, overdueN := firstWithStatus(s.Checkups, engine.StatusOverdue)
	if overdue != nil {
		hits = append(hits, Hit{
			Rule: RuleCheckupOverdue, Kind: KindReminder, Priority: PriorityMedium, Link: checkupLink(overdue),
			Checkup: overdue, CheckupCount: overdueN,
		})
	}
	if sc := s.Score; scoreWorsened(sc, s.Date) {
		hits = append(hits, Hit{Rule: RuleScoreWorsened, Kind: KindAlert, Priority: PriorityMedium, Link: LinkScore, Score: sc})
	}
	for i, it := range s.HRT {
		if !it.ReviewOn.Valid {
			continue
		}
		if days := s.Date.DiffDays(it.ReviewOn.Date); days >= 0 && days <= HRTReviewDays {
			hits = append(hits, Hit{
				Rule: RuleHRTReview, Kind: KindReminder, Priority: PriorityMedium, Link: LinkTreatment,
				Treatment: &s.HRT[i], DaysLeft: days,
			})
		}
	}
	if overdue == nil {
		if due, n := firstWithStatus(s.Checkups, engine.StatusDue); due != nil {
			hits = append(hits, Hit{
				Rule: RuleCheckupDue, Kind: KindReminder, Priority: PriorityLow, Link: checkupLink(due),
				Checkup: due, CheckupCount: n,
			})
		}
	}
	for i := range s.Tips {
		hits = append(hits, Hit{Rule: s.Tips[i].Code, Kind: KindTip, Priority: PriorityLow, Item: &s.Tips[i]})
	}
	return hits
}

// scoreWorsened: a fresh questionnaire ScoreWorseBy+ points above the previous one.
func scoreWorsened(sc *menopause.ScoreEntry, today civildate.Date) bool {
	if sc == nil {
		return false
	}
	d := sc.Delta()
	if d == nil || *d < ScoreWorseBy {
		return false
	}
	oldest := menopause.MonthStart(today)
	for range ScoreFreshMonths - 1 {
		oldest = menopause.PrevMonth(oldest)
	}
	return !sc.Month.Before(oldest)
}

// firstWithStatus is the first item of status (the list is sorted overdue first, then by due date) and the count.
func firstWithStatus(items []engine.Item, st engine.Status) (*engine.Item, int) {
	var first *engine.Item
	n := 0
	for i := range items {
		if items[i].Status != st {
			continue
		}
		if first == nil {
			first = &items[i]
		}
		n++
	}
	return first, n
}

func checkupLink(it *engine.Item) string {
	if it == nil || it.TypeID == 0 {
		return LinkCheckups
	}
	return LinkCheckups + "/" + uitoa(it.TypeID)
}
