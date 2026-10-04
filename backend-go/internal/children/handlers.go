package children

import (
	"context"
	"errors"
	"strconv"
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
	"github.com/ritme/backend-go/seeds/who"
)

// Error codes of the non-validation failures.
const (
	ErrorCodeNotFound            = "child_not_found"
	ErrorCodeReadOnly            = "child_read_only"
	ErrorCodeTooMany             = "children_limit"
	ErrorCodeMeasurementNotFound = "measurement_not_found"
)

// Handlers are the /api/v1/children actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

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

func loc(c fiber.Ctx) Loc {
	return Loc{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
}

// fail maps the domain errors: unknown or foreign child → uniform 404, shared child write → 403.
func fail(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale), "error_code", ErrorCodeNotFound)
	case errors.Is(err, ErrReadOnly):
		return httpx.Fail(fiber.StatusForbidden, T("messages.read_only", locale), "error_code", ErrorCodeReadOnly)
	case errors.Is(err, ErrTooMany):
		return httpx.Fail(fiber.StatusUnprocessableEntity,
			Tp("messages.too_many", map[string]string{"max": num(MaxChildren, locale)}, locale), "error_code", ErrorCodeTooMany)
	case errors.Is(err, ErrMeasurementNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.measurement_not_found", locale), "error_code", ErrorCodeMeasurementNotFound)
	case errors.Is(err, ErrDateTaken):
		return fieldFail(locale, "measured_on", "measured_on_taken")
	case errors.Is(err, ErrUnknownBand):
		return failValidation(locale, jsonx.Obj("month", []string{
			validationLine("validation.in", "month", locale),
		}))
	}
	return err
}

func validationLine(key, attr, locale string) string {
	return lang.Default().Trans(key, map[string]string{"attribute": attrName(attr, locale)}, locale)
}

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

// access resolves :id for reading (edit = for writing).
func (h *Handlers) access(c fiber.Ctx, edit bool) (Access, error) {
	userID, err := h.user(c)
	if err != nil {
		return Access{}, err
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return Access{}, fail(ErrNotFound, locale)
	}
	var a Access
	if edit {
		a, err = h.svc.Editable(c, userID, id, h.now(c))
	} else {
		a, err = h.svc.Access(c, userID, id, h.now(c))
	}
	if err != nil {
		return Access{}, fail(err, locale)
	}
	return a, nil
}

func (h *Handlers) ownerName(ctx context.Context, a Access) (string, error) {
	if a.Role != RoleShared {
		return "", nil
	}
	return h.svc.OwnerName(ctx, a.Child.OwnerID)
}

// summary is a list card: profile, next vaccine and vaccine counts, growth verdict.
func (h *Handlers) summary(ctx context.Context, a Access, today civildate.Date, l Loc) (*jsonx.OrderedMap, error) {
	name, err := h.ownerName(ctx, a)
	if err != nil {
		return nil, err
	}
	out := ChildJSON(a, name, today, l.Locale)
	sch, err := h.svc.Schedule(ctx, a.Child, today)
	if err != nil {
		return nil, err
	}
	ms, err := h.svc.Measurements(ctx, a.Child.ID)
	if err != nil {
		return nil, err
	}
	out.Set("vaccines", ScheduleSummaryJSON(sch, today, l))
	out.Set("growth", VerdictJSON(Verdict(a.Child, Points(a.Child, ms)), l.Locale))
	return out, nil
}

// Index is GET /children: the user's children (youngest first) and the ones shared with them.
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	today, l := civildate.InTehran(now), loc(c)
	list, err := h.svc.List(c, userID, now)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(list))
	owned := 0
	for _, a := range list {
		if a.Role == RoleOwner {
			owned++
		}
		s, err := h.summary(c, a, today, l)
		if err != nil {
			return err
		}
		items = append(items, s)
	}
	return httpx.OK(c, jsonx.Obj(
		"children", items,
		"count", len(items),
		"owned_count", owned,
		"max_children", MaxChildren,
		"can_add", owned < MaxChildren,
		"sharing_note", T("notes.sharing", l.Locale),
	))
}

