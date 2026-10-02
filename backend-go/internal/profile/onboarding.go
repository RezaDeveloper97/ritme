package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/profile/model"
	"github.com/ritme/backend-go/internal/profile/store"
)

// Onboarding v2 and the life-stage modes (B-N2-01, bloom/N2; screens nbl_Onb_* and nbl_Me_Mode). Go only:
//
//	GET  /api/v1/onboarding                 the answers so far (+ effective mode, completion)
//	PUT  /api/v1/onboarding/steps/{step}    one step, idempotent (same body → same state)
//	POST /api/v1/onboarding/complete        «ورود به ریتمی» (idempotent: the first completion time is kept)
//	GET  /api/v1/profile/life-stage         effective mode + stored mode + the IVF/IUI and contraception switches
//	PUT  /api/v1/profile/life-stage         partial update of the three
//	GET  /api/v1/profile/life-stage/loss-copy  admin-edited copy of the calm pregnancy exit (B-N2-03)
//
// What the legacy profile already holds stays there (users.name; user_profiles birthday / height / weight / last
// period / lengths / user_goal) and is written through Service.Save, so the cycle side effects (onboarding period
// log, recalculation) are the ones POST /profile runs. The rest lives in user_life_profiles (goose 00014). The
// Pregnancy branch (nbl_Onb_Preg) is the existing POST /pregnancy/onboarding; the phone-screen consents are
// PUT /profile/consents (B-N1-12).

// Onboarding steps, in screen order.
const (
	StepName       = "name"
	StepGender     = "gender"
	StepGoal       = "goal"
	StepCycle      = "cycle"
	StepMenopause  = "menopause"
	StepConditions = "conditions"
	StepHealth     = "health"
)

// OnboardingSteps are the {step} values PUT /onboarding/steps/{step} accepts.
var OnboardingSteps = []string{StepName, StepGender, StepGoal, StepCycle, StepMenopause, StepConditions, StepHealth}

// OT is the onboarding line for key ("messages.saved") in locale (lang/<code>/onboarding.json).
func OT(key, locale string) string {
	return privacyTranslator().Trans("onboarding."+key, nil, locale)
}

