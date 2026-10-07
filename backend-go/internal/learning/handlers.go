package learning

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Error codes of the 403 / 404 bodies.
const (
	ErrorCodeInstructorRequired = "instructor_required"
	ErrorCodeInstructorPending  = "instructor_pending"
	ErrorCodeCourseNotFound     = "learning_course_not_found"
	ErrorCodeChapterNotFound    = "learning_chapter_not_found"
	ErrorCodeLessonNotFound     = "learning_lesson_not_found"
	ErrorCodeGroupNotFound      = "learning_group_not_found"
	ErrorCodeGrantNotFound      = "learning_grant_not_found"
	ErrorCodeAccessExpired      = "learning_access_expired"
	ErrorCodeChapterLocked      = "learning_chapter_locked"
)

const instructorLocal = "learning.instructor"

// Handlers are the /api/v1/learning (student) and /api/instructor/v1 (instructor) actions. Mount them behind the
// locale middleware and auth RequireUser; instructor routes (except /me and /apply) also behind RequireInstructor.
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

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

func notFound(locale, key, code string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages."+key, locale), "error_code", code)
}

// fail maps the service errors to their bodies.
func fail(err error, locale string) error {
	var locked *ChapterLockedError
	switch {
	case errors.As(err, &locked):
		return httpx.Fail(fiber.StatusForbidden, T("messages.chapter_locked", locale), "error_code", ErrorCodeChapterLocked,
			"unlock_at", iso(locked.UnlockAt))
	case errors.Is(err, ErrNotInstructor):
		return httpx.Fail(fiber.StatusForbidden, T("messages.instructor_required", locale), "error_code", ErrorCodeInstructorRequired)
	case errors.Is(err, ErrInstructorPending):
		return httpx.Fail(fiber.StatusForbidden, T("messages.instructor_pending", locale), "error_code", ErrorCodeInstructorPending)
	case errors.Is(err, ErrAccessExpired):
		return httpx.Fail(fiber.StatusForbidden, T("messages.access_expired", locale), "error_code", ErrorCodeAccessExpired)
	case errors.Is(err, ErrCourseNotFound):
		return notFound(locale, "course_not_found", ErrorCodeCourseNotFound)
	case errors.Is(err, ErrChapterNotFound):
		return notFound(locale, "chapter_not_found", ErrorCodeChapterNotFound)
	case errors.Is(err, ErrLessonNotFound):
		return notFound(locale, "lesson_not_found", ErrorCodeLessonNotFound)
	case errors.Is(err, ErrGroupNotFound):
		return notFound(locale, "group_not_found", ErrorCodeGroupNotFound)
	case errors.Is(err, ErrGrantNotFound):
		return notFound(locale, "grant_not_found", ErrorCodeGrantNotFound)
	case errors.Is(err, ErrTooMany):
		return fieldFail(locale, "limit", "too_many")
	case errors.Is(err, ErrStandaloneSingle):
		return fieldFail(locale, "kind", "standalone_single_lesson")
	case errors.Is(err, ErrStandaloneChapter):
		return fieldFail(locale, "chapter_id", "standalone_no_chapters")
	case errors.Is(err, ErrChapterInvalid):
		return fieldFail(locale, "chapter_id", "chapter_invalid")
	case errors.Is(err, ErrTargetInvalid):
		return fieldFail(locale, "target_id", "target_invalid")
	case errors.Is(err, ErrCourseIDsInvalid):
		return fieldFail(locale, "course_ids", "course_ids_invalid")
	}
	return err
}

// ───────────── instructor guard / profile ─────────────

// RequireInstructor lets approved instructors through (403 instructor_required / instructor_pending otherwise — never
// 401, which would end the session) and keeps the row for the handlers.
func (h *Handlers) RequireInstructor(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	ins, ok, err := h.svc.InstructorByUser(c, userID)
	if err != nil {
		return err
	}
	switch {
	case !ok || ins.Status == InstructorRevoked:
		return fail(ErrNotInstructor, locale)
	case ins.Status == InstructorPending:
		return fail(ErrInstructorPending, locale)
	}
	c.Locals(instructorLocal, ins)
	return c.Next()
}

func instructor(c fiber.Ctx) store.LearningInstructor {
	ins, _ := c.Locals(instructorLocal).(store.LearningInstructor)
	return ins
}

// CurrentInstructor is the approved instructor RequireInstructor let through (zero row otherwise; B-N8-02 media).
func CurrentInstructor(c fiber.Ctx) store.LearningInstructor { return instructor(c) }

// Fail maps a learning service error (OpenLesson's 403 / 404) to its response (B-N8-02 media playback).
func Fail(err error, locale string) error { return fail(err, locale) }

