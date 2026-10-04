// Package profile is ProfileController: GET/POST /profile, GET /profile/export and
// DELETE /account. Model serialisation lives in internal/profile/model.
package profile

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/notify"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/profile/model"
	"github.com/ritme/backend-go/internal/profile/store"
)

// allowedFields is ProfileController::store's whitelist (and the 422 `allowed_fields`).
var allowedFields = []string{
	"name", "birthday", "weight", "height", "period_duration",
	"cycle_duration", "last_period_start", "user_goal", "subscription_type",
	"pregnancy_intention", "chronic_conditions",
}

// Handlers is Api\V1\ProfileController.
type Handlers struct {
	svc    *Service
	bmi    Bmi
	clock  clock.Clock
	debug  bool
	logger *slog.Logger
}

// Options wires the handlers.
type Options struct {
	DB       *sql.DB
	Telegram *notify.Telegram // nil → no notices
	// StoragePath is STORAGE_PATH (support-report screenshots are removed with the account; "" = none stored).
	StoragePath string
	Content     MessageContents // nil → the store-backed message_contents lookup
	Clock       clock.Clock     // fallback when the request carries no test clock
	Debug       bool            // APP_DEBUG: 500 bodies carry the error text
	Logger      *slog.Logger
}

// NewHandlers builds the controller.
func NewHandlers(o Options) *Handlers {
	q := store.New(o.DB)
	if o.Content == nil {
		o.Content = StoreMessageContents{Q: q}
	}
	if o.Clock == nil {
		o.Clock = clock.Real{}
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	return &Handlers{
		svc:    &Service{DB: o.DB, Q: q, Telegram: o.Telegram, StoragePath: o.StoragePath, Logger: o.Logger},
		bmi:    Bmi{Content: o.Content},
		clock:  o.Clock,
		debug:  o.Debug,
		logger: o.Logger,
	}
}

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.clock).Now() }

