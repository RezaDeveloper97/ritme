package postpartum

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/messages/content"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

// ErrorCodeNotActive is the error_code of the 409 a write gets outside postpartum mode.
const ErrorCodeNotActive = "postpartum_not_active"

// Handlers are the /api/v1/postpartum actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	mq    *mstore.Queries // message_contents (the guide copy)
	clock clock.Clock
}

// NewHandlers wires the handlers on db; base is the fallback clock (tests pin it per request).
func NewHandlers(db mstore.DBTX, svc *Service, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, mq: mstore.New(db), clock: base}
}

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// now is the request time, Tehran wall-clock, whole seconds.
func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func (h *Handlers) copyFor(c fiber.Ctx) *guide.Copy {
	return guide.NewCopy(content.NewRepository(h.mq), i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode())
}

func notActive(locale string) error {
	return httpx.Fail(fiber.StatusConflict, T("messages.not_active", locale), "error_code", ErrorCodeNotActive)
}

// activeState loads the state and refuses a companion account (409 companion_account) and a user outside postpartum
// mode or without a birth date (409 postpartum_not_active).
func (h *Handlers) activeState(c fiber.Ctx, userID uint64) (State, error) {
	st, err := h.svc.State(c, userID)
	if err != nil {
		return State{}, err
	}
	locale := i18n.Locale(c)
	if st.Mode == enums.LifeModeCompanion {
		return State{}, companion.AccountConflict(locale)
	}
	if !st.Active() || st.Profile == nil {
		return State{}, notActive(locale)
	}
	return st, nil
}

