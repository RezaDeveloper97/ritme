package learning

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Ins_Dashboard (bloom B-N8-06): the instructor's own aggregates. Counts only — never a phone, a student name or
// whether a number has a Ritme account (B-N8-01b H1).

// DashboardWindow is the «this week» / «last 7 days» window of the stat tiles.
const DashboardWindow = 7 * 24 * time.Hour

// activityWindow bounds the recent-activity feed.
const activityWindow = 30 * 24 * time.Hour

// Dashboard is the data behind GET /api/instructor/v1/dashboard.
type Dashboard struct {
	Students store.DashboardStudentCountsRow
	Watch    int64
	Courses  []store.ListInstructorCoursesRow
	Groups   []store.ListGroupsRow
	Lessons  []store.DashboardRecentLessonsRow
	Grants   []store.DashboardRecentGrantsRow
	Kinds    []store.DashboardLessonKindsRow
}

// Dashboard reads the instructor's stat tiles, courses and recent activity.
func (s *Service) Dashboard(ctx context.Context, instructorID uint64, now time.Time) (Dashboard, error) {
	var d Dashboard
	var err error
	week := nt(now.Add(-DashboardWindow))
	month := nt(now.Add(-activityWindow))
	if d.Students, err = s.q.DashboardStudentCounts(ctx, store.DashboardStudentCountsParams{Now: nt(now), Since: week, InstructorID: instructorID}); err != nil {
		return d, fmt.Errorf("learning: dashboard students: %w", err)
	}
	if d.Watch, err = s.q.DashboardWatchSeconds(ctx, store.DashboardWatchSecondsParams{InstructorID: instructorID, Since: week}); err != nil {
		return d, fmt.Errorf("learning: dashboard watch: %w", err)
	}
	if d.Courses, err = s.q.ListInstructorCourses(ctx, instructorID); err != nil {
		return d, fmt.Errorf("learning: dashboard courses: %w", err)
	}
	if d.Groups, err = s.q.ListGroups(ctx, instructorID); err != nil {
		return d, fmt.Errorf("learning: dashboard groups: %w", err)
	}
	if d.Lessons, err = s.q.DashboardRecentLessons(ctx, store.DashboardRecentLessonsParams{InstructorID: instructorID, Since: month}); err != nil {
		return d, fmt.Errorf("learning: dashboard lessons: %w", err)
	}
	if d.Grants, err = s.q.DashboardRecentGrants(ctx, store.DashboardRecentGrantsParams{InstructorID: instructorID, Since: month}); err != nil {
		return d, fmt.Errorf("learning: dashboard grants: %w", err)
	}
	if d.Kinds, err = s.q.DashboardLessonKinds(ctx, instructorID); err != nil {
		return d, fmt.Errorf("learning: dashboard kinds: %w", err)
	}
	return d, nil
}

type activity struct {
	at  time.Time
	obj *jsonx.OrderedMap
}

// DashboardJSON is the Ins_Dashboard payload.
func DashboardJSON(d Dashboard) *jsonx.OrderedMap {
	kinds := map[uint64][]any{}
	for _, k := range d.Kinds {
		kinds[k.CourseID] = append(kinds[k.CourseID], k.Kind)
	}
	drafts := 0
	courses := make([]any, 0, len(d.Courses))
	for _, c := range d.Courses {
		if c.Status == StatusDraft {
			drafts++
		}
		ks := kinds[c.ID]
		if ks == nil {
			ks = []any{}
		}
		courses = append(courses, CourseListJSON(c).Set("lesson_kinds", ks))
	}
	var members int64
	for _, g := range d.Groups {
		members += g.ActiveMembers + g.PendingMembers
	}
	acts := make([]activity, 0, len(d.Lessons)+len(d.Grants))
	for _, l := range d.Lessons {
		acts = append(acts, activity{timeOf(l.PublishedAt), jsonx.Obj(
			"type", "lesson_published", "at", isoN(l.PublishedAt),
			"lesson", jsonx.Obj("id", l.ID, "title", l.Title),
			"course", jsonx.Obj("id", l.CourseID, "title", l.CourseTitle),
			"group", nil, "count", nil,
		)})
	}
	for _, g := range d.Grants {
		var group, course any
		if g.GroupID.Valid {
			group = jsonx.Obj("id", g.GroupID.Int64, "name", strN(g.GroupName))
		}
		if g.CourseID.Valid {
			course = jsonx.Obj("id", g.CourseID.Int64, "title", strN(g.CourseTitle))
		}
		acts = append(acts, activity{g.LastAt, jsonx.Obj(
			"type", "students_added", "at", iso(g.LastAt),
			"lesson", nil, "course", course, "group", group, "count", g.Added,
		)})
	}
	sort.SliceStable(acts, func(i, j int) bool { return acts[i].at.After(acts[j].at) })
	if len(acts) > 6 {
		acts = acts[:6]
	}
	feed := make([]any, 0, len(acts))
	for _, a := range acts {
		feed = append(feed, a.obj)
	}
	return jsonx.Obj(
		"stats", jsonx.Obj(
			"active_students", d.Students.ActiveStudents,
			"new_students_week", d.Students.NewStudents,
			"courses", len(d.Courses),
			"drafts", drafts,
			"watch_seconds_week", d.Watch,
			"groups", len(d.Groups),
			"group_members", members,
		),
		"courses", courses,
		"activity", feed,
	)
}

// Dashboard is GET /api/instructor/v1/dashboard (Ins_Dashboard): stat tiles, courses and recent activity.
func (h *Handlers) Dashboard(c fiber.Ctx) error {
	d, err := h.svc.Dashboard(c, instructor(c).ID, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, DashboardJSON(d))
}