func currentUser(c fiber.Ctx) (*auth.User, error) {
	u := auth.CurrentUser(c)
	if u == nil {
		return nil, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return u, nil
}

func (h *Handlers) loadProfile(c fiber.Ctx, userID uint64) (*model.UserProfile, error) {
	p, err := h.svc.Q.GetProfileByUserID(c.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("profile: load: %w", err)
	}
	return &p, nil
}

// Show is GET /profile: {user (with the loaded profile relation), profile, bmi}.
func (h *Handlers) Show(c fiber.Ctx) error {
	u, err := currentUser(c)
	if err != nil {
		return err
	}
	p, err := h.loadProfile(c, u.ID)
	if err != nil {
		return err
	}
	locale := i18n.ResolveLocale(c, "")
	var bmi any
	if p != nil {
		if bmi, err = h.bmi.ForProfile(c.Context(), p, locale, i18n.LanguagesOf(c).DefaultCode()); err != nil {
			return err
		}
	}
	return httpx.OK(c, jsonx.Obj(
		"user", model.UserWithProfileJSON(u, p),
		"profile", model.ProfileJSON(p),
		"bmi", bmi,
	))
}

func trans(key, locale string, params map[string]string) string {
	return lang.Default().Trans(key, params, locale)
}

func storeRules() validation.Rules {
	return validation.Rules{
		validation.F("name", "nullable|string|max:255"),
		validation.F("birthday", "nullable|date|before:today"),
		validation.F("weight", "nullable|numeric|min:20|max:300"),
		validation.F("height", "nullable|integer|min:50|max:250"),
		validation.F("period_duration", "nullable|integer|min:1|max:15"),
		validation.F("cycle_duration", "nullable|integer|min:15|max:60"),
		validation.F("last_period_start", "nullable|date|before_or_equal:today"),
		validation.F("user_goal", "nullable", validation.In(enums.UserGoalValues()...)),
		validation.F("subscription_type", "nullable", validation.In(enums.SubscriptionTypeValues()...)),
		validation.F("pregnancy_intention", "nullable", validation.In(enums.PregnancyIntentionValues()...)),
		validation.F("chronic_conditions", "nullable", "array"),
		validation.F("chronic_conditions.*", validation.In(enums.ChronicConditionValues()...)),
	}
}

// storeMessages are the controller's custom messages (lang/*/profile.php → errors.*).
func storeMessages(locale string) validation.Option {
	e := func(key string) string { return trans("profile.errors."+key, locale, nil) }
	in := func(key string, values []string) string {
		return trans("profile.errors."+key, locale, map[string]string{"values": strings.Join(values, ", ")})
	}
	return validation.Messages(
		"name.string", e("name_string"),
		"name.max", e("name_max"),
		"birthday.date", e("birthday_date"),
		"birthday.before", e("birthday_before"),
		"weight.numeric", e("weight_numeric"),
		"weight.min", e("weight_min"),
		"weight.max", e("weight_max"),
		"height.integer", e("height_integer"),
		"height.min", e("height_min"),
		"height.max", e("height_max"),
		"period_duration.integer", e("period_duration_integer"),
		"period_duration.min", e("period_duration_min"),
		"period_duration.max", e("period_duration_max"),
		"cycle_duration.integer", e("cycle_duration_integer"),
		"cycle_duration.min", e("cycle_duration_min"),
		"cycle_duration.max", e("cycle_duration_max"),
		"last_period_start.date", e("last_period_start_date"),
		"last_period_start.before_or_equal", e("last_period_start_before_or_equal"),
		"user_goal.in", in("user_goal_in", enums.UserGoalValues()),
		"subscription_type.in", in("subscription_type_in", enums.SubscriptionTypeValues()),
		"pregnancy_intention.in", in("pregnancy_intention_in", enums.PregnancyIntentionValues()),
		"chronic_conditions.array", e("chronic_conditions_array"),
		"chronic_conditions.*.in", in("chronic_conditions_in", enums.ChronicConditionValues()),
	)
}

// Store is POST /profile (create or update, onboarding side effects).
func (h *Handlers) Store(c fiber.Ctx) error {
	u, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.ResolveLocale(c, "")
	now := h.now(c)
	input := validation.Input(c)

	var unknown []string
	for _, k := range input.Keys() {
		if !slices.Contains(allowedFields, k) {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return httpx.Fail(fiber.StatusUnprocessableEntity,
			trans("profile.unknown_fields", locale, map[string]string{"fields": strings.Join(unknown, ", ")}),
			"allowed_fields", allowedFields)
	}

	v := validation.Make(lang.Default(), locale, input, storeRules(), validation.Now(now), storeMessages(locale))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity,
			trans("profile.validation_failed", locale, map[string]string{"first": v.First()}),
			"errors", v.ErrorBag())
	}

	res, err := h.svc.Save(c.Context(), u, input, now)
	if err != nil {
		return h.storeFailure(c, u, locale, err)
	}
	status := res.Profile.CalculationStatus
	if status == "" {
		status = string(enums.CalculationStatusPending)
	}
	return httpx.OK(c, jsonx.Obj(
		"user", model.UserJSON(&res.User),
		"profile", model.ProfileJSON(&res.Profile),
		"calculation_status", status,
	), trans("profile.updated", locale, nil))
}

// storeFailure is the controller's catch blocks: 500 {success:false, message, error}.
func (h *Handlers) storeFailure(c fiber.Ctx, u *auth.User, locale string, err error) error {
	key := "profile.unexpected_error"
	var dbe *DBError
	if errors.As(err, &dbe) {
		key = "profile.db_error"
		h.logger.ErrorContext(c.Context(), "Profile store DB error", slog.String("error", err.Error()), slog.Uint64("user_id", u.ID))
	} else {
		h.logger.ErrorContext(c.Context(), "Profile store error", slog.String("error", err.Error()))
	}
	detail := trans("profile.try_again", locale, nil)
	if h.debug {
		detail = err.Error()
	}
	return httpx.Fail(fiber.StatusInternalServerError, trans(key, locale, nil), "error", detail)
}