// overview is the GET /postpartum payload.
func (h *Handlers) overview(c fiber.Ctx, userID uint64, st State, now time.Time) (*jsonx.OrderedMap, error) {
	ctx := context.Context(c)
	locale := i18n.Locale(c)
	today := civildate.InTehran(now)
	cp := h.copyFor(c)
	callWhen, err := cp.Text(ctx, guide.AlertGroup, guide.AlertCallWhen)
	if err != nil {
		return nil, err
	}
	out := jsonx.Obj(
		"active", st.Active(),
		"mode", string(st.Mode),
		"setup_required", st.Active() && st.Profile == nil,
		"profile", ProfileJSON(st.Profile, locale),
		"status", nil,
		"checkin", nil,
		"today", nil,
		"alerts", []any{},
		"week_tip", nil,
		"call_when", jsonx.Obj("title", callWhen["title"], "body", callWhen["body"]),
	)
	if !st.Active() || st.Profile == nil {
		return out, nil
	}
	birth := st.Profile.BirthDate
	status := guide.StatusOn(birth, today)
	out.Set("status", StatusJSON(status, locale))

	last, lastFull, err := h.svc.LastChecks(ctx, userID)
	if err != nil {
		return nil, err
	}
	sch := ScheduleOn(today, birth, last, lastFull)
	out.Set("checkin", ScheduleJSON(sch, last, &birth, locale))

	rec, err := h.svc.Recovery(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	alerts, err := RecoveryAlerts(ctx, cp, rec)
	if err != nil {
		return nil, err
	}
	out.Set("today", RecoveryJSON(today, rec, alerts))
	if a, err := CheckinAlert(ctx, cp, sch); err != nil {
		return nil, err
	} else if a != nil {
		alerts = append(alerts, a)
	}
	out.Set("alerts", alerts)

	tipKey := guide.TipKey(status.Week)
	tip, err := cp.Text(ctx, guide.TipGroup, tipKey)
	if err != nil {
		return nil, err
	}
	if tip["title"] != "" {
		out.Set("week_tip", jsonx.Obj("week", status.Week, "key", tipKey, "title", tip["title"], "body", tip["body"]))
	}
	return out, nil
}

// Show is GET /postpartum: the mode, the birth, the time since it, the check-in schedule, today's recovery log with
// its alerts, the week's tip and «کی فوراً تماس بگیرم؟». Outside postpartum mode: active false, the rest null.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	st, err := h.svc.State(c, userID)
	if err != nil {
		return err
	}
	out, err := h.overview(c, userID, st, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

// Activate is POST /postpartum/activate {birth_date, delivery_type?, baby_count?}: from an active pregnancy (closed
// as delivered) or directly; again later = edit the birth. Answers the GET /postpartum payload.
func (h *Handlers) Activate(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	st, err := h.svc.State(c, userID)
	if err != nil {
		return err
	}
	if st.Mode == enums.LifeModeCompanion {
		return companion.AccountConflict(locale)
	}
	in, err := validateActivate(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Activate(c, userID, in, now); err != nil {
		return err
	}
	if st, err = h.svc.State(c, userID); err != nil {
		return err
	}
	out, err := h.overview(c, userID, st, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, out, T("messages.activated", locale))
}

// ShowRecovery is GET /postpartum/recovery?date=: the day's recovery log (409 outside postpartum mode).
func (h *Handlers) ShowRecovery(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if _, err := h.activeState(c, userID); err != nil {
		return err
	}
	day, err := validateDate(validation.Query(c), i18n.Locale(c), h.now(c))
	if err != nil {
		return err
	}
	rec, err := h.svc.Recovery(c, userID, day)
	if err != nil {
		return err
	}
	alerts, err := RecoveryAlerts(c, h.copyFor(c), rec)
	if err != nil {
		return err
	}
	return httpx.OK(c, RecoveryJSON(day, rec, alerts))
}

// SaveRecovery is PUT /postpartum/recovery {date?, …}: a partial save of the day's recovery log.
func (h *Handlers) SaveRecovery(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if _, err := h.activeState(c, userID); err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	body := validation.Input(c)
	day, err := validateDate(body, locale, now)
	if err != nil {
		return err
	}
	stored, err := h.svc.Recovery(c, userID, day)
	if err != nil {
		return err
	}
	in, err := validateRecovery(body, stored, locale, now)
	if err != nil {
		return err
	}
	rec, err := h.svc.SaveRecovery(c, userID, day, in, locale, now)
	if err != nil {
		return err
	}
	alerts, err := RecoveryAlerts(c, h.copyFor(c), rec)
	if err != nil {
		return err
	}
	return httpx.OK(c, RecoveryJSON(day, rec, alerts), T("messages.recovery_saved", locale))
}

// Questions is GET /postpartum/epds/questions?kind=short|full: the questionnaire with the option scores.
func (h *Handlers) Questions(c fiber.Ctx) error {
	if _, err := h.user(c); err != nil {
		return err
	}
	locale := i18n.Locale(c)
	kind, err := validateKind(validation.Query(c), locale, h.now(c))
	if err != nil {
		return err
	}
	q, err := QuestionsJSON(c, h.copyFor(c), kind, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, q)
}

// SaveCheck is POST /postpartum/epds {kind, answers}: scores and stores today's check and answers it with its band,
// the safety message (urgent with call actions when item 10 > 0 or the full total ≥ 13) and the next due check.
// The answers are never logged.
func (h *Handlers) SaveCheck(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	st, err := h.activeState(c, userID)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	kind, answers, err := validateCheck(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	today := civildate.InTehran(now)
	res := Score(kind, answers)
	saved, err := h.svc.SaveCheck(c, userID, res, today, now)
	if err != nil {
		return err
	}
	cp := h.copyFor(c)
	safety, err := SafetyJSON(c, cp, res)
	if err != nil {
		return err
	}
	last, lastFull, err := h.svc.LastChecks(c, userID)
	if err != nil {
		return err
	}
	birth := st.Profile.BirthDate
	var followUp any
	if res.FollowUp != "" {
		followUp = res.FollowUp
	}
	return httpx.OK(c, jsonx.Obj(
		"check", CheckJSON(saved, &birth, locale),
		"safety", safety,
		"follow_up", followUp,
		"checkin", ScheduleJSON(ScheduleOn(today, birth, last, lastFull), last, &birth, locale),
	), T("messages.check_saved", locale))
}

// History is GET /postpartum/epds?limit=: the user's checks, newest first (totals and bands, never the answers).
func (h *Handlers) History(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	limit, err := validateLimit(validation.Query(c), locale, h.now(c))
	if err != nil {
		return err
	}
	st, err := h.svc.State(c, userID)
	if err != nil {
		return err
	}
	checks, err := h.svc.History(c, userID, limit)
	if err != nil {
		return err
	}
	var birth *civildate.Date
	if st.Profile != nil {
		b := st.Profile.BirthDate
		birth = &b
	}
	list := make([]*jsonx.OrderedMap, 0, len(checks))
	for _, ch := range checks {
		list = append(list, CheckJSON(ch, birth, locale))
	}
	return httpx.OK(c, jsonx.Obj("checks", list, "thresholds", jsonx.Obj(
		"short_elevated", ShortElevatedMin, "full_possible", FullPossibleMin, "full_likely", FullLikelyMin,
	)))
}
