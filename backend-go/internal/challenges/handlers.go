package challenges

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Error codes of the 404 / 409 / 422 bodies.
const (
	ErrorCodeNotFound     = "challenge_not_found"
	ErrorCodeNotEligible  = "challenge_not_eligible"
	ErrorCodeNotStartable = "challenge_not_startable"
	ErrorCodeNotJoined    = "challenge_not_joined"
	ErrorCodeBadDay       = "challenge_bad_day"
	ErrorCodeRunEnded     = "challenge_run_ended"
)

// Handlers are the /api/v1/challenges actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// today is the user-local calendar day of the request (T-M9-26).
func (h *Handlers) today(c fiber.Ctx) civildate.Date {
	return civildate.LocalToday(c.Context(), clock.FromContext(c, h.clock))
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func fail(err error, locale string) error {
	status, key, code := 0, "", ""
	switch {
	case errors.Is(err, ErrNotFound):
		status, key, code = fiber.StatusNotFound, "not_found", ErrorCodeNotFound
	case errors.Is(err, ErrContraindicated): // same 403 and code as not eligible; reason tells them apart (D-167)
		return httpx.Fail(fiber.StatusForbidden, T("messages.contraindicated", locale), "error_code", ErrorCodeNotEligible,
			"reason", ReasonContraindicated)
	case errors.Is(err, ErrNotEligible):
		return httpx.Fail(fiber.StatusForbidden, T("messages.not_eligible", locale), "error_code", ErrorCodeNotEligible,
			"reason", ReasonNotEligible)
	case errors.Is(err, ErrNotStartable):
		status, key, code = fiber.StatusConflict, "not_startable", ErrorCodeNotStartable
	case errors.Is(err, ErrNotJoined):
		status, key, code = fiber.StatusConflict, "not_joined", ErrorCodeNotJoined
	case errors.Is(err, ErrBadDay):
		status, key, code = fiber.StatusUnprocessableEntity, "bad_day", ErrorCodeBadDay
	case errors.Is(err, ErrRunEnded):
		status, key, code = fiber.StatusConflict, "run_ended", ErrorCodeRunEnded
	default:
		return err
	}
	return httpx.Fail(status, T("messages."+key, locale), "error_code", code)
}

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

func decode(raw json.RawMessage, locale, def string) any {
	var v any
	if err := json.Unmarshal(i18n.Pick(raw, locale, def), &v); err != nil {
		return nil
	}
	return v
}

func decodeNull(raw db.NullRawJSON, locale, def string) any {
	if !raw.Valid {
		return nil
	}
	return decode(raw.V, locale, def)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (h *Handlers) itemJSON(it Item, locale, def string) *jsonx.OrderedMap {
	return ItemJSON(it, locale, def)
}

// ItemJSON is one challenge of GET /challenges in locale (def = the default language), also the home card's run.
func ItemJSON(it Item, locale, def string) *jsonx.OrderedMap {
	p := it.Program
	tasks := make([]*jsonx.OrderedMap, 0, len(it.Tasks))
	for _, t := range it.Tasks {
		tasks = append(tasks, jsonx.Obj("id", t.ID, "day_from", t.DayFrom, "day_to", t.DayTo, "title", decode(t.Title, locale, def)))
	}
	var startsOn any
	if p.StartsOn.Valid {
		startsOn = p.StartsOn.Date.String()
	}
	var minDays any
	if p.CompletionMinDays > 0 {
		minDays = p.CompletionMinDays
	}
	var enr any
	if e := it.Enrollment; e != nil {
		doneToday := make([]uint64, 0, len(e.DoneToday))
		for _, t := range DayTasks(it.Tasks, e.Day) {
			if e.DoneToday[t.ID] {
				doneToday = append(doneToday, t.ID)
			}
		}
		var completedAt any
		if e.CompletedAt.Valid {
			completedAt = e.CompletedAt.Time.In(civildate.Tehran).Format("2006-01-02T15:04:05-07:00")
		}
		enr = jsonx.Obj("status", e.Status, "started_on", e.StartedOn.String(), "day", e.Day,
			"done_days", e.DoneDays, "required_days", e.RequiredDays, "done_task_ids_today", doneToday,
			"completed_at", completedAt, "reward_granted", e.RewardGranted)
	}
	return jsonx.Obj(
		"id", p.ID,
		"title", decode(p.Title, locale, def),
		"description", decodeNull(p.Description, locale, def),
		"category", nullStr(p.Category),
		"duration_days", p.DurationDays,
		"start_rule", p.StartRule,
		"start_phase", nullStr(p.StartPhase),
		"starts_on", startsOn,
		"reminder_time", nullStr(p.ReminderTime),
		"completion_min_days", minDays,
		"reward", jsonx.Obj("badge_code", nullStr(p.RewardBadgeCode), "badge_title", decodeNull(p.RewardBadgeTitle, locale, def),
			"plus_days", p.RewardPlusDays),
		"tasks", tasks,
		"can_join", it.CanJoin,
		"contraindicated", it.Contraindicated,
		"enrollment", enr,
	)
}

// Index is GET /challenges: the active multi-day challenges the user may join or already joined.
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	items, err := h.svc.List(c.Context(), userID, h.today(c))
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(items))
	for _, it := range items {
		out = append(out, h.itemJSON(it, locale, def))
	}
	return httpx.OK(c, jsonx.Obj("challenges", out))
}