// Store is POST /children {name, birth_date, sex?, birth_weight_kg?, birth_length_cm?, birth_head_cm?, delivery_type?}.
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now, l := h.now(c), loc(c)
	in, err := validateChild(validation.Input(c), nil, l.Locale, now)
	if err != nil {
		return err
	}
	id, err := h.svc.Create(c, userID, in, now)
	if err != nil {
		return fail(err, l.Locale)
	}
	a, err := h.svc.Access(c, userID, id, now)
	if err != nil {
		return err
	}
	out, err := h.summary(c, a, civildate.InTehran(now), l)
	if err != nil {
		return err
	}
	return httpx.Created(c, out, T("messages.created", l.Locale))
}

// Show is GET /children/{id}: the child home (nbl_v16_ChildHome) — profile, latest measurement placed on the WHO
// standard, growth verdict, vaccines with the next visit, the milestone band summary, «این هفته …», learn teaser.
// today (feeds, sleep, diapers) belongs to the baby logs (B-N5-03) and is null here.
func (h *Handlers) Show(c fiber.Ctx) error {
	a, err := h.access(c, false)
	if err != nil {
		return err
	}
	today, l := civildate.InTehran(h.now(c)), loc(c)
	out, err := h.summary(c, a, today, l)
	if err != nil {
		return err
	}
	ms, err := h.svc.Measurements(c, a.Child.ID)
	if err != nil {
		return err
	}
	var latest any
	if pts := Points(a.Child, ms); len(pts) > 0 {
		latest = PointJSON(a.Child, pts[0], l.Locale)
	}
	age := AgeOn(a.Child.BirthDate, today)
	band, _, err := h.svc.Band(c, a.Child.ID, age.Months, nil)
	if err != nil {
		return err
	}
	note, err := h.svc.AgeNote(c, age.Months)
	if err != nil {
		return err
	}
	var thisWeek any
	if note != nil {
		thisWeek = jsonx.Obj("weeks", age.Weeks, "months", age.Months, "body", l.text(note.Body))
	}
	tips, err := h.svc.LearnTips(c, age.Months, "")
	if err != nil {
		return err
	}
	var featured any
	if len(tips) > 0 {
		featured = LearnTipJSON(tips[0], l)
	}
	out.Set("latest", latest)
	out.Set("milestones", MilestoneSummaryJSON(band, l.Locale))
	out.Set("this_week", thisWeek)
	out.Set("learn", jsonx.Obj("count", len(tips), "featured", featured))
	out.Set("today", nil)
	return httpx.OK(c, out)
}

// Update is PUT /children/{id}: a partial profile update (owner only).
func (h *Handlers) Update(c fiber.Ctx) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	now, l := h.now(c), loc(c)
	ch := a.Child
	base := ChildInput{Name: ch.Name, BirthDate: ch.BirthDate, Sex: ch.Sex, BirthWeightKg: ch.BirthWeightKg,
		BirthLengthCm: ch.BirthLengthCm, BirthHeadCm: ch.BirthHeadCm, DeliveryType: ch.DeliveryType}
	in, err := validateChild(validation.Input(c), &base, l.Locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Update(c, a, in, now); err != nil {
		return err
	}
	if a, err = h.svc.Access(c, ch.OwnerID, ch.ID, now); err != nil {
		return err
	}
	out, err := h.summary(c, a, civildate.InTehran(now), l)
	if err != nil {
		return err
	}
	return httpx.OK(c, out, T("messages.updated", l.Locale))
}

// Destroy is DELETE /children/{id} (owner only): the child and everything recorded for it.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c, a); err != nil {
		return err
	}
	return httpx.Message(c, T("messages.deleted", i18n.Locale(c)))
}

// pointsJSON is the measurement history (newest first, birth last) with the verdict.
func pointsJSON(a Access, pts []Point, locale string) *jsonx.OrderedMap {
	list := make([]*jsonx.OrderedMap, 0, len(pts))
	for _, p := range pts {
		list = append(list, PointJSON(a.Child, p, locale))
	}
	return jsonx.Obj(
		"measurements", list,
		"verdict", VerdictJSON(Verdict(a.Child, pts), locale),
		"disclaimer", T("growth.disclaimer", locale),
		"source", who.Source,
	)
}

