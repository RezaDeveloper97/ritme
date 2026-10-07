package learning

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	domain "github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// CourseStatuses is the ?status= filter of the courses list.
var CourseStatuses = []string{FilterAll, domain.StatusDraft, domain.StatusPublished}

func usageJSON(u usage) *jsonx.OrderedMap {
	return jsonx.Obj("students_count", u.Students, "completion_percent", u.Percent(), "completed_count", u.Completed)
}

// Courses is GET /learning/courses?status=all|draft|published&instructor_id=&q=&page=&per_page= — newest first, with
// counts (chapters, lessons, review queue) and usage (students with a running grant, mean completion).
func (h *Handlers) Courses(c fiber.Ctx) error {
	status := pick(c.Query("status"), CourseStatuses, FilterAll)
	var instructorID uint64
	if s := strings.TrimSpace(c.Query("instructor_id")); s != "" {
		if n, err := strconv.ParseUint(s, 10, 64); err == nil {
			instructorID = n
		}
	}
	q, like := search(c)
	_, now := dbNow(c)
	ctx := c.Context()
	total, err := h.q.CountLearningCourses(ctx, store.CountLearningCoursesParams{
		Status: pattern(status), InstructorID: instructorID, Q: q, QLike: like,
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListLearningCourses(ctx, store.ListLearningCoursesParams{
		Status: pattern(status), InstructorID: instructorID, Q: q, QLike: like,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	use, err := h.courseUsage(c, ids, now)
	if err != nil {
		return err
	}
	counts, err := h.q.CountLearningCoursesByStatus(ctx)
	if err != nil {
		return err
	}
	by := map[string]int64{}
	for _, r := range counts {
		by[r.Status] = r.Total
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		m := jsonx.Obj(
			"id", r.ID,
			"title", r.Title,
			"kind", r.Kind,
			"status", r.Status,
			"instructor", jsonx.Obj("id", r.InstructorID, "display_name", r.InstructorName, "status", r.InstructorStatus),
			"chapters_count", r.ChaptersCount,
			"lessons_count", r.LessonsCount,
			"published_lessons", r.PublishedLessons,
			"review_open", r.ReviewOpen,
			"flagged_count", r.FlaggedCount,
		)
		u := use[r.ID]
		m.Set("students_count", u.Students)
		m.Set("completion_percent", u.Percent())
		m.Set("completed_count", u.Completed)
		m.Set("published_at", httpadmin.Time(r.PublishedAt))
		m.Set("created_at", httpadmin.Time(r.CreatedAt))
		m.Set("updated_at", httpadmin.Time(r.UpdatedAt))
		items = append(items, m)
	}
	page := httpadmin.Page(items, p, int(total))
	filters := jsonx.Obj("status", status, "q", q, "instructor_id", nil)
	if instructorID != 0 {
		filters.Set("instructor_id", instructorID)
	}
	page.Set("filters", filters)
	page.Set("counts", countsOf(by, domain.StatusDraft, domain.StatusPublished))
	page.Set("statuses", CourseStatuses)
	return httpadmin.OK(c, page)
}

// lessonFields are the lesson columns every admin lesson object carries.
type lessonFields struct {
	ID              uint64
	ChapterID       sql.NullInt64
	Kind, Title     string
	Description     sql.NullString
	DurationSeconds sql.NullInt32
	PageCount       sql.NullInt16
	SizeBytes       sql.NullInt64
	MediaStatus     string
	Status          string
	PublishedAt     sql.NullTime
	ReviewStatus    string
	ReviewedAt      sql.NullTime
	ReviewedBy      sql.NullInt64
	ReviewNote      sql.NullString
	ReviewedByName  sql.NullString
	CreatedAt       sql.NullTime
	UpdatedAt       sql.NullTime
}

func lessonJSON(l lessonFields) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", l.ID,
		"chapter_id", nullID(l.ChapterID),
		"kind", l.Kind,
		"title", l.Title,
		"description", httpadmin.NullString(l.Description),
		"duration_seconds", nullInt(l.DurationSeconds.Valid, l.DurationSeconds.Int32),
		"page_count", nullInt(l.PageCount.Valid, l.PageCount.Int16),
		"size_bytes", nullInt(l.SizeBytes.Valid, l.SizeBytes.Int64),
		"media_status", l.MediaStatus,
		"status", l.Status,
		"published_at", httpadmin.Time(l.PublishedAt),
		"review", jsonx.Obj(
			"state", ReviewState(l.ReviewStatus, l.ReviewedAt, l.UpdatedAt),
			"status", l.ReviewStatus,
			"reviewed_at", httpadmin.Time(l.ReviewedAt),
			"reviewed_by", adminRef(l.ReviewedBy, l.ReviewedByName),
			"note", httpadmin.NullString(l.ReviewNote),
		),
		"created_at", httpadmin.Time(l.CreatedAt),
		"updated_at", httpadmin.Time(l.UpdatedAt),
	)
}

// Course is GET /learning/courses/:id: the course, its instructor, chapters, every lesson with its review state,
// grants by state and usage. Never the students or their phone numbers.
func (h *Handlers) Course(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Course")
	}
	ctx := c.Context()
	cr, err := h.q.GetLearningCourse(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Course")
	}
	if err != nil {
		return err
	}
	_, now := dbNow(c)
	chapters, err := h.q.ListLearningCourseChapters(ctx, id)
	if err != nil {
		return err
	}
	lessons, err := h.q.ListLearningCourseLessons(ctx, id)
	if err != nil {
		return err
	}
	grants, err := h.q.CountLearningCourseGrants(ctx, store.CountLearningCourseGrantsParams{
		Now: now, DirectCourseID: sql.NullInt64{Int64: int64(id), Valid: true}, CourseID: id, //nolint:gosec // G115: ids
	})
	if err != nil {
		return err
	}
	use, err := h.courseUsage(c, []uint64{id}, now)
	if err != nil {
		return err
	}
	chJSON := make([]*jsonx.OrderedMap, 0, len(chapters))
	for _, ch := range chapters {
		chJSON = append(chJSON, jsonx.Obj("id", ch.ID, "title", ch.Title, "sort_order", ch.SortOrder,
			"unlock_at", httpadmin.Time(ch.UnlockAt)))
	}
	lJSON := make([]*jsonx.OrderedMap, 0, len(lessons))
	for _, l := range lessons {
		m := lessonJSON(lessonFields{
			ID: l.ID, ChapterID: l.ChapterID, Kind: l.Kind, Title: l.Title, Description: l.Description,
			DurationSeconds: l.DurationSeconds, PageCount: l.PageCount, SizeBytes: l.SizeBytes, MediaStatus: l.MediaStatus,
			Status: l.Status, PublishedAt: l.PublishedAt, ReviewStatus: l.ReviewStatus, ReviewedAt: l.ReviewedAt,
			ReviewedBy: l.ReviewedBy, ReviewNote: l.ReviewNote, ReviewedByName: l.ReviewedByName,
			CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
		})
		m.Set("sort_order", l.SortOrder)
		lJSON = append(lJSON, m)
	}
	course := jsonx.Obj(
		"id", cr.ID,
		"title", cr.Title,
		"description", httpadmin.NullString(cr.Description),
		"kind", cr.Kind,
		"status", cr.Status,
		"instructor", jsonx.Obj("id", cr.InstructorID, "display_name", cr.InstructorName,
			"title", httpadmin.NullString(cr.InstructorTitle), "status", cr.InstructorStatus),
		"chapters", chJSON,
		"lessons", lJSON,
		"grants", grantCounts(func(by map[string]int64) {
			for _, g := range grants {
				by[stateString(g.State)] += g.Total
			}
		}),
		"usage", usageJSON(use[id]),
		"published_at", httpadmin.Time(cr.PublishedAt),
		"created_at", httpadmin.Time(cr.CreatedAt),
		"updated_at", httpadmin.Time(cr.UpdatedAt),
	)
	return httpadmin.OK(c, jsonx.Obj("course", course))
}

// stateString reads a computed CASE column (the driver hands it over as bytes or a string).
func stateString(v any) string {
	switch s := v.(type) {
	case []byte:
		return string(s)
	case string:
		return s
	}
	return ""
}

// grantCounts is {all, active, pending, expired, revoked} from (state, total) rows.
func grantCounts(add func(by map[string]int64)) *jsonx.OrderedMap {
	by := map[string]int64{}
	add(by)
	return countsOf(by, domain.GrantActive, domain.GrantPending, domain.GrantExpired, domain.GrantRevoked)
}

// Stats is GET /learning/stats: platform-wide counts and usage of the courses module.
func (h *Handlers) Stats(c fiber.Ctx) error {
	t, now := dbNow(c)
	ctx := c.Context()
	inst, err := h.q.CountLearningInstructorsByStatus(ctx)
	if err != nil {
		return err
	}
	byInst := map[string]int64{}
	for _, r := range inst {
		byInst[r.Status] = r.Total
	}
	courses, err := h.q.CountLearningCoursesByStatus(ctx)
	if err != nil {
		return err
	}
	byCourse := map[string]int64{}
	for _, r := range courses {
		byCourse[r.Status] = r.Total
	}
	lessons, err := h.q.CountLearningLessons(ctx)
	if err != nil {
		return err
	}
	review, err := h.reviewCounts(c)
	if err != nil {
		return err
	}
	grants, err := h.q.CountLearningGrantsByState(ctx, now)
	if err != nil {
		return err
	}
	since := httpadmin.DBTime(t.Add(-ActiveDays * 24 * time.Hour))
	newGrants, err := h.q.CountLearningGrantsSince(ctx, since)
	if err != nil {
		return err
	}
	active, err := h.q.CountLearningActiveStudents(ctx, since)
	if err != nil {
		return err
	}
	completedLessons, err := h.q.CountLearningCompletedLessons(ctx)
	if err != nil {
		return err
	}
	total, withAccess, err := h.totalUsage(c, now)
	if err != nil {
		return err
	}
	gc := grantCounts(func(by map[string]int64) {
		for _, g := range grants {
			by[stateString(g.State)] += g.Total
		}
	})
	gc.Set("new_30d", newGrants)
	return httpadmin.OK(c, jsonx.Obj(
		"instructors", countsOf(byInst, domain.InstructorPending, domain.InstructorApproved, domain.InstructorRevoked),
		"courses", countsOf(byCourse, domain.StatusDraft, domain.StatusPublished),
		"lessons", jsonx.Obj("all", lessons.Total, "published", lessons.Published),
		"review", review,
		"grants", gc,
		"students", jsonx.Obj("with_access", withAccess, "active_30d", active),
		"completion", jsonx.Obj(
			"percent", total.Percent(), "enrolments", total.Students, "completed", total.Completed,
			"lessons_completed", completedLessons,
		),
		"active_days", ActiveDays,
	))
}

// totalUsage is the completion over every (course, student) pair with access, and the distinct students.
func (h *Handlers) totalUsage(c fiber.Ctx, now sql.NullTime) (usage, int, error) {
	pr, err := h.q.ListLearningAllAccessPairs(c.Context(), now)
	if err != nil {
		return usage{}, 0, err
	}
	sr, err := h.q.SumAllLearningProgress(c.Context())
	if err != nil {
		return usage{}, 0, err
	}
	lessons, err := h.lessonCounts(c)
	if err != nil {
		return usage{}, 0, err
	}
	pairs := make([]pairKey, 0, len(pr))
	users := map[int64]struct{}{}
	for _, p := range pr {
		pairs = append(pairs, pairKey{p.CourseID, uint64(p.UserID.Int64)}) //nolint:gosec // G115: ids
		users[p.UserID.Int64] = struct{}{}
	}
	sums := make(map[pairKey]int64, len(sr))
	for _, s := range sr {
		sums[pairKey{s.CourseID, s.UserID}] = s.PercentSum
	}
	_, total := aggregate(pairs, sums, lessons)
	return total, len(users), nil
}
