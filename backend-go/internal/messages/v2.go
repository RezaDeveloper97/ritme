package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
	pregv2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Smart messages v2 (bloom B-N9-05, D-86): the rule engine's user endpoints. GET /messages/selected answers the one
// message for a placement; POST /messages/events records impression / click / dismiss / goal (no health payload: rule
// id, variant and event only).

// BuildFacts computes every fact of one user (the admin rule preview). Under CYCLE_ENGINE=shadow|v6 the facts come
// from the fact registry (T-M10-03, D-161: the v6 engine and the modules, lazy producers); CYCLE_ENGINE=v1 keeps the
// legacy builder (mode via the message manager, the legacy cycle engine).
func BuildFacts(ctx context.Context, h *Handlers, userID uint64, today civildate.Date) (rules.Facts, error) {
	src := NewStoreSource(h.q, h.pq, userID, today).WithEngine(h.states).WithNow(h.clock.Now()).WithSegments(h.segs)
	return buildFacts(ctx, h, src, userID, today, "", nil)
}

// buildFacts builds the facts of src's user: under v6 the registry facts of needed (nil = all) for a request in locale
// ("" = unknown); under v1 the legacy facts (needed and locale unused).
func buildFacts(ctx context.Context, h *Handlers, src *StoreSource, userID uint64, today civildate.Date, locale string,
	needed []string,
) (rules.Facts, error) {
	if h.states != nil {
		fs := facts.New(src.FactSources(), today, locale)
		var err error
		if needed == nil {
			err = fs.All(ctx)
		} else {
			err = fs.Need(ctx, needed...)
		}
		if err != nil {
			return nil, err
		}
		return rules.Facts(fs.Values()), nil
	}
	f := rules.Facts{"is_premium": false}
	mgr := manager.New(WithPostpartum(src, h.pp), content.NewRepository(h.q), "fa", today)
	mode, err := mgr.DetectMode(ctx)
	if err != nil {
		return nil, err
	}
	f["mode"] = string(mode)
	if p, err := src.Profile(ctx); err != nil {
		return nil, err
	} else if p != nil {
		f["user_goal"] = p.UserGoal
		f["is_premium"] = p.SubscriptionType == "premium"
	}
	if lm, err := src.LifeMode(ctx); err == nil && lm != "" {
		f["life_mode"] = lm
	}
	if name, err := h.q.GetMessageUserName(ctx, userID); err == nil {
		if first, _, _ := strings.Cut(strings.TrimSpace(name.String), " "); first != "" {
			f["name"] = first
		}
	}
	if string(mode) == "cycle" {
		if c, err := src.Cycle(ctx, today); err == nil && c.Complete {
			f["cycle_day"] = float64(c.Day)
			f["cycle_length"] = float64(c.CycleLength)
			f["phase"] = string(c.Phase)
			f["subphase"] = string(c.CurrentSubphase)
			f["days_to_period"] = float64(max(c.CycleLength-c.Day+1, 0))
			f["is_pms_window"] = c.IsPmsWindow
			f["is_fertile_window"] = c.IsFertileWindow
		}
	}
	return withPregnancy(ctx, f, mode, src, today), nil
}

// withPregnancy adds the gestational week in pregnancy mode.
func withPregnancy(ctx context.Context, f rules.Facts, mode enums.MessageMode, src *StoreSource, today civildate.Date) rules.Facts {
	if mode == enums.MessageModePregnancy {
		if p, err := src.PregnancyProfile(ctx); err == nil {
			if week, ok := pregnancyWeek(p, today); ok {
				f["pregnancy_week"] = week
			}
		}
	}
	return f
}

// pregnancyWeek is the 1-based gestational week of p on today, read from the pregnancy v2 dating the pregnancy screens
// use (LMP, ultrasound or manual entry), so a message fact and the pregnancy UI can never disagree. ok is false when the
// profile has no dating data.
func pregnancyWeek(p *pstore.PregnancyProfile, today civildate.Date) (float64, bool) {
	d, ok := pregv2.Resolve(p, today)
	if !ok {
		return 0, false
	}
	return float64(d.CurrentWeek()), true
}