// Export is GET /profile/export: every personal record as raw model serialisations.
func (h *Handlers) Export(c fiber.Ctx) error {
	u, err := currentUser(c)
	if err != nil {
		return err
	}
	ctx, q := c.Context(), h.svc.Q
	p, err := h.loadProfile(c, u.ID)
	if err != nil {
		return err
	}
	logs, err := q.ExportDailyHealthLogs(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("profile: export health logs: %w", err)
	}
	cycles, err := q.ListCycleHistories(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("profile: export cycle histories: %w", err)
	}
	var pregnancy any
	pp, err := q.ExportPregnancyProfile(ctx, u.ID)
	switch {
	case err == nil:
		pregnancy = model.Attributes(&pp, model.PregnancyProfileCasts)
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("profile: export pregnancy profile: %w", err)
	}
	symptoms, err := q.ExportPregnancySymptomLogs(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("profile: export symptom logs: %w", err)
	}
	weekly, err := q.ExportPregnancyWeeklyLogs(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("profile: export weekly logs: %w", err)
	}
	movements, err := q.ExportPregnancyFetalMovements(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("profile: export fetal movements: %w", err)
	}
	reminders, err := q.ExportReminders(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("profile: export reminders: %w", err)
	}
	privacy, err := exportPrivacy(ctx, q, u.ID)
	if err != nil {
		return err
	}
	var life *store.UserLifeProfile
	switch lp, err := q.GetLifeProfile(ctx, u.ID); {
	case err == nil:
		life = &lp
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("profile: export life profile: %w", err)
	}

	var createdAt any
	if u.CreatedAt.Valid {
		createdAt = jsonx.ISO8601(u.CreatedAt.Time)
	}
	var name any
	if u.Name.Valid {
		name = u.Name.String
	}
	var mobile any
	if u.Mobile.Valid {
		mobile = u.Mobile.String
	}
	return httpx.OK(c, jsonx.Obj(
		"exported_at", jsonx.ISO8601(h.now(c)),
		"account", jsonx.Obj("id", u.ID, "name", name, "mobile", mobile, "created_at", createdAt),
		"profile", model.ProfileJSON(p),
		"health_logs", model.List(logs, model.DailyHealthLogCasts),
		"cycle_histories", model.List(cycles, model.CycleHistoryCasts),
		"pregnancy", jsonx.Obj(
			"profile", pregnancy,
			"symptom_logs", model.List(symptoms, model.PregnancySymptomLogCasts),
			"weekly_logs", model.List(weekly, model.PregnancyWeeklyLogCasts),
			"fetal_movements", model.List(movements, model.PregnancyFetalMovementCasts),
		),
		"reminders", model.List(reminders, model.ReminderCasts),
		// B-N1-12 (Go only): consents, support reports, notification settings.
		"consents", privacy.consents,
		"support_reports", privacy.reports,
		"notification_settings", privacy.notifications,
		// B-N6-05b (Go only, D-58): the AI usage rows still linked to the user.
		"ai_usage", privacy.aiUsage,
		// B-N2-01 (Go only, D-34): onboarding v2 answers and the life-stage mode.
		"life_profile", LifeProfileExportJSON(life),
	))
}

// DestroyAccount is DELETE /account: delete the user's OAuth tokens and the user in one
// transaction (every user-owned table cascades). D-25: Laravel only revokes the access
// tokens and leaves them (and the refresh tokens) behind; the dead token still answers the
// same 401 `token_revoked`.
func (h *Handlers) DestroyAccount(c fiber.Ctx) error {
	u, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.ResolveLocale(c, "")
	ctx := c.Context()
	if err := h.svc.DeleteAccount(ctx, u.ID); err != nil {
		return err
	}
	// Laravel hard-codes this pair (`$locale === 'fa' ? … : …`); ported as is.
	msg := "Your account and all associated data have been deleted"
	if locale == "fa" {
		msg = "حساب کاربری و همه داده‌های شما حذف شد" //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	}
	return httpx.Message(c, msg)
}