// Me is GET /api/instructor/v1/me: the caller's instructor profile and status (null when she never applied), so
// instructor-web can route between «درخواست»، «در انتظار تأیید» and the panel.
func (h *Handlers) Me(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	ins, ok, err := h.svc.InstructorByUser(c, userID)
	if err != nil {
		return err
	}
	var out any
	if ok {
		out = InstructorJSON(ins)
	}
	return httpx.OK(c, jsonx.Obj("instructor", out))
}

// Apply is POST /api/instructor/v1/apply {display_name, title?, bio?}: 201 with a pending application, 200 when it
// updates an existing one (an approved instructor keeps her status).
func (h *Handlers) Apply(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := ValidateApply(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	ins, created, err := h.svc.Apply(c, userID, in, now)
	if err != nil {
		return fail(err, locale)
	}
	if created {
		return httpx.Created(c, jsonx.Obj("instructor", InstructorJSON(ins)), T("messages.applied", locale))
	}
	return httpx.OK(c, jsonx.Obj("instructor", InstructorJSON(ins)), T("messages.updated", locale))
}

// ───────────── courses ─────────────

// Courses is GET /api/instructor/v1/courses (Ins_Content).
func (h *Handlers) Courses(c fiber.Ctx) error {
	rows, err := h.svc.Courses(c, instructor(c).ID)
	if err != nil {
		return err
	}
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, CourseListJSON(r))
	}
	return httpx.OK(c, jsonx.Obj("courses", out))
}

func (h *Handlers) course(c fiber.Ctx, locale string) (store.LearningCourse, error) {
	id, ok := idParam(c, "id")
	if !ok {
		return store.LearningCourse{}, fail(ErrCourseNotFound, locale)
	}
	r, err := h.svc.Course(c, instructor(c).ID, id)
	if err != nil {
		return r, fail(err, locale)
	}
	return r, nil
}

func (h *Handlers) outline(c fiber.Ctx, course store.LearningCourse, now time.Time) (*jsonx.OrderedMap, error) {
	chapters, lessons, err := h.svc.Outline(c, course.ID)
	if err != nil {
		return nil, err
	}
	acc, err := h.svc.Access(c, course.InstructorID, course.ID)
	if err != nil {
		return nil, err
	}
	return OutlineJSON(course, chapters, lessons, acc, now), nil
}

// StoreCourse is POST /api/instructor/v1/courses {kind, title, description?, status?} (201 outline).
func (h *Handlers) StoreCourse(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	in, err := ValidateCourse(validation.Input(c), nil, locale, now)
	if err != nil {
		return err
	}
	r, err := h.svc.CreateCourse(c, instructor(c).ID, in, now)
	if err != nil {
		return fail(err, locale)
	}
	out, err := h.outline(c, r, now)
	if err != nil {
		return err
	}
	return httpx.Created(c, out, T("messages.saved", locale))
}

