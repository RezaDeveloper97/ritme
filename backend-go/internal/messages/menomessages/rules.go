// Package menomessages is the menopause part of the message engine (CB-MENO-12, canvas Main / nbl_Meno_Home; T-M10-09,
// D-166): the alerts and tips of a user in menopause mode. Since T-M10-09 every message is a message rule (rule_key
// meno.<name>, seeded by goose 00115): the menopause signals (internal/menopause.Service.Signals — nothing
// re-implemented) become the meno.* facts (messages/facts/menopause.go), the warnings and reminders run through the
// alerts inbox (one row per rule, merged on new evidence, resolved when the condition ends) and the stage tips are
// daily_advice rules. GET /messages/menopause is an adapter over them with the D-43 JSON shape.
//
// Rules, in display order (the thresholds are the conditions of the seeded rules, [needs clinical review]):
//
//   - postmenopausal_bleeding (alert, high): the home's bleeding flag — bleeding or spotting logged in the last
//     menopause.BleedingAlertDays days while the stage is meno or post (docs/canvas-build/menopause.md §2);
//   - checkup_overdue (reminder, medium): a menopause checkup of her M4 plan is overdue;
//   - score_worsened (alert, medium): her newest monthly score (this or the previous Jalali month) is 4+ points
//     above the questionnaire before it;
//   - hrt_review (reminder, medium, one per item): an active HRT item's review_on is today or within 14 days;
//   - checkup_due (reminder, low): no checkup is overdue but one is due (never done counts as due, like the plan);
//   - stage tips (tip, low): the active catalog meno_tips placed on the home for her stage.
//
// Only users whose effective life mode is menopause get messages; every other user gets none.
//
// Texts of the adapter are unchanged (D-43): the bleeding alert and the tips are the admin-editable catalog items
// (meno_alerts postmenopausal_bleeding, meno_tips) the menopause screens already show; the other rules are
// message_contents rows of group `menopause_message` (item = rule key, payload {title, body, action}) with a per-field
// fallback to the embedded copy in lang/<code>/menopause_messages.json (also the bleeding alert's fallback when its
// catalog item is inactive). `needs_review` is true while an embedded text is used or the catalog item is flagged for
// review. The seeded rules carry the same copy (copy/<code>.json) for the alerts inbox and the feed.
package menomessages

