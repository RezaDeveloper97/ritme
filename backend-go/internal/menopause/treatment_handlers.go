package menopause

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Treatment & care routes (CB-MENO-03). Every write answers with the treatment screen of the week it touched, so the
// client redraws from one body.

// treatmentErr turns the treatment service errors into responses.
func treatmentErr(err error, locale string) error {
	switch {
	case errors.Is(err, ErrItemNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.item_not_found", locale))
	case errors.Is(err, ErrItemLimit):
		msg := T("messages.item_limit", locale)
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg,
			"errors", jsonx.Obj("limit", []string{msg}), "error_code", care.ErrorCodeLimitReached)
	case errors.Is(err, ErrMedicationLimit):
		msg := T("messages.medication_limit", locale)
		return httpx.Fail(fiber.StatusUnprocessableEntity, msg,
			"errors", jsonx.Obj("limit", []string{msg}), "error_code", care.ErrorCodeLimitReached)
	case errors.Is(err, ErrItemInactive):
		return fieldFail(locale, "date", "item_inactive")
	}
	return err
}

func (h *Handlers) itemID(c fiber.Ctx, locale string) (uint64, error) {
	id, ok := parseID(c.Params("id"))
	if !ok {
		return 0, httpx.Fail(fiber.StatusNotFound, T("messages.item_not_found", locale))
	}
	return id, nil
}

func (h *Handlers) screen(c fiber.Ctx, userID uint64, day civildate.Date, status int, msg ...string) error {
	sc, err := h.svc.TreatmentScreen(c, userID, day)
	if err != nil {
		return err
	}
	body := TreatmentScreenJSON(sc, localizer(c))
	if status == fiber.StatusCreated {
		return httpx.Created(c, body, msg...)
	}
	return httpx.OK(c, body, msg...)
}

// subtitleLocale is the language of the legacy reminders.subtitle (the default language, like /care).
func subtitleLocale(c fiber.Ctx) string { return i18n.LanguagesOf(c).DefaultCode() }

// Treatment is GET /menopause/treatment?date=: the items with the week's dots, the next review, the week's side
// effects and the treatment tips.
func (h *Handlers) Treatment(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	day, err := validateDayQuery(validation.Query(c), i18n.Locale(c), h.now(c))
	if err != nil {
		return err
	}
	return h.screen(c, userID, day, fiber.StatusOK)
}

// StoreItem is POST /menopause/treatment/items {kind, name, dose?, schedule?, form?, started_on?, review_on?,
// stopped_on?, weekly_goal?, goal_unit?, remind?}: 201 with the screen.
func (h *Handlers) StoreItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateItem(validation.Input(c), "", locale, now)
	if err != nil {
		return err
	}
	if _, err := h.svc.CreateItem(c, userID, in, now, subtitleLocale(c)); err != nil {
		return treatmentErr(err, locale)
	}
	return h.screen(c, userID, civildate.InTehran(now), fiber.StatusCreated, T("messages.item_saved", locale))
}

// UpdateItem is PUT /menopause/treatment/items/{id}: a full replace (kind stays); stopped_on stops it.
func (h *Handlers) UpdateItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := h.itemID(c, locale)
	if err != nil {
		return err
	}
	it, err := h.svc.GetItem(c, userID, id)
	if err != nil {
		return treatmentErr(err, locale)
	}
	in, err := validateItem(validation.Input(c), it.Kind, locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdateItem(c, userID, id, in, now, subtitleLocale(c)); err != nil {
		return treatmentErr(err, locale)
	}
	return h.screen(c, userID, civildate.InTehran(now), fiber.StatusOK, T("messages.item_saved", locale))
}

// DestroyItem is DELETE /menopause/treatment/items/{id}: the item, its intakes and its care reminder.
func (h *Handlers) DestroyItem(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := h.itemID(c, locale)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteItem(c, userID, id); err != nil {
		return treatmentErr(err, locale)
	}
	return h.screen(c, userID, civildate.InTehran(now), fiber.StatusOK, T("messages.item_removed", locale))
}

// LogIntake is PUT /menopause/treatment/items/{id}/intakes/{date} {amount?}: taken that day (idempotent).
func (h *Handlers) LogIntake(c fiber.Ctx) error {
	return h.intake(c, true)
}

// UnlogIntake is DELETE /menopause/treatment/items/{id}/intakes/{date}: not taken that day (idempotent).
func (h *Handlers) UnlogIntake(c fiber.Ctx) error {
	return h.intake(c, false)
}

func (h *Handlers) intake(c fiber.Ctx, taken bool) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := h.itemID(c, locale)
	if err != nil {
		return err
	}
	it, err := h.svc.GetItem(c, userID, id)
	if err != nil {
		return treatmentErr(err, locale)
	}
	day, err := parseLogDay(c.Params("date"), locale, now)
	if err != nil {
		return err
	}
	if !taken {
		if err := h.svc.UnlogIntake(c, userID, id, day); err != nil {
			return treatmentErr(err, locale)
		}
		return h.screen(c, userID, day, fiber.StatusOK, T("messages.intake_removed", locale))
	}
	amount, err := validateIntake(validation.Input(c), ItemKind{Kind: it.Kind, Unit: it.GoalUnit.String}, locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.LogIntake(c, userID, id, day, amount, now); err != nil {
		return treatmentErr(err, locale)
	}
	return h.screen(c, userID, day, fiber.StatusOK, T("messages.intake_saved", locale))
}

// SaveSideEffects is PUT /menopause/treatment/side-effects/{date} {codes, treatment_item_id?}: the day's side
// effects as a set (empty clears).
func (h *Handlers) SaveSideEffects(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	day, err := parseLogDay(c.Params("date"), locale, now)
	if err != nil {
		return err
	}
	in, err := validateSideEffects(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.SaveSideEffects(c, userID, day, in, now); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			return fieldFail(locale, "treatment_item_id", "item_unknown")
		}
		return err
	}
	return h.screen(c, userID, day, fiber.StatusOK, T("messages.side_effects_saved", locale))
}

// Report is GET /menopause/report?months=1|3|6: the owner's preview of the menopause doctor report (the same data
// as the `menopause` section of GET /health-record/report).
func (h *Handlers) Report(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	months, err := validateReportMonths(validation.Query(c), locale, now)
	if err != nil {
		return err
	}
	from, to := ReportWindow(civildate.InTehran(now), months)
	r, err := h.svc.Report(c, userID, from, to)
	if err != nil {
		return err
	}
	name := SymptomNamerFor(h.labels, locale, i18n.LanguagesOf(c).DefaultCode())
	return httpx.OK(c, jsonx.Obj("months", months, "months_options", ReportMonths, "empty", r.Empty(),
		"report", ReportJSON(r, name)))
}