// Measurements is GET /children/{id}/measurements.
func (h *Handlers) Measurements(c fiber.Ctx) error {
	a, err := h.access(c, false)
	if err != nil {
		return err
	}
	ms, err := h.svc.Measurements(c, a.Child.ID)
	if err != nil {
		return err
	}
	return httpx.OK(c, pointsJSON(a, Points(a.Child, ms), i18n.Locale(c)))
}

// StoreMeasurement is POST /children/{id}/measurements {measured_on?, weight_kg?, length_cm?, head_cm?}: the day's
// measurement (same day again = merged), answered with the row placed on the standard.
func (h *Handlers) StoreMeasurement(c fiber.Ctx) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := validateMeasurement(validation.Input(c), a.Child.BirthDate, nil, locale, now)
	if err != nil {
		return err
	}
	m, err := h.svc.SaveMeasurement(c, a.Child.ID, in, now)
	if err != nil {
		return err
	}
	p := pointOf(m.ID, m.MeasuredOn, m.WeightKg, m.LengthCm, m.HeadCm)
	return httpx.Created(c, PointJSON(a.Child, p, locale), T("messages.measurement_saved", locale))
}

// UpdateMeasurement is PUT /children/{id}/measurements/{mid}: a partial update.
func (h *Handlers) UpdateMeasurement(c fiber.Ctx) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	mid, ok := idParam(c, "mid")
	if !ok {
		return fail(ErrMeasurementNotFound, locale)
	}
	m, err := h.svc.Measurement(c, a.Child.ID, mid)
	if err != nil {
		return fail(err, locale)
	}
	base := MeasurementInput{MeasuredOn: m.MeasuredOn, WeightKg: m.WeightKg, LengthCm: m.LengthCm, HeadCm: m.HeadCm}
	in, err := validateMeasurement(validation.Input(c), a.Child.BirthDate, &base, locale, now)
	if err != nil {
		return err
	}
	if m, err = h.svc.UpdateMeasurement(c, m, in, now); err != nil {
		return fail(err, locale)
	}
	p := pointOf(m.ID, m.MeasuredOn, m.WeightKg, m.LengthCm, m.HeadCm)
	return httpx.OK(c, PointJSON(a.Child, p, locale), T("messages.measurement_saved", locale))
}

// DestroyMeasurement is DELETE /children/{id}/measurements/{mid}.
func (h *Handlers) DestroyMeasurement(c fiber.Ctx) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	mid, ok := idParam(c, "mid")
	if !ok {
		return fail(ErrMeasurementNotFound, locale)
	}
	if err := h.svc.DeleteMeasurement(c, a.Child.ID, mid); err != nil {
		return fail(err, locale)
	}
	return httpx.Message(c, T("messages.measurement_deleted", locale))
}

// Growth is GET /children/{id}/growth?indicator=weight|length|head: the WHO chart series (nbl_v16_Growth).
func (h *Handlers) Growth(c fiber.Ctx) error {
	a, err := h.access(c, false)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	ind, err := validateIndicator(validation.Query(c), locale, now)
	if err != nil {
		return err
	}
	ms, err := h.svc.Measurements(c, a.Child.ID)
	if err != nil {
		return err
	}
	return httpx.OK(c, SeriesJSON(a.Child, ind, Points(a.Child, ms), civildate.InTehran(now), locale))
}

// vaccinesJSON is the GET /children/{id}/vaccines payload.
func vaccinesJSON(s Schedule, today civildate.Date, l Loc) *jsonx.OrderedMap {
	visits := make([]*jsonx.OrderedMap, 0, len(s.Visits))
	for _, v := range s.Visits {
		visits = append(visits, VisitJSON(v, today, l, true))
	}
	return jsonx.Obj(
		"summary", ScheduleSummaryJSON(s, today, l),
		"visits", visits,
		"reminders", jsonx.Obj("days_before", RemindDaysBefore, "label", T("notes.reminders", l.Locale)),
		"note", T("notes.vaccines", l.Locale),
	)
}