import (
	"slices"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Group is the message_contents group of the rule texts.
const Group = "menopause_message"

// Rule keys (= message_contents item keys for the TextRules, and the `key` of the GET answer).
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

// Kinds and priorities of the GET answer (D-43).
const (
	KindAlert    = "alert"
	KindReminder = "reminder"
	KindTip      = "tip"

	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Freshness of a questionnaire [needs clinical review].
const (
	// ScoreFreshMonths: only a questionnaire of the current Jalali month or the one before counts (an old rise is
	// stale news): older, meno.score_rise is 0.
	ScoreFreshMonths = 2
)

// Links of the menopause screens (frontend CB-MENO-08..10) and the checkups screen.
const (
	LinkBleeding  = "/menopause/alert"
	LinkScore     = "/menopause/score"
	LinkTreatment = "/menopause/treatment"
	LinkCheckups  = "/checkups"
)

// KeyPrefix is the rule_key prefix of the menopause message rules.
const KeyPrefix = "meno."

// Def is one seeded menopause message rule (goose 00115 is generated from these and the copy; its int test pins them).
type Def struct {
	Key  string // rule_key
	Msg  string // the GET answer's key (the D-43 rule key, or the catalog tip code)
	Name string // admin title
	Slot string // rules.SlotAlertsInbox or daily_advice
	// Kind / Level / Prio are the rule's (rules.Kind*, stored rules.Level*).
	Kind, Level string
	Prio        int
	// Locked: only the medical advisor edits; the code-held rule stands in when the row is off or gone.
	Locked bool
	Groups []rules.Group
	// MsgKind / MsgPriority / Link are the GET answer's kind, priority and link (D-43).
	MsgKind, MsgPriority, Link string
}

// SlotDailyAdvice is the slot of the stage tips.
const SlotDailyAdvice = "daily_advice"

func all(items ...rules.Condition) []rules.Group { return []rules.Group{{Op: "and", Items: items}} }

func c(fact, op string, v any) rules.Condition { return rules.Condition{Fact: fact, Op: op, Value: v} }

func tipDef(stage, name string, prio int) Def {
	return Def{Key: KeyPrefix + "tip_stage_" + stage, Msg: "stage_" + stage, Name: "Menopause tip: " + name,
		Slot: SlotDailyAdvice, Kind: rules.KindContent, Level: rules.LevelInfo, Prio: prio,
		Groups: all(c("meno.stage", "eq", stage)), MsgKind: KindTip, MsgPriority: PriorityLow}
}

// Defs are the menopause message rules in display order (D-43: alerts and reminders, then the stage tips). The level
// and kind keep today's behaviour: the two «alert» messages are clinical follow_up (with evidence), the reminders
// content suggestion / info, the tips content info in daily_advice.
var Defs = []Def{
	{Key: KeyPrefix + RuleBleeding, Msg: RuleBleeding, Name: "Menopause: bleeding after menopause", Slot: rules.SlotAlertsInbox,
		Kind: rules.KindClinical, Level: rules.LevelWorthFollowUp, Prio: 700, Locked: true,
		Groups:  all(c("meno.postmenopausal_bleeding", "eq", true)),
		MsgKind: KindAlert, MsgPriority: PriorityHigh, Link: LinkBleeding},
	{Key: KeyPrefix + RuleCheckupOverdue, Msg: RuleCheckupOverdue, Name: "Menopause: a checkup is overdue", Slot: rules.SlotAlertsInbox,
		Kind: rules.KindContent, Level: rules.LevelSuggestion, Prio: 400,
		Groups:  all(c("meno.checkups_overdue", "gte", 1)),
		MsgKind: KindReminder, MsgPriority: PriorityMedium, Link: LinkCheckups},
	{Key: KeyPrefix + RuleScoreWorsened, Msg: RuleScoreWorsened, Name: "Menopause: symptom score went up", Slot: rules.SlotAlertsInbox,
		Kind: rules.KindClinical, Level: rules.LevelWorthFollowUp, Prio: 500,
		Groups:  all(c("meno.score_rise", "gte", 4)),
		MsgKind: KindAlert, MsgPriority: PriorityMedium, Link: LinkScore},
	{Key: KeyPrefix + RuleHRTReview, Msg: RuleHRTReview, Name: "Menopause: treatment review coming up", Slot: rules.SlotAlertsInbox,
		Kind: rules.KindContent, Level: rules.LevelSuggestion, Prio: 350,
		Groups:  all(c("meno.hrt_review_in_days", "between", []any{0, 14})),
		MsgKind: KindReminder, MsgPriority: PriorityMedium, Link: LinkTreatment},
	{Key: KeyPrefix + RuleCheckupDue, Msg: RuleCheckupDue, Name: "Menopause: a checkup is due", Slot: rules.SlotAlertsInbox,
		Kind: rules.KindContent, Level: rules.LevelInfo, Prio: 300,
		Groups:  all(c("meno.checkups_due", "gte", 1), c("meno.checkups_overdue", "eq", 0)),
		MsgKind: KindReminder, MsgPriority: PriorityLow, Link: LinkCheckups},
	tipDef(menopause.StagePeri, "perimenopause", 150),
	tipDef(menopause.StageMeno, "menopause", 150),
	tipDef(menopause.StagePost, "postmenopause", 150),
	tipDef(menopause.StageUnsure, "stage not known", 150),
}

// DefOf is the definition of a rule key.
func DefOf(key string) (Def, bool) {
	i := slices.IndexFunc(Defs, func(d Def) bool { return d.Key == key })
	if i < 0 {
		return Def{}, false
	}
	return Defs[i], true
}

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

// Holds reports whether the rule of d holds on f for this user (the served or code-held rule; for an alerts inbox
// rule also: its inbox row is open).
type Holds func(d *Def, f rules.Facts) bool

// CodeHolds evaluates the code-held rules (Detect, the parity table).
func CodeHolds(d *Def, f rules.Facts) bool {
	r := d.Rule(0)
	return rules.MatchRule(&r, f)
}

// Detect runs the code-held rules on the signals, in display order.
func Detect(s menopause.Signals) []Hit { return DetectWith(s, CodeHolds) }

// DetectWith builds the messages of the signals whose rules hold, in display order: the facts are Input(s); an HRT
// review is matched per item (meno.hrt_review_in_days of that item), so each item near its review is one message.
func DetectWith(s menopause.Signals, holds Holds) []Hit {
	if s.Mode != enums.LifeModeMenopause {
		return nil
	}
	f := rules.Facts(facts.MenopauseValues(Input(s, nil)))
	var hits []Hit
	for i := range Defs {
		d := &Defs[i]
		h := Hit{Rule: d.Msg, Kind: d.MsgKind, Priority: d.MsgPriority, Link: d.Link}
		switch d.Msg {
		case RuleBleeding:
			if holds(d, f) {
				h.BleedingLastOn, h.Item = s.Bleeding.LastOn, s.Bleeding.Item
				hits = append(hits, h)
			}
		case RuleCheckupOverdue, RuleCheckupDue:
			st := engine.StatusOverdue
			if d.Msg == RuleCheckupDue {
				st = engine.StatusDue
			}
			if it, n := firstWithStatus(s.Checkups, st); it != nil && holds(d, f) {
				h.Link, h.Checkup, h.CheckupCount = checkupLink(it), it, n
				hits = append(hits, h)
			}
		case RuleScoreWorsened:
			if s.Score != nil && s.Score.Delta() != nil && holds(d, f) {
				h.Score = s.Score
				hits = append(hits, h)
			}
		case RuleHRTReview:
			for j, it := range s.HRT {
				days := reviewDays(it, s.Date)
				if days < 0 {
					continue
				}
				fi := clone(f)
				fi["meno.hrt_review_in_days"] = float64(days)
				if holds(d, fi) {
					h.Treatment, h.DaysLeft = &s.HRT[j], days
					hits = append(hits, h)
				}
			}
		default: // a stage tip: its catalog item (placed on the home for her stage) carries the texts
			j := slices.IndexFunc(s.Tips, func(it catalog.Item) bool { return it.Code == d.Msg })
			if j >= 0 && holds(d, f) {
				h.Item = &s.Tips[j]
				hits = append(hits, h)
			}
		}
	}
	return hits
}

func clone(f rules.Facts) rules.Facts {
	out := make(rules.Facts, len(f)+1)
	for k, v := range f {
		out[k] = v
	}
	return out
}

// reviewDays are the days from today to the item's review_on (-1 when it has none or it passed).
func reviewDays(it store.TreatmentItem, today civildate.Date) int {
	if !it.ReviewOn.Valid {
		return facts.NoHRTReview
	}
	if d := today.DiffDays(it.ReviewOn.Date); d >= 0 {
		return d
	}
	return facts.NoHRTReview
}

// Input is the meno.* fact input of the signals; title names a checkup in the evaluating language (nil = no titles).
// A user not in menopause mode gets the «nothing» input (no stage, nothing raised), so her open rows resolve.
func Input(s menopause.Signals, title func(*engine.Item) string) *facts.MenopauseInput {
	in := &facts.MenopauseInput{HRTDays: facts.NoHRTReview}
	if s.Mode != enums.LifeModeMenopause {
		return in
	}
	if in.Stage = s.Stage.Stage; in.Stage == "" {
		in.Stage = menopause.StageUnsure
	}
	in.Bleeding = s.Bleeding.Alert
	if s.Bleeding.LastOn != nil {
		in.BleedingOn = *s.Bleeding.LastOn
	}
	name := func(it *engine.Item) string {
		if it == nil || title == nil {
			return ""
		}
		return title(it)
	}
	var ov, due *engine.Item
	ov, in.Overdue = firstWithStatus(s.Checkups, engine.StatusOverdue)
	due, in.Due = firstWithStatus(s.Checkups, engine.StatusDue)
	in.OverdueTitle, in.DueTitle = name(ov), name(due)
	if sc := s.Score; sc != nil && sc.Delta() != nil && fresh(sc, s.Date) {
		in.ScoreRise, in.ScoreTotal = *sc.Delta(), sc.Total
		if sc.Previous != nil {
			in.ScorePrevious = sc.Previous.Total
		}
	}
	for _, it := range s.HRT {
		if d := reviewDays(it, s.Date); d >= 0 && (in.HRTDays < 0 || d < in.HRTDays) {
			in.HRTDays, in.HRTName, in.HRTOn = d, it.Name, it.ReviewOn.Date
		}
	}
	return in
}

// fresh: the questionnaire is of the current Jalali month or one of the ScoreFreshMonths-1 before.
func fresh(sc *menopause.ScoreEntry, today civildate.Date) bool {
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