// Join is POST /challenges/{id}/join: start the run (idempotent).
func (h *Handlers) Join(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	now := h.now(c)
	it, created, err := h.svc.Join(c.Context(), userID, id, h.today(c), now)
	if err != nil {
		return fail(err, locale)
	}
	body := jsonx.Obj("challenge", h.itemJSON(it, locale, def))
	if created {
		return httpx.Created(c, body, T("messages.joined", locale))
	}
	return httpx.OK(c, body)
}

// CompleteTask is POST /challenges/{id}/tasks/{task}/complete {day?}: mark a task done (idempotent); the last one
// of the completion rule finishes the run and grants the reward once.
func (h *Handlers) CompleteTask(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	id, ok1 := idParam(c, "id")
	taskID, ok2 := idParam(c, "task")
	if !ok1 || !ok2 {
		return fail(ErrNotFound, locale)
	}
	now := h.now(c)
	in := validation.Input(c)
	v := validation.Make(lang.Default(), locale, in, validation.Rules{validation.F("day", "nullable|integer|min:1|max:366")},
		validation.Now(now))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", v.ErrorBag())
	}
	var day *int
	if raw, ok := in.Get("day"); ok && raw != nil {
		n, _ := strconv.Atoi(phpval.ToString(raw))
		day = &n
	}
	res, err := h.svc.CompleteTask(c.Context(), userID, id, taskID, day, h.today(c), now)
	if err != nil {
		return fail(err, locale)
	}
	msg := T("messages.task_done", locale)
	if res.JustFinally {
		msg = T("messages.completed", locale)
	}
	return httpx.OK(c, jsonx.Obj("challenge", h.itemJSON(res.Item, locale, def), "task_id", taskID,
		"completed_now", res.JustFinally), msg)
}

// Rewards is GET /challenges/rewards: the badges the user earned (and the Plus days they brought).
func (h *Handlers) Rewards(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	rows, err := h.svc.Rewards(c.Context(), userID)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		granted := ""
		if r.GrantedAt.Valid {
			granted = r.GrantedAt.Time.In(civildate.Tehran).Format("2006-01-02T15:04:05-07:00")
		}
		out = append(out, jsonx.Obj("challenge_id", r.ChallengeID, "challenge_title", decode(r.ChallengeTitle, locale, def),
			"badge_code", nullStr(r.BadgeCode.String), "badge_title", decodeNull(r.BadgeTitle, locale, def),
			"plus_days", r.PlusDays, "granted_at", nullStr(granted)))
	}
	return httpx.OK(c, jsonx.Obj("rewards", out))
}
