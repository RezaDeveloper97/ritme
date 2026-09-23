package content

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// MaxCycleDay is Challenge::MAX_CYCLE_DAY.
const MaxCycleDay = 35

func challengeJSON(ch *store.Challenge) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", ch.ID,
		"slug", httpadmin.NullString(ch.Slug),
		"title", form.Raw(ch.Title),
		"description", form.NullRaw(ch.Description),
		"cycle_day_from", form.NullInt(ch.CycleDayFrom),
		"cycle_day_to", form.NullInt(ch.CycleDayTo),
		"category", httpadmin.NullString(ch.Category),
		"is_active", ch.IsActive,
		"sort_order", ch.SortOrder,
		"created_at", httpadmin.Time(ch.CreatedAt),
		"updated_at", httpadmin.Time(ch.UpdatedAt),
	)
}

func (h *Handlers) challenges() resource[store.Challenge] {
	return resource[store.Challenge]{
		name: "Challenge", key: "challenge", get: h.q.GetChallenge, json: challengeJSON,
		id: func(ch *store.Challenge) uint64 { return ch.ID },
		toggle: func(ctx context.Context, now sql.NullTime, id uint64) error {
			return h.q.ToggleChallenge(ctx, store.ToggleChallengeParams{Now: now, ID: id})
		},
		del: h.q.DeleteChallenge,
	}
}

