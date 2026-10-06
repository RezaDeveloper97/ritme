package healthrecord

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

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

// ErrorCodeNotFound is the 404 code of an unknown or foreign manual pregnancy.
const ErrorCodeNotFound = "record_pregnancy_not_found"

// Handlers are the /api/v1/health-record actions. Mount them behind the locale middleware and auth RequireUser.
// There is no id of another user anywhere: every action reads and writes the authenticated user's own rows, and no
// action logs a health value.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func notFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale), "error_code", ErrorCodeNotFound)
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func entryID(c fiber.Ctx) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

func (h *Handlers) record(c fiber.Ctx, userID uint64, now time.Time) (*jsonx.OrderedMap, error) {
	rec, err := h.svc.Build(c, userID, AudienceOwner, Options{
		Today: civildate.InTehran(now), Locale: i18n.Locale(c), DefaultLocale: i18n.LanguagesOf(c).DefaultCode(),
	})
	if err != nil {
		return nil, err
	}
	return rec.JSON(), nil
}

// Show is GET /health-record (nbl_Record_Summary): the owner's record, every section.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	body, err := h.record(c, userID, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}

func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func pick(body phpval.Map, keys ...string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

// UpdateBasics is PUT /health-record/basics {blood_type?, allergies?}: a present key replaces its value (null clears
// it; allergies [] = «ندارم»), an absent key keeps it. Returns the whole record.
func (h *Handlers) UpdateBasics(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	body := validation.Input(c)
	data := pick(body, "blood_type", "allergies")
	if err := validate(data, validation.Rules{
		validation.F("blood_type", "nullable", "string", validation.In(BloodTypes...)),
		validation.F("allergies", "nullable", "array", "max:"+strconv.Itoa(MaxAllergies)),
		validation.F("allergies.*", "required", "string", "max:"+strconv.Itoa(MaxAllergyLen)),
	}, locale, now); err != nil {
		return err
	}
	in := BasicsInput{}
	if _, ok := data.Get("blood_type"); ok {
		in.SetBloodType, in.BloodType = true, str(data, "blood_type")
	}
	if v, ok := data.Get("allergies"); ok {
		in.SetAllergies = true
		if v != nil {
			in.Allergies = cleanAllergies(v)
			if in.Allergies == nil {
				return failValidation(locale, jsonx.Obj("allergies.0", []string{T("validation.allergy_blank", locale)}))
			}
		}
	}
	if err := h.svc.SaveBasics(c, userID, in, now); err != nil {
		return err
	}
	rec, err := h.record(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, rec, T("messages.saved", locale))
}

// cleanAllergies trims, drops case-insensitive duplicates (first spelling wins) and keeps order; nil when an item is
// blank (the 422), [] for an empty list.
func cleanAllergies(v any) []string {
	_, vals := phpval.Entries(v)
	out := []string{}
	seen := map[string]bool{}
	for _, x := range vals {
		s := strings.Join(strings.Fields(phpval.ToString(x)), " ")
		if s == "" {
			return nil
		}
		k := strings.ToLower(s)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

// minEndedOn is the earliest accepted manual pregnancy date.
const minEndedOn = "1950-01-01"

func (h *Handlers) pregnancyInput(c fiber.Ctx, locale string, now time.Time) (PregnancyInput, error) {
	data := pick(validation.Input(c), "outcome", "ended_on", "baby_count")
	if err := validate(data, validation.Rules{
		validation.F("outcome", "required", "string", validation.In(Outcomes...)),
		validation.F("ended_on", "nullable", "date", "after_or_equal:"+minEndedOn, "before_or_equal:today"),
		validation.F("baby_count", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxBabyCount)),
	}, locale, now); err != nil {
		return PregnancyInput{}, err
	}
	in := PregnancyInput{Outcome: str(data, "outcome")}
	if s := str(data, "ended_on"); s != "" {
		if t, err := civildate.ParseLenient(s, now, civildate.Tehran); err == nil {
			in.EndedOn = civildate.InTehran(t)
		}
	}
	if s := str(data, "baby_count"); s != "" {
		in.BabyCount, _ = strconv.Atoi(s)
	}
	return in, nil
}

// StorePregnancy is POST /health-record/pregnancies {outcome, ended_on?, baby_count?} (201 with the entry).
func (h *Handlers) StorePregnancy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := h.pregnancyInput(c, locale, now)
	if err != nil {
		return err
	}
	row, err := h.svc.CreatePregnancy(c, userID, in, now)
	if errors.Is(err, ErrTooMany) {
		return failValidation(locale, jsonx.Obj("outcome", []string{
			Tp("validation.too_many_pregnancies", map[string]string{"max": strconv.Itoa(MaxManualPregnancy)}, locale),
		}))
	}
	if err != nil {
		return err
	}
	return httpx.Created(c, ManualJSON(row), T("messages.created", locale))
}

// UpdatePregnancy is PUT /health-record/pregnancies/{id}: a full replace of a manual entry.
func (h *Handlers) UpdatePregnancy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := entryID(c)
	if !ok {
		return notFound(locale)
	}
	if _, err := h.svc.GetPregnancy(c, userID, id); errors.Is(err, ErrNotFound) {
		return notFound(locale)
	} else if err != nil {
		return err
	}
	in, err := h.pregnancyInput(c, locale, now)
	if err != nil {
		return err
	}
	row, err := h.svc.UpdatePregnancy(c, userID, id, in, now)
	if errors.Is(err, ErrNotFound) {
		return notFound(locale)
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, ManualJSON(row), T("messages.updated", locale))
}

// DestroyPregnancy is DELETE /health-record/pregnancies/{id}.
func (h *Handlers) DestroyPregnancy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := entryID(c)
	if !ok {
		return notFound(locale)
	}
	if err := h.svc.DeletePregnancy(c, userID, id); errors.Is(err, ErrNotFound) {
		return notFound(locale)
	} else if err != nil {
		return err
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}