// Vaccines is GET /children/{id}/vaccines: the national schedule dated from the birth with each dose's status.
func (h *Handlers) Vaccines(c fiber.Ctx) error {
	a, err := h.access(c, false)
	if err != nil {
		return err
	}
	today, l := civildate.InTehran(h.now(c)), loc(c)
	s, err := h.svc.Schedule(c, a.Child, today)
	if err != nil {
		return err
	}
	return httpx.OK(c, vaccinesJSON(s, today, l))
}

// MarkDose is PUT /children/{id}/vaccines/{code} {given_on?, note?}: one dose given (default today).
func (h *Handlers) MarkDose(c fiber.Ctx) error {
	return h.mark(c, func(s Schedule) []string {
		code := c.Params("code")
		for _, v := range s.Visits {
			for _, d := range v.Doses {
				if d.Code == code {
					return []string{code}
				}
			}
		}
		return nil
	}, "code", "unknown_dose", "messages.dose_saved")
}

// MarkVisit is POST /children/{id}/vaccines/visits/{visit} {given_on?, note?}: every dose of a visit given
// («ثبت نوبت · تزریق شد»).
func (h *Handlers) MarkVisit(c fiber.Ctx) error {
	return h.mark(c, func(s Schedule) []string {
		visit := c.Params("visit")
		for _, v := range s.Visits {
			if v.Code == visit {
				codes := make([]string, 0, len(v.Doses))
				for _, d := range v.Doses {
					codes = append(codes, d.Code)
				}
				return codes
			}
		}
		return nil
	}, "visit", "unknown_visit", "messages.visit_saved")
}

func (h *Handlers) mark(c fiber.Ctx, pick func(Schedule) []string, field, unknownKey, msgKey string) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	now, l := h.now(c), loc(c)
	today := civildate.InTehran(now)
	s, err := h.svc.Schedule(c, a.Child, today)
	if err != nil {
		return err
	}
	codes := pick(s)
	if len(codes) == 0 {
		return fieldFail(l.Locale, field, unknownKey)
	}
	day, note, err := validateDose(validation.Input(c), a.Child.BirthDate, l.Locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.MarkDoses(c, a.Child.ID, codes, day, note, now); err != nil {
		return err
	}
	if s, err = h.svc.Schedule(c, a.Child, today); err != nil {
		return err
	}
	return httpx.OK(c, vaccinesJSON(s, today, l), T(msgKey, l.Locale))
}

// UnmarkDose is DELETE /children/{id}/vaccines/{code}: the dose record removed (marked by mistake).
func (h *Handlers) UnmarkDose(c fiber.Ctx) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	today, l := civildate.InTehran(h.now(c)), loc(c)
	if err := h.svc.UnmarkDose(c, a.Child.ID, c.Params("code")); err != nil {
		return err
	}
	s, err := h.svc.Schedule(c, a.Child, today)
	if err != nil {
		return err
	}
	return httpx.OK(c, vaccinesJSON(s, today, l), T("messages.dose_removed", l.Locale))
}

// milestonesJSON is the GET /children/{id}/milestones payload.
func milestonesJSON(b MilestoneBand, bs []int, current int, l Loc) *jsonx.OrderedMap {
	tabs := make([]*jsonx.OrderedMap, 0, len(bs))
	for _, m := range bs {
		tabs = append(tabs, jsonx.Obj("months", m, "label", VisitLabel(m, l.Locale), "current", m == current))
	}
	var band any
	if b.Months >= 0 {
		band = MilestoneBandJSON(b, l)
	}
	return jsonx.Obj("bands", tabs, "band", band, "intro", T("notes.milestones", l.Locale))
}