func onboardingAttributes(locale string) []string {
	line, ok := privacyTranslator().Get("onboarding.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

// LifeProfileStore is the persistence of the endpoints (profile store; user_id in every query).
type LifeProfileStore interface {
	GetUser(ctx context.Context, id uint64) (store.User, error)
	GetProfileByUserID(ctx context.Context, userID uint64) (store.UserProfile, error)
	GetLifeProfile(ctx context.Context, userID uint64) (store.UserLifeProfile, error)
	UpsertLifeProfile(ctx context.Context, arg store.UpsertLifeProfileParams) error
	PregnancyModeActive(ctx context.Context, userID uint64) (bool, error)
}

// OnboardingHandlers serve the onboarding and life-stage endpoints. Mount behind locale + auth RequireUser.
type OnboardingHandlers struct {
	q     LifeProfileStore
	pq    pregstore.Querier // admin-edited copy (message_contents) of the calm pregnancy exit
	svc   *Service
	clock clock.Clock
}

// NewOnboardingHandlers wires the handlers on d (base = fallback clock; tests pin it per request).
func NewOnboardingHandlers(d *sql.DB, base clock.Clock) *OnboardingHandlers {
	q := store.New(d)
	if base == nil {
		base = clock.Real{}
	}
	return &OnboardingHandlers{q: q, pq: pregstore.New(d), svc: &Service{DB: d, Q: q}, clock: base}
}

func (h *OnboardingHandlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// lifeState is everything the responses are built from.
type lifeState struct {
	user      store.User
	profile   *store.UserProfile
	life      *store.UserLifeProfile
	pregnancy bool
}

func (h *OnboardingHandlers) load(ctx context.Context, userID uint64) (lifeState, error) {
	var st lifeState
	var err error
	if st.user, err = h.q.GetUser(ctx, userID); err != nil {
		return st, fmt.Errorf("profile: onboarding user: %w", err)
	}
	p, err := h.q.GetProfileByUserID(ctx, userID)
	switch {
	case err == nil:
		st.profile = &p
	case !errors.Is(err, sql.ErrNoRows):
		return st, fmt.Errorf("profile: onboarding profile: %w", err)
	}
	l, err := h.q.GetLifeProfile(ctx, userID)
	switch {
	case err == nil:
		st.life = &l
	case !errors.Is(err, sql.ErrNoRows):
		return st, fmt.Errorf("profile: life profile: %w", err)
	}
	if st.pregnancy, err = h.q.PregnancyModeActive(ctx, userID); err != nil {
		return st, fmt.Errorf("profile: pregnancy mode: %w", err)
	}
	return st, nil
}

// lifeRow is the stored row or a fresh one (defaults) to apply a change to.
func (st lifeState) lifeRow(userID uint64) store.UserLifeProfile {
	if st.life != nil {
		return *st.life
	}
	return store.UserLifeProfile{UserID: userID}
}

// mode is the effective life mode (enums.ResolveAccountMode: a male account is a companion, B-N4-03).
func (st lifeState) mode() enums.LifeMode {
	stored, goal := "", ""
	if st.life != nil && st.life.LifeMode.Valid {
		stored = st.life.LifeMode.String
	}
	if st.profile != nil {
		goal = st.profile.UserGoal
	}
	return enums.ResolveAccountMode(st.gender(), stored, st.pregnancy, goal)
}

// gender is the stored gender ("" = not asked yet).
func (st lifeState) gender() string {
	if st.life != nil && st.life.Gender.Valid {
		return st.life.Gender.String
	}
	return ""
}

func nullStrAny(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func nullBool(b sql.NullBool) any {
	if !b.Valid {
		return nil
	}
	return b.Bool
}

// jsonList decodes a JSON list column (NULL → nil = not answered, [] = «هیچ‌کدام»).
func jsonList(v db.NullRawJSON) any {
	if !v.Valid {
		return nil
	}
	out := []string{}
	if err := json.Unmarshal(v.V, &out); err != nil {
		return nil
	}
	return out
}

func dateOrNil(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

// LifeStageJSON is the life-stage block: effective `mode`, the `stored_mode` (null = legacy derivation) and the
// two switches.
func LifeStageJSON(st lifeState) *jsonx.OrderedMap {
	var stored any
	ivf, contra := false, false
	if st.life != nil {
		stored = nullStrAny(st.life.LifeMode)
		ivf, contra = st.life.IvfIui, st.life.TrackContraception
	}
	return jsonx.Obj(
		"mode", string(st.mode()),
		"stored_mode", stored,
		"ivf_iui", ivf,
		"track_contraception", contra,
	)
}

// OnboardingJSON is the onboarding state (GET /onboarding and every step's response).
func OnboardingJSON(st lifeState) *jsonx.OrderedMap {
	var name any
	if st.user.Name.Valid {
		name = st.user.Name.String
	}
	l := st.lifeRow(st.user.ID)
	var lastPeriod, periodLen, cycleLen, birthday, height, weight any
	if p := st.profile; p != nil {
		lastPeriod, birthday = dateOrNil(p.LastPeriodStart), dateOrNil(p.Birthday)
		if p.PeriodDuration.Valid {
			periodLen = int(p.PeriodDuration.Int16)
		}
		if p.CycleDuration.Valid {
			cycleLen = int(p.CycleDuration.Int16)
		}
		if v, ok := model.ProfileHeight(p); ok {
			height = v
		}
		if v, ok := model.ProfileWeight(p); ok {
			weight = v
		}
	}
	var started, completed any
	if l.OnboardingStartedAt.Valid {
		started = jsonx.ISO8601(l.OnboardingStartedAt.Time)
	}
	if l.OnboardingCompletedAt.Valid {
		completed = jsonx.ISO8601(l.OnboardingCompletedAt.Time)
	}
	return jsonx.Obj(
		"name", name,
		"gender", nullStrAny(l.Gender),
		"goal", nullStrAny(l.LifeMode),
		"life_stage", LifeStageJSON(st),
		"cycle", jsonx.Obj(
			"last_period_start", lastPeriod,
			"period_duration", periodLen,
			"cycle_duration", cycleLen,
		),
		"menopause", jsonx.Obj(
			"stage", nullStrAny(l.MenopauseStage),
			"last_period", dateOrNil(l.MenopauseLastPeriod),
			"surgical", nullBool(l.MenopauseSurgical),
			"hrt", nullBool(l.MenopauseHrt),
		),
		"conditions", jsonx.Obj(
			"chronic_illnesses", jsonList(l.ChronicIllnesses),
			"gyn_conditions", jsonList(l.GynConditions),
			"medications", jsonList(l.Medications),
		),
		"health", jsonx.Obj(
			"birthday", birthday,
			"height", height,
			"weight", weight,
		),
		"started_at", started,
		"completed_at", completed,
		"completed", l.OnboardingCompletedAt.Valid,
	)
}

// LifeProfileExportJSON is the user_life_profiles row in GET /profile/export (null when the user has none).
func LifeProfileExportJSON(l *store.UserLifeProfile) any {
	if l == nil {
		return nil
	}
	ts := func(t sql.NullTime) any {
		if !t.Valid {
			return nil
		}
		return jsonx.ISO8601(t.Time)
	}
	return jsonx.Obj(
		"gender", nullStrAny(l.Gender),
		"life_mode", nullStrAny(l.LifeMode),
		"ivf_iui", l.IvfIui,
		"track_contraception", l.TrackContraception,
		"chronic_illnesses", jsonList(l.ChronicIllnesses),
		"gyn_conditions", jsonList(l.GynConditions),
		"medications", jsonList(l.Medications),
		"menopause_stage", nullStrAny(l.MenopauseStage),
		"menopause_last_period", dateOrNil(l.MenopauseLastPeriod),
		"menopause_surgical", nullBool(l.MenopauseSurgical),
		"menopause_hrt", nullBool(l.MenopauseHrt),
		"onboarding_started_at", ts(l.OnboardingStartedAt),
		"onboarding_completed_at", ts(l.OnboardingCompletedAt),
		"created_at", ts(l.CreatedAt),
		"updated_at", ts(l.UpdatedAt),
	)
}

func (h *OnboardingHandlers) save(ctx context.Context, l store.UserLifeProfile, now time.Time) error {
	at := sql.NullTime{Time: now, Valid: true}
	if err := h.q.UpsertLifeProfile(ctx, store.UpsertLifeProfileParams{
		UserID: l.UserID, Gender: l.Gender, LifeMode: l.LifeMode, IvfIui: l.IvfIui,
		TrackContraception: l.TrackContraception, ChronicIllnesses: l.ChronicIllnesses,
		GynConditions: l.GynConditions, Medications: l.Medications, MenopauseStage: l.MenopauseStage,
		MenopauseLastPeriod: l.MenopauseLastPeriod, MenopauseSurgical: l.MenopauseSurgical,
		MenopauseHrt: l.MenopauseHrt, OnboardingStartedAt: l.OnboardingStartedAt,
		OnboardingCompletedAt: l.OnboardingCompletedAt, Now: at,
	}); err != nil {
		return fmt.Errorf("profile: save life profile: %w", err)
	}
	return nil
}

// markStarted stamps onboarding_started_at on the first step. A user who had already finished the legacy
// onboarding (name + profile, auth's profile_completed) is stamped completed too, so replaying a step never
// sends them back to onboarding.
func markStarted(l *store.UserLifeProfile, st lifeState, now time.Time) {
	if l.OnboardingStartedAt.Valid {
		return
	}
	l.OnboardingStartedAt = sql.NullTime{Time: now, Valid: true}
	named := st.user.Name.Valid && strings.TrimSpace(st.user.Name.String) != ""
	if named && st.profile != nil && !l.OnboardingCompletedAt.Valid {
		l.OnboardingCompletedAt = l.OnboardingStartedAt
	}
}

// syncLegacyGoal keeps user_profiles.user_goal / pregnancy_intention in step with a chosen mode (only ttc is TTC;
// pregnancy → intention pregnant; any other mode drops a trying / pregnant intention), through Service.Save.
func (h *OnboardingHandlers) syncLegacyGoal(ctx context.Context, u *auth.User, st lifeState, mode enums.LifeMode, now time.Time) error {
	input := phpval.NewMap()
	input.Set("user_goal", string(mode.LegacyUserGoal()))
	var current string
	if st.profile != nil && st.profile.PregnancyIntention.Valid {
		current = st.profile.PregnancyIntention.String
	}
	switch mode {
	case enums.LifeModeTTC:
		input.Set("pregnancy_intention", string(enums.PregnancyIntentionTrying))
	case enums.LifeModePregnancy:
		input.Set("pregnancy_intention", string(enums.PregnancyIntentionPregnant))
	default:
		if current == string(enums.PregnancyIntentionTrying) || current == string(enums.PregnancyIntentionPregnant) {
			input.Set("pregnancy_intention", nil)
		}
	}
	_, err := h.svc.Save(ctx, u, input, now)
	return err
}

// Show is GET /onboarding.
func (h *OnboardingHandlers) Show(c fiber.Ctx) error {
	id, err := privacyUserID(c)
	if err != nil {
		return err
	}
	st, err := h.load(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, OnboardingJSON(st))
}

func stepRules(step string) validation.Rules {
	F := validation.F
	switch step {
	case StepName:
		return validation.Rules{F("name", "required", "string", "max:255")}
	case StepGender:
		return validation.Rules{F("gender", "required", validation.In(enums.GenderValues()...))}
	case StepGoal:
		return validation.Rules{F("goal", "required", validation.In(enums.OnboardingGoalValues()...))}
	case StepCycle:
		return validation.Rules{
			F("last_period_start", "nullable", "date", "before_or_equal:today"),
			F("period_duration", "nullable", "integer", fmt.Sprintf("min:%d", minManualPeriod), fmt.Sprintf("max:%d", maxManualPeriod)),
			F("cycle_duration", "nullable", "integer", fmt.Sprintf("min:%d", minManualCycle), fmt.Sprintf("max:%d", maxManualCycle)),
		}
	case StepMenopause:
		return validation.Rules{
			F("stage", "required", validation.In(enums.MenopauseStageValues()...)),
			F("last_period", "nullable", "date", "before_or_equal:today"),
			F("surgical", "nullable", "boolean"),
			F("hrt", "nullable", "boolean"),
		}
	case StepConditions:
		return validation.Rules{
			F("chronic_illnesses", "nullable", "array"),
			F("chronic_illnesses.*", validation.In(enums.ChronicIllnessValues()...)),
			F("gyn_conditions", "nullable", "array"),
			F("gyn_conditions.*", validation.In(enums.GynConditionValues()...)),
			F("medications", "nullable", "array"),
			F("medications.*", validation.In(enums.MedicationValues()...)),
		}
	case StepHealth:
		return validation.Rules{
			F("birthday", "nullable", "date", "before:today"),
			F("height", "nullable", "integer", "min:50", "max:250"),
			F("weight", "nullable", "numeric", "min:20", "max:300"),
		}
	}
	return nil
}

// pickList is a validated list in canonical (screen) order without duplicates; nil/absent → NULL (skipped).
func pickList(body phpval.Map, key string, allowed []string) db.NullRawJSON {
	v, ok := phpval.Get(body, key)
	if !ok || v == nil {
		return db.NullRawJSON{}
	}
	_, vals := phpval.Entries(v)
	chosen := make([]string, 0, len(vals))
	for _, a := range allowed {
		for _, x := range vals {
			if phpval.ToString(x) == a {
				chosen = append(chosen, a)
				break
			}
		}
	}
	b, _ := json.Marshal(chosen)
	return db.NullRawJSON{V: b, Valid: true}
}

func pickBool(body phpval.Map, key string) sql.NullBool {
	v, ok := phpval.Get(body, key)
	if !ok || v == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: phpval.Truthy(v), Valid: true}
}

func pickString(body phpval.Map, key string) (string, bool) {
	v, ok := phpval.Get(body, key)
	if !ok || v == nil {
		return "", false
	}
	return phpval.ToString(v), true
}

// profileInput is the present, non-null keys of body (absent / null keep the stored legacy value).
func profileInput(body phpval.Map, keys ...string) phpval.Map {
	in := phpval.NewMap()
	for _, k := range keys {
		if v, ok := phpval.Get(body, k); ok && v != nil {
			in.Set(k, v)
		}
	}
	return in
}

// UpdateStep is PUT /onboarding/steps/{step}. Every step replaces its own answers, so repeating a request
// leaves the same state:
//
//	name        {name}                                               → users.name
//	gender      {gender: female|male}
//	goal        {goal: cycle|ttc|pregnancy|menopause}                → life mode (+ user_goal / intention in step)
//	cycle       {last_period_start?, period_duration?, cycle_duration?} → user_profiles (null/absent = keep;
//	            «دقیق یادم نیست» = no last_period_start)
//	menopause   {stage, last_period?, surgical?, hrt?}
//	conditions  {chronic_illnesses?, gyn_conditions?, medications?}   ([] = «هیچ‌کدام», null/absent = skipped)
//	health      {birthday?, height?, weight?}                        → user_profiles (null/absent = keep)
func (h *OnboardingHandlers) UpdateStep(c fiber.Ctx) error {
	u := auth.CurrentUser(c)
	if u == nil {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	step := c.Params("step")
	if !slices.Contains(OnboardingSteps, step) {
		return httpx.NotFound()
	}
	locale := i18n.Locale(c)
	now := h.now(c)
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, stepRules(step),
		validation.Now(now), validation.Attributes(onboardingAttributes(locale)...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, OT("messages.validation_failed", locale), "errors", v.ErrorBag())
	}

	ctx := c.Context()
	st, err := h.load(ctx, u.ID)
	if err != nil {
		return err
	}
	l := st.lifeRow(u.ID)
	markStarted(&l, st, now)

	switch step {
	case StepName:
		if err := h.svc.updateName(ctx, u, profileInput(body, "name"), now); err != nil {
			return err
		}
	case StepGender:
		g, _ := pickString(body, "gender")
		l.Gender = sql.NullString{String: g, Valid: true}
	case StepGoal:
		g, _ := pickString(body, "goal")
		mode := enums.LifeMode(g)
		l.LifeMode = sql.NullString{String: g, Valid: true}
		if err := h.syncLegacyGoal(ctx, u, st, mode, now); err != nil {
			return err
		}
	case StepCycle:
		if in := profileInput(body, "last_period_start", "period_duration", "cycle_duration"); in.Len() > 0 {
			if _, err := h.svc.Save(ctx, u, in, now); err != nil {
				return err
			}
		}
	case StepMenopause:
		stage, _ := pickString(body, "stage")
		l.MenopauseStage = sql.NullString{String: stage, Valid: true}
		l.MenopauseLastPeriod = civildate.NullDate{}
		if s, ok := pickString(body, "last_period"); ok {
			if d, err := civildate.ParseLenient(s, now, civildate.Tehran); err == nil {
				l.MenopauseLastPeriod = civildate.NullDate{Date: civildate.InTehran(d), Valid: true}
			}
		}
		l.MenopauseSurgical, l.MenopauseHrt = pickBool(body, "surgical"), pickBool(body, "hrt")
	case StepConditions:
		l.ChronicIllnesses = pickList(body, "chronic_illnesses", enums.ChronicIllnessValues())
		l.GynConditions = pickList(body, "gyn_conditions", enums.GynConditionValues())
		l.Medications = pickList(body, "medications", enums.MedicationValues())
	case StepHealth:
		if in := profileInput(body, "birthday", "height", "weight"); in.Len() > 0 {
			if _, err := h.svc.Save(ctx, u, in, now); err != nil {
				return err
			}
		}
	}
	if err := h.save(ctx, l, now); err != nil {
		return err
	}
	if st, err = h.load(ctx, u.ID); err != nil {
		return err
	}
	return httpx.OK(c, OnboardingJSON(st), OT("messages.saved", locale))
}

// Complete is POST /onboarding/complete («ورود به ریتمی»). Needs a name and a gender, and for a woman the goal;
// otherwise a controller 422 names the missing steps. Creates the profile row if no step did (auth's
// profile_completed needs one) and stamps onboarding_completed_at once.
func (h *OnboardingHandlers) Complete(c fiber.Ctx) error {
	u := auth.CurrentUser(c)
	if u == nil {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	now := h.now(c)
	ctx := c.Context()
	st, err := h.load(ctx, u.ID)
	if err != nil {
		return err
	}
	l := st.lifeRow(u.ID)

	missing := httpx.NewValidationError()
	if !st.user.Name.Valid || strings.TrimSpace(st.user.Name.String) == "" {
		missing.Add(StepName, OT("messages.step_missing.name", locale))
	}
	if !l.Gender.Valid {
		missing.Add(StepGender, OT("messages.step_missing.gender", locale))
	} else if l.Gender.String == string(enums.GenderFemale) && !l.LifeMode.Valid {
		missing.Add(StepGoal, OT("messages.step_missing.goal", locale))
	}
	if !missing.Empty() {
		errs := jsonx.Obj()
		for _, k := range []string{StepName, StepGender, StepGoal} {
			if m := missing.Messages(k); len(m) > 0 {
				errs.Set(k, m)
			}
		}
		return httpx.Fail(fiber.StatusUnprocessableEntity, OT("messages.incomplete", locale), "errors", errs)
	}

	if st.profile == nil {
		if _, err := h.svc.Save(ctx, u, phpval.NewMap(), now); err != nil {
			return err
		}
	}
	if !l.OnboardingStartedAt.Valid {
		l.OnboardingStartedAt = sql.NullTime{Time: now, Valid: true}
	}
	if !l.OnboardingCompletedAt.Valid {
		l.OnboardingCompletedAt = sql.NullTime{Time: now, Valid: true}
	}
	if err := h.save(ctx, l, now); err != nil {
		return err
	}
	if st, err = h.load(ctx, u.ID); err != nil {
		return err
	}
	return httpx.OK(c, OnboardingJSON(st), OT("messages.completed", locale))
}

// ShowLifeStage is GET /profile/life-stage.
func (h *OnboardingHandlers) ShowLifeStage(c fiber.Ctx) error {
	id, err := privacyUserID(c)
	if err != nil {
		return err
	}
	st, err := h.load(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, LifeStageJSON(st))
}

// UpdateLifeStage is PUT /profile/life-stage, a partial update {mode?, ivf_iui?, track_contraception?}. A mode
// keeps user_goal / pregnancy_intention in step (syncLegacyGoal); every mode keeps the user's data. Pregnancy
// itself is switched on and off by the pregnancy endpoints (/pregnancy/onboarding, /pregnancy/deactivate): while
// a pregnancy profile is active the effective mode stays pregnancy.
func (h *OnboardingHandlers) UpdateLifeStage(c fiber.Ctx) error {
	u := auth.CurrentUser(c)
	if u == nil {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	now := h.now(c)
	body := validation.Input(c)
	F := validation.F
	v := validation.Make(lang.Default(), locale, body, validation.Rules{
		F("mode", "sometimes", "required", validation.In(enums.LifeModeValues()...)),
		F("ivf_iui", "sometimes", "boolean"),
		F("track_contraception", "sometimes", "boolean"),
	}, validation.Now(now), validation.Attributes(onboardingAttributes(locale)...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, OT("messages.validation_failed", locale), "errors", v.ErrorBag())
	}

	ctx := c.Context()
	st, err := h.load(ctx, u.ID)
	if err != nil {
		return err
	}
	l := st.lifeRow(u.ID)
	if m, ok := pickString(body, "mode"); ok && st.mode() == enums.LifeModeCompanion {
		// B-N4-03: a companion (male) account has no life stage to switch (no cycle, no pregnancy of his own).
		return httpx.Fail(fiber.StatusUnprocessableEntity, OT("messages.validation_failed", locale),
			"errors", jsonx.Obj("mode", []string{OT("messages.companion_no_mode", locale)}))
	} else if ok {
		l.LifeMode = sql.NullString{String: m, Valid: true}
		if err := h.syncLegacyGoal(ctx, u, st, enums.LifeMode(m), now); err != nil {
			return err
		}
	}
	if b := pickBool(body, "ivf_iui"); b.Valid {
		l.IvfIui = b.Bool
	}
	if b := pickBool(body, "track_contraception"); b.Valid {
		l.TrackContraception = b.Bool
	}
	if err := h.save(ctx, l, now); err != nil {
		return err
	}
	if st, err = h.load(ctx, u.ID); err != nil {
		return err
	}
	return httpx.OK(c, LifeStageJSON(st), OT("messages.life_stage_saved", locale))
}

// SwitchLifeMode stores mode as u's life-stage mode the way PUT /profile/life-stage {mode} does (user_goal and
// pregnancy_intention kept in step), for another domain acting on the user's behalf — the pregnancy loss path
// (CB-LOSS-01) leaves pregnancy for cycle and applies the chosen next step. A companion account is left alone.
func (h *OnboardingHandlers) SwitchLifeMode(ctx context.Context, u *auth.User, mode enums.LifeMode, now time.Time) error {
	st, err := h.load(ctx, u.ID)
	if err != nil {
		return err
	}
	if st.mode() == enums.LifeModeCompanion {
		return nil
	}
	l := st.lifeRow(u.ID)
	l.LifeMode = sql.NullString{String: string(mode), Valid: true}
	if err := h.syncLegacyGoal(ctx, u, st, mode, now); err != nil {
		return err
	}
	return h.save(ctx, l, now)
}
