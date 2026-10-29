package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/facts"
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

// EventKinds are the accepted event names.
var EventKinds = []string{"impression", "click", "dismiss", "goal"}

// BuildFacts computes every fact of one user (the admin rule preview). Under CYCLE_ENGINE=shadow|v6 the facts come
// from the fact registry (T-M10-03, D-161: the v6 engine and the modules, lazy producers); CYCLE_ENGINE=v1 keeps the
// legacy builder (mode via the message manager, the legacy cycle engine).
func BuildFacts(ctx context.Context, h *Handlers, userID uint64, today civildate.Date) (rules.Facts, error) {
	src := NewStoreSource(h.q, h.pq, userID, today).WithEngine(h.states).WithNow(h.clock.Now())
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

// neededFacts are the fact names the rules read: condition and exception facts and the {variables} of every text.
func neededFacts(rs []rules.Rule) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	for i := range rs {
		for _, g := range rs[i].Groups {
			for _, c := range g.Items {
				add(c.Fact)
			}
		}
		for _, c := range rs[i].Exceptions {
			add(c.Fact)
		}
		for _, p := range rules.Placeholders(rs[i].Content.Texts()...) {
			add(p)
		}
	}
	return out
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

// Selected is GET /messages/selected?placement= → data {placement, message|null}.
func (h *Handlers) Selected(c fiber.Ctx) error {
	_, src, locale, today, err := h.request(c)
	if err != nil {
		return err
	}
	placement := c.Query("placement")
	if !slices.Contains(rules.Placements, placement) {
		ve := httpx.NewValidationError()
		ve.Add("placement", pick(locale, "جایگاه نامعتبر است", "The placement is invalid."))
		errs, _ := ve.Body().Get("errors")
		return httpx.Fail(fiber.StatusUnprocessableEntity, ve.Summary(), "errors", errs)
	}
	uid, _ := auth.CurrentUserID(c)
	now := clock.FromContext(c, h.clock).Now()
	ctx := c.Context()
	rows, err := rules.ActiveRows(ctx, h.q, placement) // process snapshot, invalidated by admin writes (T-M9-13)
	if err != nil {
		return err
	}
	all := make([]rules.Rule, 0, len(rows))
	for i := range rows {
		all = append(all, rules.FromRow(&rows[i]))
	}
	src.WithNow(now)
	facts, err := buildFacts(ctx, h, src, uid, today, locale, neededFacts(all))
	if err != nil {
		return err
	}
	// A rule whose text needs a variable the user has no value for is skipped (never a raw placeholder).
	renderable := all[:0]
	for i := range all {
		if _, ok := rules.RenderVariant(all[i].VariantContent(all[i].Assign(uid)), facts); ok {
			renderable = append(renderable, all[i])
		}
	}
	ch, err := rules.Select(renderable, facts, uid, now, rules.DBHistory{Ctx: ctx, Q: h.q, UserID: uid})
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("placement", placement, "message", MessageJSON(ch, facts)))
}

// MessageJSON is the user-facing shape of a choice (nil → null). Level fields «seen / confidence / contact» only
// appear when the level needs them.
func MessageJSON(ch *rules.Choice, facts rules.Facts) any {
	if ch == nil {
		return nil
	}
	r := ch.Rule
	v, _ := rules.RenderVariant(r.VariantContent(ch.Variant), facts)
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
		out.Set("seen", r.Content.Seen)
		out.Set("confidence", r.Content.Confidence)
	}
	if r.Level == rules.LevelFollowSoon {
		out.Set("contact", r.Content.Contact)
	}
	return out
}

// Event is POST /messages/events {rule_id, variant?, event} → 201 {recorded: true}. The rule must exist and be
// served; the row holds ids and the event name only.
func (h *Handlers) Event(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.ResolveLocale(c, "")
	var body struct {
		RuleID  uint64 `json:"rule_id"`
		Variant string `json:"variant"`
		Event   string `json:"event"`
	}
	ve := httpx.NewValidationError()
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		ve.Add("rule_id", pick(locale, "درخواست نامعتبر است", "The request body is invalid."))
	}
	if body.RuleID == 0 {
		ve.Add("rule_id", pick(locale, "شناسه قاعده الزامی است", "The rule id is required."))
	}
	if !slices.Contains(EventKinds, body.Event) {
		ve.Add("event", pick(locale, "رویداد نامعتبر است", "The event is invalid."))
	}
	if body.Variant == "" {
		body.Variant = "a"
	}
	if body.Variant != "a" && body.Variant != "b" {
		ve.Add("variant", pick(locale, "نسخه نامعتبر است", "The variant is invalid."))
	}
	if !ve.Empty() {
		errs, _ := ve.Body().Get("errors")
		return httpx.Fail(fiber.StatusUnprocessableEntity, ve.Summary(), "errors", errs)
	}
	row, err := h.q.GetMessageRule(c.Context(), body.RuleID)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.Fail(fiber.StatusNotFound, pick(locale, "پیام پیدا نشد", "Message not found."))
	}
	if err != nil {
		return err
	}
	if r := rules.FromRow(&row); !r.Served() {
		return httpx.Fail(fiber.StatusNotFound, pick(locale, "پیام پیدا نشد", "Message not found."))
	}
	if err := h.q.InsertMessageEvent(c.Context(), store.InsertMessageEventParams{
		RuleID: body.RuleID, UserID: uid, Variant: body.Variant, Event: body.Event,
		Now: sql.NullTime{Time: clock.FromContext(c, h.clock).Now(), Valid: true},
	}); err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("recorded", true))
}