// ShowCourse is GET /api/instructor/v1/courses/{id} (Ins_Course): chapters, lessons (drafts too), access.
func (h *Handlers) ShowCourse(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	r, err := h.course(c, locale)
	if err != nil {
		return err
	}
	out, err := h.outline(c, r, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

// UpdateCourse is PUT /api/instructor/v1/courses/{id} (partial; `status` publishes / unpublishes).
func (h *Handlers) UpdateCourse(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	cur, err := h.course(c, locale)
	if err != nil {
		return err
	}
	in, err := ValidateCourse(validation.Input(c), &cur, locale, now)
	if err != nil {
		return err
	}
	r, err := h.svc.UpdateCourse(c, cur, in, now)
	if err != nil {
		return fail(err, locale)
	}
	out, err := h.outline(c, r, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, out, T("messages.updated", locale))
}

// DestroyCourse is DELETE /api/instructor/v1/courses/{id}.
func (h *Handlers) DestroyCourse(c fiber.Ctx) error {
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrCourseNotFound, locale)
	}
	if err := h.svc.DeleteCourse(c, instructor(c).ID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}

// ───────────── chapters ─────────────

// StoreChapter is POST /api/instructor/v1/courses/{id}/chapters {title, unlock_at?, sort_order?} (201 {chapter}).
func (h *Handlers) StoreChapter(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	course, err := h.course(c, locale)
	if err != nil {
		return err
	}
	in, err := ValidateChapter(validation.Input(c), nil, locale, now)
	if err != nil {
		return err
	}
	ch, err := h.svc.CreateChapter(c, course, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, jsonx.Obj("chapter", ChapterJSON(ch, now)), T("messages.saved", locale))
}

func (h *Handlers) chapter(c fiber.Ctx, course store.LearningCourse, locale string) (store.LearningChapter, error) {
	id, ok := idParam(c, "chapter")
	if !ok {
		return store.LearningChapter{}, fail(ErrChapterNotFound, locale)
	}
	ch, err := h.svc.Chapter(c, course.ID, id)
	if err != nil {
		return ch, fail(err, locale)
	}
	return ch, nil
}

// UpdateChapter is PUT /api/instructor/v1/courses/{id}/chapters/{chapter} (partial; unlock_at null opens it).
func (h *Handlers) UpdateChapter(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	course, err := h.course(c, locale)
	if err != nil {
		return err
	}
	cur, err := h.chapter(c, course, locale)
	if err != nil {
		return err
	}
	in, err := ValidateChapter(validation.Input(c), &cur, locale, now)
	if err != nil {
		return err
	}
	ch, err := h.svc.UpdateChapter(c, cur, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("chapter", ChapterJSON(ch, now)), T("messages.updated", locale))
}

// DestroyChapter is DELETE /api/instructor/v1/courses/{id}/chapters/{chapter} (its lessons stay, without chapter).
func (h *Handlers) DestroyChapter(c fiber.Ctx) error {
	locale := i18n.Locale(c)
	course, err := h.course(c, locale)
	if err != nil {
		return err
	}
	id, ok := idParam(c, "chapter")
	if !ok {
		return fail(ErrChapterNotFound, locale)
	}
	if err := h.svc.DeleteChapter(c, course.ID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}

// ───────────── lessons ─────────────

// StoreLesson is POST /api/instructor/v1/courses/{id}/lessons (201 {lesson}). The media is attached by the upload
// flow of B-N8-02 (media stays null until then).
func (h *Handlers) StoreLesson(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	course, err := h.course(c, locale)
	if err != nil {
		return err
	}
	in, err := ValidateLesson(validation.Input(c), nil, locale, now)
	if err != nil {
		return err
	}
	l, err := h.svc.CreateLesson(c, course, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, jsonx.Obj("lesson", LessonJSON(l)), T("messages.saved", locale))
}

func (h *Handlers) lesson(c fiber.Ctx, course store.LearningCourse, locale string) (store.LearningLesson, error) {
	id, ok := idParam(c, "lesson")
	if !ok {
		return store.LearningLesson{}, fail(ErrLessonNotFound, locale)
	}
	l, err := h.svc.Lesson(c, course.ID, id)
	if err != nil {
		return l, fail(err, locale)
	}
	return l, nil
}

// UpdateLesson is PUT /api/instructor/v1/courses/{id}/lessons/{lesson} (partial: placement, order, publish).
func (h *Handlers) UpdateLesson(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	course, err := h.course(c, locale)
	if err != nil {
		return err
	}
	cur, err := h.lesson(c, course, locale)
	if err != nil {
		return err
	}
	in, err := ValidateLesson(validation.Input(c), &cur, locale, now)
	if err != nil {
		return err
	}
	l, err := h.svc.UpdateLesson(c, course, cur, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("lesson", LessonJSON(l)), T("messages.updated", locale))
}

// DestroyLesson is DELETE /api/instructor/v1/courses/{id}/lessons/{lesson}.
func (h *Handlers) DestroyLesson(c fiber.Ctx) error {
	locale := i18n.Locale(c)
	course, err := h.course(c, locale)
	if err != nil {
		return err
	}
	id, ok := idParam(c, "lesson")
	if !ok {
		return fail(ErrLessonNotFound, locale)
	}
	if err := h.svc.DeleteLesson(c, course.ID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}

// ───────────── groups & grants ─────────────

// Groups is GET /api/instructor/v1/groups (Ins_Groups).
func (h *Handlers) Groups(c fiber.Ctx) error {
	rows, courses, err := h.svc.Groups(c, instructor(c).ID)
	if err != nil {
		return err
	}
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, GroupListJSON(r, courses[r.ID]))
	}
	return httpx.OK(c, jsonx.Obj("groups", out))
}

// groupDetail is Ins_Group: the group, its courses and members (pending signups included) with their progress over
// the group's published courses.
func (h *Handlers) groupDetail(c fiber.Ctx, g store.LearningGroup, courses []store.ListGroupCoursesRow, now time.Time) (*jsonx.OrderedMap, error) {
	members, err := h.svc.Grants(c, g.InstructorID, g.ID)
	if err != nil {
		return nil, err
	}
	var users, courseIDs []uint64
	for _, m := range members {
		if id := uid(m.UserID); id != 0 {
			users = append(users, id)
		}
	}
	for _, r := range courses {
		if r.Status == StatusPublished {
			courseIDs = append(courseIDs, r.CourseID)
		}
	}
	progress, err := h.svc.MemberProgress(c, users, courseIDs)
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, len(members))
	pending := 0
	for _, m := range members {
		var p *int
		if v, ok := progress[uid(m.UserID)]; ok {
			p = &v
		}
		if m.Status == GrantPending {
			pending++
		}
		list = append(list, GrantJSON(m, p, now))
	}
	return jsonx.Obj("group", GroupJSON(g, courses), "members", list, "pending_signups", pending), nil
}