// feedOf builds the feed of slots for the request's user (T-M10-06): one compiled snapshot, one fact prefetch over the
// union of the slots, one history read; the server-side events go to the buffered writer.
func (h *Handlers) feedOf(c fiber.Ctx, slots []string, alerts bool) (*feed.Result, string, error) {
	v6 := h.states != nil
	_, src, locale, today, err := h.requestOf(c, v6)
	if err != nil {
		return nil, "", err
	}
	uid, _ := auth.CurrentUserID(c)
	clk := clock.FromContext(c, h.clock)
	now := clk.Now()
	if v6 { // v6: selection reads the user-local clock (M-08, D-126)
		now = civildate.LocalNow(c, clk)
	}
	src.WithNow(now)
	req := feed.Request{
		UserID: uid, Slots: slots, Locale: locale, Now: now, Alerts: alerts,
		Facts: func(ctx context.Context, need []string) (rules.Facts, error) {
			return buildFacts(ctx, h, src, uid, today, locale, need)
		},
	}
	if v6 {
		// The companion check and the inbox counts off the fact profile row (memoised: the account facts reuse it),
		// so a warm feed is the profile, the engine-state revisions and the history: three queries plus auth.
		if err := src.companionConflict(c.Context(), i18n.Locale(c)); err != nil {
			return nil, "", err
		}
		req.Inbox = src.inbox
		req.ChallengeRuns = src.challengeRuns // T-M10-10: runs off the same profile row (no extra query)
		req.OpenRows = src.openRows           // T-M9-49b: the open inbox rows off the same row
	}
	res, err := h.feed.Build(c.Context(), req)
	return res, locale, err
}

// companionConflict is companion.Accounts.IsCompanion off the fact profile row: a male account gets the 409 (bloom
// B-N4-03). No row or no gender = a woman's account.
func (s *StoreSource) companionConflict(ctx context.Context, locale string) error {
	row, err := s.factProfile(ctx)
	if err != nil {
		return err
	}
	if row != nil && row.Gender.Valid && row.Gender.String == string(enums.GenderMale) {
		return companion.AccountConflict(locale)
	}
	return nil
}

// inbox is the open inbox counts off the fact profile row (feed.Request.Inbox).
func (s *StoreSource) inbox(ctx context.Context) (feed.Alerts, bool, error) {
	row, err := s.factProfile(ctx)
	if err != nil || row == nil {
		return feed.Alerts{}, false, err
	}
	return feed.Alerts{Open: row.InboxOpen, Urgent: row.InboxUrgent}, true, nil
}

// openRows is the open inbox rows off the fact profile row (feed.Request.OpenRows, T-M9-49b).
func (s *StoreSource) openRows(ctx context.Context) ([]store.ListUserOpenInboxRow, bool, error) {
	row, err := s.factProfile(ctx)
	if err != nil || row == nil {
		return nil, false, err
	}
	rows, ok := parseInboxRows(row.InboxRows)
	return rows, ok, nil
}

// challengeRuns is the user's challenge runs off the fact profile row (feed.Request.ChallengeRuns, T-M10-10).
func (s *StoreSource) challengeRuns(ctx context.Context) ([]feed.ChallengeRun, bool, error) {
	row, err := s.factProfile(ctx)
	if err != nil || row == nil {
		return nil, false, err
	}
	return feed.ParseChallengeRuns(row.ChallengeRuns), true, nil
}

// Feed is GET /messages/feed?slots=a,b → data {slots: {<slot>: [item…]}, alerts: {open, urgent}, rules_rev}
// (Go-only, D-163). Without ?slots= every screen slot (feed.DefaultSlots) is answered.
func (h *Handlers) Feed(c fiber.Ctx) error {
	slots := feed.DefaultSlots
	if raw := strings.TrimSpace(c.Query("slots")); raw != "" {
		slots = nil
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" && !slices.Contains(slots, p) {
				slots = append(slots, p)
			}
		}
		if len(slots) == 0 || slices.ContainsFunc(slots, func(s string) bool { return !feed.ValidSlot(s) }) {
			locale := i18n.ResolveLocale(c, "")
			ve := httpx.NewValidationError()
			ve.Add("slots", pick(locale, "جایگاه نامعتبر است", "The slots are invalid."))
			errs, _ := ve.Body().Get("errors")
			return httpx.Fail(fiber.StatusUnprocessableEntity, ve.Summary(), "errors", errs)
		}
	}
	res, locale, err := h.feedOf(c, slots, true)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return httpx.OK(c, res.JSON(locale))
}