// ListChallenges is GET /challenges?q=&status=all|active|inactive&cycle_day=1…35
// (ChallengeController::filtered: cycle_day also matches untargeted challenges).
func (h *Handlers) ListChallenges(c fiber.Ctx) error {
	search := strings.TrimSpace(c.Query("q"))
	status := c.Query("status")
	var only *bool
	switch status {
	case "active":
		only = new(bool)
		*only = true
	case "inactive":
		only = new(bool)
	default:
		status = "all"
	}
	lo, hi := form.BoolRange(only)
	var day any // echoed filter: the day, or null
	dayArg := sql.NullInt16{Int16: 0, Valid: true}
	if n, err := strconv.Atoi(strings.TrimSpace(c.Query("cycle_day"))); err == nil && n >= 1 && n <= MaxCycleDay {
		dayArg.Int16 = int16(n) //nolint:gosec // G115: 1…35
		day = n
	}
	pattern, jsonPattern := form.Contains(search), form.ContainsJSON(search)
	total, err := h.q.CountAdminChallenges(c.Context(), store.CountAdminChallengesParams{
		Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi, Day: dayArg,
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminChallenges(c.Context(), store.ListAdminChallengesParams{
		Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi, Day: dayArg,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, challengeJSON(&rows[i]))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("q", search, "status", status, "cycle_day", day))
	return httpadmin.OK(c, page)
}

// ChallengeOptions is GET /challenges/options.
func (h *Handlers) ChallengeOptions(c fiber.Ctx) error {
	return httpadmin.OK(c, jsonx.Obj("max_cycle_day", MaxCycleDay))
}

// ShowChallenge is GET /challenges/:id.
func (h *Handlers) ShowChallenge(c fiber.Ctx) error { return h.challenges().show(c) }

func challengeRules(c fiber.Ctx) validation.Rules {
	rules := form.Translatable(c, "title", true)
	rules = append(rules, form.Translatable(c, "description", false)...)
	day := "nullable|integer|min:1|max:" + strconv.Itoa(MaxCycleDay)
	return append(rules,
		validation.F("cycle_day_from", day),
		validation.F("cycle_day_to", day),
		validation.F("category", "nullable|string|max:255"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	)
}

// checkDayRange is `cycle_day_to … gte:cycle_day_from`, including Laravel's quirk that
// a `to` without a `from` fails (gte compares against a null field).
func checkDayRange(c fiber.Ctx) form.Check {
	return func(in phpval.Map, add form.Add) error {
		to, ok := in.Get("cycle_day_to")
		if !ok || to == nil || !phpval.IsNumeric(to) {
			return nil
		}
		from, _ := in.Get("cycle_day_from")
		if from == nil || !phpval.IsNumeric(from) || phpval.ToFloat(to) < phpval.ToFloat(from) {
			add("cycle_day_to", form.Msg(c, "validation.gte.numeric", "cycle_day_to",
				"value", phpval.ToString(from)))
		}
		return nil
	}
}

// StoreChallenge is POST /challenges.
func (h *Handlers) StoreChallenge(c fiber.Ctx) error {
	data, err := form.Validate(c, challengeRules(c), checkDayRange(c))
	if err != nil {
		return err
	}
	res, err := h.q.CreateChallenge(c.Context(), store.CreateChallengeParams{
		Title: form.ReqJSON(data, "title"), Description: form.NullJSON(data, "description"),
		CycleDayFrom: form.NullInt16(data, "cycle_day_from"), CycleDayTo: form.NullInt16(data, "cycle_day_to"),
		Category: form.Str(data, "category"), IsActive: httpadmin.Bool(data, "is_active"),
		SortOrder: form.Int32(data, "sort_order", 0), Now: h.now(c),
	})
	if err != nil {
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "challenge.create", "challenge", id)
	return h.challenges().respond(c, id, true, "Challenge created.")
}

// UpdateChallenge is PUT /challenges/:id.
func (h *Handlers) UpdateChallenge(c fiber.Ctx) error {
	cur, err := find(c, "Challenge", h.q.GetChallenge)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, challengeRules(c), checkDayRange(c))
	if err != nil {
		return err
	}
	if err := h.q.UpdateChallenge(c.Context(), store.UpdateChallengeParams{
		Title: form.ReqJSON(data, "title"), Description: form.KeepJSON(data, "description", cur.Description),
		CycleDayFrom: form.NullInt16(data, "cycle_day_from"), CycleDayTo: form.NullInt16(data, "cycle_day_to"),
		Category: form.KeepStr(data, "category", cur.Category), IsActive: httpadmin.Bool(data, "is_active"),
		SortOrder: form.Int32(data, "sort_order", 0), Now: h.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "challenge.update", "challenge", cur.ID)
	return h.challenges().respond(c, cur.ID, false, "Challenge updated.")
}

// DestroyChallenge is DELETE /challenges/:id.
func (h *Handlers) DestroyChallenge(c fiber.Ctx) error {
	return h.challenges().destroy(c, h.logger, "Challenge deleted.")
}

// ToggleChallenge is POST /challenges/:id/toggle (is_active).
func (h *Handlers) ToggleChallenge(c fiber.Ctx) error {
	return h.challenges().doToggle(c, h.logger, "Status changed.")
}

// ---------------------------------------------------------------------------
// Completions report

var (
	minDate = civildate.New(1000, 1, 1)
	maxDate = civildate.New(9999, 12, 31)
)

// ChallengeCompletions is GET /challenge-completions?challenge_id=&q=&from=&to=&page=&per_page=
// (ChallengeCompletionController::index): the completions, totals for the filter,
// completions today, a per-challenge leaderboard and the challenge list for the filter.
func (h *Handlers) ChallengeCompletions(c fiber.Ctx) error {
	data, err := form.Validate(c, validation.Rules{
		validation.F("challenge_id", "nullable|integer"),
		validation.F("from", "nullable|date"),
		validation.F("to", "nullable|date"),
	}, func(in phpval.Map, add form.Add) error {
		v, ok := in.Get("challenge_id")
		if !ok || v == nil || !phpval.IsNumeric(v) || phpval.ToFloat(v) < 1 {
			if ok && v != nil && phpval.IsNumeric(v) {
				add("challenge_id", form.Msg(c, "validation.exists", "challenge_id"))
			}
			return nil
		}
		found, err := h.q.ChallengeExists(c.Context(), uint64(phpval.ToFloat(v)))
		if err != nil {
			return err
		}
		if !found {
			add("challenge_id", form.Msg(c, "validation.exists", "challenge_id"))
		}
		return nil
	})
	if err != nil {
		return err
	}
	var challengeID uint64
	if form.Has(data, "challenge_id") {
		challengeID = uint64(form.Int(data, "challenge_id", 0)) //nolint:gosec // G115: validated ≥ 1
	}
	from, to := minDate, maxDate
	if t, ok := form.ParseTime(httpadmin.String(data, "from")); ok {
		from = civildate.FromTime(t)
	}
	if t, ok := form.ParseTime(httpadmin.String(data, "to")); ok {
		to = civildate.FromTime(t)
	}
	search := strings.TrimSpace(c.Query("q"))
	pattern := sql.NullString{String: form.Contains(search), Valid: true}

	totals, err := h.q.CompletionTotals(c.Context(), store.CompletionTotalsParams{
		ChallengeID: challengeID, DateFrom: from, DateTo: to, Pattern: pattern,
	})
	if err != nil {
		return err
	}
	today, err := h.q.CountCompletionsOn(c.Context(), civildate.InTehran(httpadmin.Now(c)))
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListCompletions(c.Context(), store.ListCompletionsParams{
		ChallengeID: challengeID, DateFrom: from, DateTo: to, Pattern: pattern,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	per, err := h.q.CompletionsPerChallenge(c.Context(), store.CompletionsPerChallengeParams{
		ChallengeID: challengeID, DateFrom: from, DateTo: to, Pattern: pattern,
	})
	if err != nil {
		return err
	}
	titles, err := h.q.ListChallengeTitles(c.Context())
	if err != nil {
		return err
	}

	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj(
			"id", r.ID,
			"completion_date", r.CompletionDate,
			"completed_at", httpadmin.Time(r.CompletedAt),
			"user", jsonx.Obj("id", r.UserID, "name", httpadmin.NullString(r.UserName),
				"mobile", httpadmin.NullString(r.UserMobile)),
			"challenge", jsonx.Obj("id", r.ChallengeID, "title", form.NullRaw(r.ChallengeTitle)),
		))
	}
	leaders := make([]*jsonx.OrderedMap, 0, len(per))
	for _, r := range per {
		leaders = append(leaders, jsonx.Obj("challenge_id", r.ChallengeID, "title", form.NullRaw(r.ChallengeTitle),
			"completions", r.Completions, "users", r.Users))
	}
	list := make([]*jsonx.OrderedMap, 0, len(titles))
	for _, t := range titles {
		list = append(list, jsonx.Obj("id", t.ID, "title", form.Raw(t.Title)))
	}
	nullable := func(key string) any {
		if s := httpadmin.String(data, key); s != "" {
			return s
		}
		return nil
	}
	var cid any
	if challengeID > 0 {
		cid = challengeID
	}
	page := httpadmin.Page(items, p, int(totals.Total))
	page.Set("filters", jsonx.Obj("challenge_id", cid, "q", search, "from", nullable("from"), "to", nullable("to")))
	page.Set("stats", jsonx.Obj("total", totals.Total, "users", totals.Users, "today", today))
	page.Set("per_challenge", jsonx.List(leaders))
	page.Set("challenges", jsonx.List(list))
	return httpadmin.OK(c, page)
}