// StoreGroup is POST /api/instructor/v1/groups {name, course_ids?} (201 group detail).
func (h *Handlers) StoreGroup(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	in, err := ValidateGroup(validation.Input(c), nil, locale, now)
	if err != nil {
		return err
	}
	id, err := h.svc.CreateGroup(c, instructor(c).ID, in, now)
	if err != nil {
		return fail(err, locale)
	}
	g, courses, err := h.svc.Group(c, instructor(c).ID, id)
	if err != nil {
		return err
	}
	out, err := h.groupDetail(c, g, courses, now)
	if err != nil {
		return err
	}
	return httpx.Created(c, out, T("messages.saved", locale))
}

func (h *Handlers) group(c fiber.Ctx, locale string) (store.LearningGroup, []store.ListGroupCoursesRow, error) {
	id, ok := idParam(c, "id")
	if !ok {
		return store.LearningGroup{}, nil, fail(ErrGroupNotFound, locale)
	}
	g, courses, err := h.svc.Group(c, instructor(c).ID, id)
	if err != nil {
		return g, nil, fail(err, locale)
	}
	return g, courses, nil
}

// ShowGroup is GET /api/instructor/v1/groups/{id}.
func (h *Handlers) ShowGroup(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	g, courses, err := h.group(c, locale)
	if err != nil {
		return err
	}
	out, err := h.groupDetail(c, g, courses, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

// UpdateGroup is PUT /api/instructor/v1/groups/{id} {name?, course_ids?}.
func (h *Handlers) UpdateGroup(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	cur, _, err := h.group(c, locale)
	if err != nil {
		return err
	}
	in, err := ValidateGroup(validation.Input(c), &cur, locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdateGroup(c, cur, in, now); err != nil {
		return fail(err, locale)
	}
	g, courses, err := h.svc.Group(c, cur.InstructorID, cur.ID)
	if err != nil {
		return err
	}
	out, err := h.groupDetail(c, g, courses, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, out, T("messages.updated", locale))
}

// DestroyGroup is DELETE /api/instructor/v1/groups/{id} (the grants given through it end).
func (h *Handlers) DestroyGroup(c fiber.Ctx) error {
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrGroupNotFound, locale)
	}
	if err := h.svc.DeleteGroup(c, instructor(c).ID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}

// Students is GET /api/instructor/v1/students?status=active|pending|expired (Ins_Students): every non-revoked grant.
func (h *Handlers) Students(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	q := pick(validation.Query(c), []string{"status"})
	if err := validate(q, validation.Rules{
		validation.F("status", "nullable", validation.In(GrantActive, GrantPending, GrantExpired)),
	}, locale, now); err != nil {
		return err
	}
	status := str(q, "status")
	rows, err := h.svc.Grants(c, instructor(c).ID, 0)
	if err != nil {
		return err
	}
	out := make([]any, 0, len(rows))
	counts := map[string]int{GrantActive: 0, GrantPending: 0, GrantExpired: 0}
	for _, r := range rows {
		st := GrantState(r.Status, r.ExpiresAt, now)
		counts[st]++
		if status == "" || st == status {
			out = append(out, GrantJSON(r, nil, now))
		}
	}
	return httpx.OK(c, jsonx.Obj(
		"counts", jsonx.Obj("active", counts[GrantActive], "pending", counts[GrantPending], "expired", counts[GrantExpired]),
		"students", out,
	))
}

// StoreGrants is POST /api/instructor/v1/grants {phones, scope, target_id, duration, until?} (Ins_AddUser): 201 with
// one result per phone (registered → active now, else pending until signup).
func (h *Handlers) StoreGrants(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	in, err := ValidateGrants(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	res, err := h.svc.CreateGrants(c, instructor(c), in, now)
	if err != nil {
		return fail(err, locale)
	}
	out := make([]any, 0, len(res))
	active, pending := 0, 0
	for _, r := range res {
		if r.Registered {
			active++
		} else {
			pending++
		}
		out = append(out, GrantResultJSON(r, now))
	}
	return httpx.Created(c, jsonx.Obj("grants", out, "summary", jsonx.Obj("active", active, "pending", pending)),
		T("messages.granted", locale))
}

// DestroyGrant is DELETE /api/instructor/v1/grants/{id}: revokes the access at once.
func (h *Handlers) DestroyGrant(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrGrantNotFound, locale)
	}
	if err := h.svc.RevokeGrant(c, instructor(c).ID, id, now); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.revoked", locale))
}