// Selected is GET /messages/selected?placement= → data {placement, message|null}: the feed of one slot (T-M10-06),
// in the pre-feed shape.
func (h *Handlers) Selected(c fiber.Ctx) error {
	placement := c.Query("placement")
	if !slices.Contains(rules.Placements, placement) {
		if _, ok := auth.CurrentUserID(c); !ok {
			return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
		}
		locale := i18n.ResolveLocale(c, "")
		ve := httpx.NewValidationError()
		ve.Add("placement", pick(locale, "جایگاه نامعتبر است", "The placement is invalid."))
		errs, _ := ve.Body().Get("errors")
		return httpx.Fail(fiber.StatusUnprocessableEntity, ve.Summary(), "errors", errs)
	}
	res, locale, err := h.feedOf(c, []string{placement}, false)
	if err != nil {
		return err
	}
	var msg any
	if s := res.Slots[0]; len(s.Choices) > 0 {
		msg = MessageJSON(&s.Choices[0], s.Texts[0], res.Evidence(s.Choices[0].Rule, locale))
	}
	return httpx.OK(c, jsonx.Obj("placement", placement, "message", msg))
}

// MessageJSON is the /messages/selected shape of a choice with its rendered text v and evidence ev. Level fields
// «seen / confidence / contact» only appear when the level needs them.
func MessageJSON(ch *rules.Choice, v rules.Variant, ev rules.Content) *jsonx.OrderedMap {
	r := ch.Rule
	out := jsonx.Obj(
		"rule_id", r.ID, "variant", ch.Variant, "level", r.Level, "kind", r.Kind, "placement", r.Placement,
		"title", v.Title, "body", v.Body,
		"dismissible", r.Level != rules.LevelFollowSoon,
	)
	if v.CTALabel != "" && v.CTATarget != "" {
		out.Set("cta", jsonx.Obj("label", v.CTALabel, "target", v.CTATarget))
	} else {
		out.Set("cta", nil)
	}
	if r.Level == rules.LevelWorthFollowUp || r.Level == rules.LevelFollowSoon {
		out.Set("seen", ev.Seen)
		out.Set("confidence", ev.Confidence)
	}
	if r.Level == rules.LevelFollowSoon {
		out.Set("contact", ev.Contact)
	}
	return out
}

// eventBody is POST /messages/events.
type eventBody struct {
	RuleID  uint64  `json:"rule_id"`
	Variant string  `json:"variant"`
	Event   string  `json:"event"`
	Version any     `json:"version"`
	Slot    *string `json:"slot"`
}

