package menopause

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Labels resolves a client translation namespace for a locale (i18n.TranslationStore.NamespaceMessages); the
// patterns read trigger names from the log-taxonomy namespace. nil = codes only.
type Labels interface {
	NamespaceMessages(code, ns, defaultCode string) any
}

// Handlers are the /api/v1/menopause actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc    *Service
	labels Labels
	clock  clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(svc *Service, labels Labels, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, labels: labels, clock: base}
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

func localizer(c fiber.Ctx) catalog.Localizer {
	langs := i18n.LanguagesOf(c)
	return catalog.Localizer{Locale: i18n.Locale(c), Default: langs.DefaultCode(), Langs: langs}
}

func notFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.flash_not_found", locale))
}

func (h *Handlers) profile(c fiber.Ctx, st Stage, msg ...string) error {
	tip, err := h.svc.StageTip(c, st)
	if err != nil {
		return err
	}
	return httpx.OK(c, ProfileJSON(st, tip, localizer(c)), msg...)
}

// ShowProfile is GET /menopause/profile: the stored answers, the effective stage and the months without a period.
func (h *Handlers) ShowProfile(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	st, err := h.svc.Profile(c, userID, civildate.InTehran(h.now(c)))
	if err != nil {
		return err
	}
	return h.profile(c, st)
}

// SaveProfile is PUT /menopause/profile {stage?, last_period?, surgical?, hrt?} (partial; null clears).
func (h *Handlers) SaveProfile(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	ch, err := validateProfile(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	st, err := h.svc.SaveProfile(c, userID, ch, now)
	if err != nil {
		return err
	}
	return h.profile(c, st, T("messages.profile_saved", locale))
}

// Today is GET /menopause/today: the home read model.
func (h *Handlers) Today(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	t, err := h.svc.Today(c, userID, now)
	if err != nil {
		return err
	}
	langs := i18n.LanguagesOf(c)
	return httpx.OK(c, TodayJSON(t, now, localizer(c), checkups.Lang{Locale: i18n.Locale(c), Default: langs.DefaultCode()}))
}

// ListFlashes is GET /menopause/hot-flashes?date=: the day's flashes, tiles and the running timer.
func (h *Handlers) ListFlashes(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	day, err := validateDayQuery(validation.Query(c), i18n.Locale(c), now)
	if err != nil {
		return err
	}
	d, err := h.svc.Day(c, userID, day, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, FlashDayJSON(d, now))
}

// StartFlash is POST /menopause/hot-flashes: 201 with a new timer (or a finished flash when duration_s is sent);
// 200 with the timer already running.
func (h *Handlers) StartFlash(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateFlashStart(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	f, created, err := h.svc.StartFlash(c, userID, in, now)
	if err != nil {
		return err
	}
	switch {
	case !created:
		return httpx.OK(c, FlashJSON(f, now), T("messages.flash_running", locale))
	case f.Running():
		return httpx.Created(c, FlashJSON(f, now), T("messages.flash_started", locale))
	}
	return httpx.Created(c, FlashJSON(f, now), T("messages.flash_logged", locale))
}

// StopFlash is POST /menopause/hot-flashes/{id}/stop: stops the timer and saves the details (on a stopped flash it
// edits the details only).
func (h *Handlers) StopFlash(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(locale)
	}
	in, err := validateFlashStop(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	f, err := h.svc.StopFlash(c, userID, id, in, now)
	if errors.Is(err, ErrFlashNotFound) {
		return notFound(locale)
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, FlashJSON(f, now), T("messages.flash_saved", locale))
}

// Scores is GET /menopause/scores?months=: the chart, the questionnaires with deltas, the bands and the HRT note.
func (h *Handlers) Scores(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	months, err := validateMonths(validation.Query(c), i18n.Locale(c), now)
	if err != nil {
		return err
	}
	sc, err := h.svc.Scale(c)
	if err != nil {
		return err
	}
	hist, err := h.svc.History(c, userID, civildate.InTehran(now), months)
	if err != nil {
		return err
	}
	return httpx.OK(c, HistoryJSON(hist, sc, localizer(c)))
}

// SaveScore is POST /menopause/scores {month?, answers}: stores (or replaces) the month's questionnaire and returns
// it with its band, domains and delta.
func (h *Handlers) SaveScore(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	sc, err := h.svc.Scale(c)
	if err != nil {
		return err
	}
	in, err := validateScore(validation.Input(c), sc, locale, now)
	if err != nil {
		return err
	}
	score := sc.Compute(in.Month, in.Answers)
	if err := h.svc.SaveScore(c, userID, score, now); err != nil {
		return err
	}
	prev, err := h.svc.PreviousScore(c, userID, in.Month)
	if err != nil {
		return err
	}
	return httpx.OK(c, ScoreJSON(ScoreEntry{Score: score, Previous: prev}, sc, localizer(c)), T("messages.score_saved", locale))
}

// Patterns is GET /menopause/patterns: associations in her own logs, never a diagnosis.
func (h *Handlers) Patterns(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	ps, err := h.svc.Patterns(c, userID, civildate.InTehran(h.now(c)))
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	var taxonomyNS any
	if h.labels != nil {
		taxonomyNS = h.labels.NamespaceMessages(locale, healthlog.TaxonomyNamespace, i18n.LanguagesOf(c).DefaultCode())
	}
	names := analysis.NewCopy(nil, taxonomyNS)
	text := func(p Pattern) string {
		params := map[string]string{}
		if p.Trigger != "" {
			params["trigger"] = names.SymptomLabel(paramTriggers + "." + p.Trigger)
		}
		return Tp("patterns."+p.Key+"."+p.Direction(), params, locale)
	}
	return httpx.OK(c, PatternsJSON(ps, text, localizer(c)))
}