// Milestones is GET /children/{id}/milestones?month=: the age band (default the child's current one) with checks,
// activities and when to mention it to the doctor.
func (h *Handlers) Milestones(c fiber.Ctx) error {
	a, err := h.access(c, false)
	if err != nil {
		return err
	}
	now, l := h.now(c), loc(c)
	month, err := validateMonth(validation.Query(c), l.Locale, now)
	if err != nil {
		return err
	}
	age := AgeOn(a.Child.BirthDate, civildate.InTehran(now))
	b, bs, err := h.svc.Band(c, a.Child.ID, age.Months, month)
	if err != nil {
		return fail(err, l.Locale)
	}
	return httpx.OK(c, milestonesJSON(b, bs, bandFor(bs, age.Months), l))
}

// Check is PUT /children/{id}/milestones/{code} {checked, checked_on?}: answered with the item's band.
func (h *Handlers) Check(c fiber.Ctx) error {
	a, err := h.access(c, true)
	if err != nil {
		return err
	}
	now, l := h.now(c), loc(c)
	code := c.Params("code")
	known, err := h.svc.MilestoneKnown(c, code)
	if err != nil {
		return err
	}
	if !known {
		return fieldFail(l.Locale, "code", "unknown_milestone")
	}
	checked, day, err := validateCheck(validation.Input(c), a.Child.BirthDate, l.Locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.SetCheck(c, a.Child.ID, code, checked, day, now); err != nil {
		return err
	}
	items, err := h.svc.Catalog(c, GroupMilestones)
	if err != nil {
		return err
	}
	month := 0
	for _, it := range items {
		if it.Code == code {
			month = it.AgeMonths
		}
	}
	age := AgeOn(a.Child.BirthDate, civildate.InTehran(now))
	b, bs, err := h.svc.Band(c, a.Child.ID, age.Months, &month)
	if err != nil {
		return fail(err, l.Locale)
	}
	return httpx.OK(c, milestonesJSON(b, bs, bandFor(bs, age.Months), l), T("messages.milestone_saved", l.Locale))
}

// Learn is GET /children/{id}/learn?topic=: tips for the child's age (nbl_v16_Learn), featured first.
func (h *Handlers) Learn(c fiber.Ctx) error {
	a, err := h.access(c, false)
	if err != nil {
		return err
	}
	now, l := h.now(c), loc(c)
	topic, err := validateTopic(validation.Query(c), l.Locale, now)
	if err != nil {
		return err
	}
	age := AgeOn(a.Child.BirthDate, civildate.InTehran(now))
	tips, err := h.svc.LearnTips(c, age.Months, topic)
	if err != nil {
		return err
	}
	list := make([]*jsonx.OrderedMap, 0, len(tips))
	var featured any
	for i, t := range tips {
		if i == 0 && t.Featured {
			featured = LearnTipJSON(t, l)
			continue
		}
		list = append(list, LearnTipJSON(t, l))
	}
	var topicOut any
	if topic != "" {
		topicOut = topic
	}
	return httpx.OK(c, jsonx.Obj(
		"age_months", age.Months,
		"age_label", age.Label(l.Locale),
		"topics", TopicsJSON(l.Locale),
		"topic", topicOut,
		"featured", featured,
		"tips", list,
		"disclaimer", T("notes.learn", l.Locale),
	))
}

// Reminders is GET /children/reminders: every visible child's vaccine visits whose reminder is due today (from 3
// days before the due date until it is OverdueAfterDays late), soonest first — the in-app due list a push sender
// also uses (reminders.go).
func (h *Handlers) Reminders(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now, l := h.now(c), loc(c)
	today := civildate.InTehran(now)
	list, err := h.svc.List(c, userID, now)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0)
	for _, a := range list {
		s, err := h.svc.Schedule(c, a.Child, today)
		if err != nil {
			return err
		}
		for _, r := range DueReminders(s, today) {
			items = append(items, jsonx.Obj(
				"child", jsonx.Obj("id", a.Child.ID, "name", a.Child.Name, "initial", initial(a.Child.Name), "role", a.Role),
				"visit", VisitJSON(r.Visit, today, l, false),
				"days_left", r.Days,
			))
		}
	}
	sortReminders(items)
	return httpx.OK(c, jsonx.Obj("reminders", items, "days_before", RemindDaysBefore))
}