// Event is POST /messages/events {rule_id, event, variant?, version?, slot?} → 201 {recorded}. Spec names (shown,
// clicked, dismissed, goal) and the pre-v6 ones (impression, click, dismiss) are accepted (D-163); a client
// impression is ignored (recorded false) when the server already logged that message's `shown` today (M-19). The
// rule must exist and be served; version and slot, when sent, must belong to it. The row holds ids and the event
// name only and goes through the buffered writer.
func (h *Handlers) Event(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.ResolveLocale(c, "")
	var body eventBody
	ve := httpx.NewValidationError()
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		ve.Add("rule_id", pick(locale, "درخواست نامعتبر است", "The request body is invalid."))
	}
	if body.RuleID == 0 {
		ve.Add("rule_id", pick(locale, "شناسه قاعده الزامی است", "The rule id is required."))
	}
	stored, known := EventAliases[body.Event]
	if !known {
		ve.Add("event", pick(locale, "رویداد نامعتبر است", "The event is invalid."))
	}
	if body.Variant == "" {
		body.Variant = "a"
	}
	if len(body.Variant) > 16 {
		ve.Add("variant", pick(locale, "نسخه نامعتبر است", "The variant is invalid."))
	}
	version, vok := eventVersion(body.Version)
	if !vok {
		ve.Add("version", pick(locale, "شماره نسخه نامعتبر است", "The version is invalid."))
	}
	slot := ""
	if body.Slot != nil {
		slot = *body.Slot
		if s, ok := rules.Slots[slot]; !ok || s.Reserved {
			ve.Add("slot", pick(locale, "جایگاه نامعتبر است", "The slot is invalid."))
		}
	}
	if !ve.Empty() {
		return failValidation(ve)
	}
	row, err := h.q.GetMessageRule(c.Context(), body.RuleID)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.Fail(fiber.StatusNotFound, pick(locale, "پیام پیدا نشد", "Message not found."))
	}
	if err != nil {
		return err
	}
	r := rules.FromRow(&row)
	if !r.Served() {
		return httpx.Fail(fiber.StatusNotFound, pick(locale, "پیام پیدا نشد", "Message not found."))
	}
	if !slices.ContainsFunc(r.EffectiveVariants(), func(v rules.VariantWeight) bool { return v.Key == body.Variant }) {
		ve.Add("variant", pick(locale, "نسخه نامعتبر است", "The variant is invalid."))
	}
	if version > r.Version {
		ve.Add("version", pick(locale, "شماره نسخه نامعتبر است", "The version is invalid."))
	}
	if slot != "" && !rules.CandidateOf(slot, r.Placement) {
		ve.Add("slot", pick(locale, "جایگاه نامعتبر است", "The slot is invalid."))
	}
	if !ve.Empty() {
		return failValidation(ve)
	}
	if version == 0 {
		version = feed.ServedVersion(&r)
	}
	if slot == "" {
		slot = r.Placement
	}
	clk := clock.FromContext(c, h.clock)
	now := clk.Now()
	if stored == events.Shown {
		first, err := h.firstShown(c, uid, r.ID, slot, civildate.LocalNow(c, clk))
		if err != nil {
			return err
		}
		if !first {
			return httpx.Created(c, jsonx.Obj("recorded", false))
		}
	}
	ev := events.Event{RuleID: r.ID, UserID: uid, Version: version, Variant: body.Variant, Slot: slot, Event: stored, At: now}
	if h.writer != nil {
		h.writer.Write(c.Context(), ev)
	} else if err := h.q.InsertMessageEventsBatch(c.Context(), store.InsertMessageEventsBatchParams{
		Batch: events.Encode([]events.Event{ev}),
	}); err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("recorded", true))
}

// EventAliases maps every accepted event name onto its stored name (M-01: the pre-v6 spelling is stored).
var EventAliases = map[string]string{
	"impression": events.Shown, "shown": events.Shown,
	"click": events.Clicked, "clicked": events.Clicked,
	"dismiss": events.Dismissed, "dismissed": events.Dismissed,
	"goal": events.GoalEvent,
}

// eventVersion reads an optional version: absent → 0; otherwise a positive integer.
func eventVersion(v any) (int, bool) {
	switch n := v.(type) {
	case nil:
		return 0, true
	case float64:
		if n >= 1 && n == float64(int(n)) && n < 1<<31 {
			return int(n), true
		}
	}
	return 0, false
}

func failValidation(ve *httpx.ValidationError) error {
	errs, _ := ve.Body().Get("errors")
	return httpx.Fail(fiber.StatusUnprocessableEntity, ve.Summary(), "errors", errs)
}

// firstShown sets the server's shown mark of (user, rule, slot, local day) and reports whether it was not set yet;
// without Redis it reads the table (a row of that day means it was shown).
func (h *Handlers) firstShown(c fiber.Ctx, uid, ruleID uint64, slot string, localNow time.Time) (bool, error) {
	day := civildate.FromTime(localNow)
	first, err := h.marks.First(c.Context(), events.MarkTTL, events.DayKey("shown", uid, ruleID, slot, day.String()))
	if err == nil {
		return first[0], nil
	}
	n, err := h.q.CountUserMessageEvents(c.Context(), store.CountUserMessageEventsParams{
		UserID: uid, RuleID: ruleID, Event: events.Shown,
		CreatedAt: sql.NullTime{Time: day.Midnight(localNow.Location()), Valid: true},
	})
	if err != nil {
		return false, err
	}
	return n == 0, nil
}
