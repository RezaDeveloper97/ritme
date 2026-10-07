package learning

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// claim activates the caller's pending grants (safety net for accounts the signup hook missed). It never fails the
// request: the grants stay pending and the next call retries.
func (h *Handlers) claim(c fiber.Ctx, userID uint64) {
	u := auth.CurrentUser(c)
	if u == nil || !u.Mobile.Valid {
		return
	}
	if _, err := h.svc.ClaimPending(c, userID, u.Mobile.String, h.now(c)); err != nil {
		h.svc.logger.ErrorContext(c, "learning: lazy claim failed", slog.Uint64("user_id", userID), slog.String("error", err.Error()))
	}
}

// MyCourses is GET /api/v1/learning/courses (Learn_Hub + the home card): the continue card and every course open to
// the caller (expired ones flagged), most recently opened first.
func (h *Handlers) MyCourses(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	h.claim(c, userID)
	courses, err := h.svc.StudentCourses(c, userID, now)
	if err != nil {
		return err
	}
	out := make([]any, 0, len(courses))
	for _, k := range courses {
		out = append(out, StudentCourseJSON(k, now))
	}
	return httpx.OK(c, jsonx.Obj("continue", ContinueJSON(FindContinue(courses, now)), "courses", out))
}

// ShowCourse is GET /api/v1/learning/courses/{id} (Learn_Course): chapters (locked ones with their unlock time),
// lessons with progress. An expired access still shows the outline (no media) so the student sees «دسترسی تمام شد».
func (h *Handlers) StudentCourse(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrCourseNotFound, locale)
	}
	k, err := h.svc.StudentCourse(c, userID, id, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, StudentOutlineJSON(k, now))
}

// ShowLesson is GET /api/v1/learning/lessons/{id} (Learn_Video / Learn_Audio / Learn_PDF): the lesson, its media
// reference and the next lesson. 403 when the access expired or the chapter is still locked; 404 without access.
func (h *Handlers) ShowLesson(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrLessonNotFound, locale)
	}
	open, err := h.svc.OpenLesson(c, userID, id, now)
	if err != nil {
		return fail(err, locale)
	}
	a := open.Course.Access
	var next any
	if open.NextID != 0 {
		next = open.NextID
	}
	return httpx.OK(c, jsonx.Obj(
		"lesson", StudentLessonJSON(open.Course, open.Lesson, open.Number, now),
		"course", jsonx.Obj("id", a.CourseID, "title", a.Title, "kind", a.Kind, "lessons_count", len(open.Course.Lessons),
			"instructor", instructorRef(a.InstructorID, a.InstructorName, a.InstructorTitle)),
		"next_lesson_id", next,
	))
}

// SaveProgress is PUT /api/v1/learning/lessons/{id}/progress {position_seconds, percent?, completed?}.
func (h *Handlers) SaveProgress(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrLessonNotFound, locale)
	}
	in, err := ValidateProgress(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	open, err := h.svc.OpenLesson(c, userID, id, now)
	if err != nil {
		return fail(err, locale)
	}
	p, err := h.svc.SaveProgress(c, userID, open, in, now)
	if err != nil {
		return err
	}
	open.Course.Progress[p.LessonID] = p
	return httpx.OK(c, jsonx.Obj(
		"progress", jsonx.Obj(
			"lesson_id", p.LessonID,
			"position_seconds", p.PositionSeconds,
			"percent", LessonPercent(int(p.Percent), p.CompletedAt.Valid),
			"completed", p.CompletedAt.Valid,
			"last_seen_at", isoN(p.LastSeenAt),
		),
		"course_percent", open.Course.Percent(),
	), T("messages.progress_saved", locale))
}

// Unlocked is GET /api/v1/learning/unlocked/{grant} (Learn_Unlocked, opened from the inbox notice).
func (h *Handlers) Unlocked(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "grant")
	if !ok {
		return fail(ErrGrantNotFound, locale)
	}
	h.claim(c, userID)
	u, err := h.svc.Unlocked(c, userID, id)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, UnlockedJSON(u, now))
}
