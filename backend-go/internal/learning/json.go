package learning

import (
	"database/sql"
	"time"

	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func iso(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return jsonx.ISO8601(t.In(civildate.Tehran))
}

func isoN(t sql.NullTime) any { return iso(timeOf(t)) }

func strN(s sql.NullString) any {
	if !s.Valid || s.String == "" {
		return nil
	}
	return s.String
}

func intN32(v sql.NullInt32) any {
	if !v.Valid {
		return nil
	}
	return v.Int32
}

func intN16(v sql.NullInt16) any {
	if !v.Valid {
		return nil
	}
	return v.Int16
}

func intN64(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

// mediaJSON is the lesson's media reference (B-N8-02 fills it; null while nothing was uploaded).
func mediaJSON(l store.LearningLesson) any {
	if !l.MediaID.Valid {
		return nil
	}
	return jsonx.Obj("id", l.MediaID.Int64, "status", l.MediaStatus)
}

// InstructorJSON is the instructor's own profile and status.
func InstructorJSON(r store.LearningInstructor) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"display_name", r.DisplayName,
		"title", strN(r.Title),
		"bio", strN(r.Bio),
		"status", r.Status,
		"approved_at", isoN(r.ApprovedAt),
		"created_at", isoN(r.CreatedAt),
	)
}

// CourseListJSON is one row of the instructor's course list.
func CourseListJSON(r store.ListInstructorCoursesRow) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"kind", r.Kind,
		"title", r.Title,
		"description", strN(r.Description),
		"status", r.Status,
		"published_at", isoN(r.PublishedAt),
		"cover_media_id", intN64(r.CoverMediaID),
		"lessons_count", r.LessonsCount,
		"published_lessons", r.PublishedLessons,
		"duration_seconds", r.DurationSeconds,
		"created_at", isoN(r.CreatedAt),
	)
}

// CourseJSON is one course (instructor view, without the outline).
func CourseJSON(r store.LearningCourse) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"kind", r.Kind,
		"title", r.Title,
		"description", strN(r.Description),
		"status", r.Status,
		"published_at", isoN(r.PublishedAt),
		"cover_media_id", intN64(r.CoverMediaID),
		"created_at", isoN(r.CreatedAt),
	)
}

// ChapterJSON is one chapter (instructor view).
func ChapterJSON(r store.LearningChapter, now time.Time) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"title", r.Title,
		"sort_order", r.SortOrder,
		"unlock_at", isoN(r.UnlockAt),
		"locked", r.UnlockAt.Valid && r.UnlockAt.Time.After(now),
	)
}

// LessonJSON is one lesson (instructor view: drafts and media state included).
func LessonJSON(r store.LearningLesson) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"chapter_id", intN64(r.ChapterID),
		"kind", r.Kind,
		"title", r.Title,
		"description", strN(r.Description),
		"duration_seconds", intN32(r.DurationSeconds),
		"page_count", intN16(r.PageCount),
		"size_bytes", intN64(r.SizeBytes),
		"status", r.Status,
		"published_at", isoN(r.PublishedAt),
		"sort_order", r.SortOrder,
		"media", mediaJSON(r),
		"media_status", r.MediaStatus,
	)
}

// OutlineJSON is a course with its chapters, lessons and access summary (Ins_Course).
func OutlineJSON(c store.LearningCourse, chapters []store.LearningChapter, lessons []store.LearningLesson, acc CourseAccess, now time.Time) *jsonx.OrderedMap {
	chs := make([]any, 0, len(chapters))
	for _, ch := range chapters {
		chs = append(chs, ChapterJSON(ch, now))
	}
	ls := make([]any, 0, len(lessons))
	total := int64(0)
	for _, l := range OrderLessons(chapters, lessons) {
		ls = append(ls, LessonJSON(l))
		total += int64(l.DurationSeconds.Int32)
	}
	groups := make([]any, 0, len(acc.Groups))
	students := acc.Direct
	for _, g := range acc.Groups {
		groups = append(groups, jsonx.Obj("id", g.ID, "name", g.Name, "members", g.ActiveMembers))
		students += g.ActiveMembers
	}
	return jsonx.Obj(
		"course", CourseJSON(c),
		"chapters", chs,
		"lessons", ls,
		"duration_seconds", total,
		"access", jsonx.Obj("students", students, "groups", groups, "direct", acc.Direct),
	)
}

func groupCoursesJSON(rows []store.ListGroupCoursesRow) []any {
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, jsonx.Obj("id", r.CourseID, "kind", r.Kind, "title", r.Title, "status", r.Status))
	}
	return out
}

// GroupListJSON is one row of the instructor's groups.
func GroupListJSON(r store.ListGroupsRow, courses []store.ListGroupCoursesRow) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"name", r.Name,
		"courses", groupCoursesJSON(courses),
		"members", jsonx.Obj("active", r.ActiveMembers, "pending", r.PendingMembers),
		"created_at", isoN(r.CreatedAt),
	)
}

// GroupJSON is a group without counts.
func GroupJSON(g store.LearningGroup, courses []store.ListGroupCoursesRow) *jsonx.OrderedMap {
	return jsonx.Obj("id", g.ID, "name", g.Name, "courses", groupCoursesJSON(courses), "created_at", isoN(g.CreatedAt))
}

func durationJSON(kind string, days sql.NullInt16, until civildate.NullDate) *jsonx.OrderedMap {
	var u any
	if until.Valid {
		u = until.Date.String()
	}
	return jsonx.Obj("type", kind, "days", intN16(days), "until", u)
}

// GrantJSON is one grant as its instructor sees it (the phone she typed, the student's name once registered).
func GrantJSON(r store.ListInstructorGrantsRow, progress *int, now time.Time) *jsonx.OrderedMap {
	var group, course any
	if r.GroupID.Valid {
		group = jsonx.Obj("id", r.GroupID.Int64, "name", strN(r.GroupName))
	}
	if r.CourseID.Valid {
		course = jsonx.Obj("id", r.CourseID.Int64, "title", strN(r.CourseTitle))
	}
	var prog any
	if progress != nil {
		prog = *progress
	}
	return jsonx.Obj(
		"id", r.ID,
		"phone", r.Phone,
		"status", GrantState(r.Status, r.ExpiresAt, now),
		"registered", r.UserID.Valid,
		"student_name", strN(r.UserName),
		"group", group,
		"course", course,
		"duration", durationJSON(r.Duration, r.DurationDays, r.UntilDate),
		"activated_at", isoN(r.ActivatedAt),
		"expires_at", isoN(r.ExpiresAt),
		"days_left", daysLeftJSON(timeOf(r.ExpiresAt), now, r.Status == GrantActive),
		"progress_percent", prog,
		"created_at", isoN(r.CreatedAt),
	)
}

// GrantResultJSON is one phone's outcome of POST /grants.
func GrantResultJSON(r GrantResult, now time.Time) *jsonx.OrderedMap {
	g := r.Grant
	return jsonx.Obj(
		"id", g.ID,
		"phone", g.Phone,
		"status", GrantState(g.Status, g.ExpiresAt, now),
		"registered", r.Registered,
		"renewed", r.Renewed,
		"duration", durationJSON(g.Duration, g.DurationDays, g.UntilDate),
		"activated_at", isoN(g.ActivatedAt),
		"expires_at", isoN(g.ExpiresAt),
	)
}

func daysLeftJSON(expires, now time.Time, active bool) any {
	if !active || expires.IsZero() {
		return nil
	}
	return DaysLeft(expires, now)
}

// ───────────── student ─────────────

func primaryKind(lessons []store.LearningLesson) any {
	if len(lessons) == 0 {
		return nil
	}
	return lessons[0].Kind
}

func instructorRef(id uint64, name string, title sql.NullString) *jsonx.OrderedMap {
	return jsonx.Obj("id", id, "name", name, "title", strN(title))
}

// AccessJSON is how long the student may open the course (Learn_Hub «۸۵ روز دسترسی» / «دسترسی تمام شد»).
func AccessJSON(c *StudentCourse, now time.Time) *jsonx.OrderedMap {
	exp := c.ExpiresAt()
	var group any
	if c.Access.GroupID.Valid {
		group = jsonx.Obj("id", c.Access.GroupID.Int64, "name", strN(c.Access.GroupName))
	}
	var days any
	if !exp.IsZero() {
		days = DaysLeft(exp, now)
	}
	return jsonx.Obj(
		"grant_id", c.Access.GrantID,
		"unlimited", exp.IsZero(),
		"expires_at", iso(exp),
		"days_left", days,
		"expired", c.Expired(now),
		"granted_at", isoN(c.Access.ActivatedAt),
		"group", group,
	)
}

// StudentCourseJSON is one card of «دوره‌های من».
func StudentCourseJSON(c *StudentCourse, now time.Time) *jsonx.OrderedMap {
	a := c.Access
	dur, pages := int64(0), int64(0)
	for _, l := range c.Lessons {
		dur += int64(l.DurationSeconds.Int32)
		pages += int64(l.PageCount.Int16)
	}
	return jsonx.Obj(
		"id", a.CourseID,
		"kind", a.Kind,
		"title", a.Title,
		"description", strN(a.Description),
		"cover_media_id", intN64(a.CoverMediaID),
		"instructor", instructorRef(a.InstructorID, a.InstructorName, a.InstructorTitle),
		"lessons_count", len(c.Lessons),
		"completed_lessons", c.Completed(),
		"primary_kind", primaryKind(c.Lessons),
		"duration_seconds", dur,
		"page_count", pages,
		"progress_percent", c.Percent(),
		"access", AccessJSON(c, now),
	)
}

// StudentLessonJSON is one lesson as the student sees it (media only when open).
func StudentLessonJSON(c *StudentCourse, l store.LearningLesson, number int, now time.Time) *jsonx.OrderedMap {
	until := c.LockedUntil(l, now)
	locked := !until.IsZero() || c.Expired(now)
	var progress any
	if p, ok := c.Progress[l.ID]; ok {
		progress = jsonx.Obj(
			"position_seconds", p.PositionSeconds,
			"percent", LessonPercent(int(p.Percent), p.CompletedAt.Valid),
			"completed", p.CompletedAt.Valid,
			"last_seen_at", isoN(p.LastSeenAt),
		)
	}
	var media any
	if !locked {
		media = mediaJSON(l)
	}
	return jsonx.Obj(
		"id", l.ID,
		"number", number,
		"chapter_id", intN64(l.ChapterID),
		"kind", l.Kind,
		"title", l.Title,
		"description", strN(l.Description),
		"duration_seconds", intN32(l.DurationSeconds),
		"page_count", intN16(l.PageCount),
		"size_bytes", intN64(l.SizeBytes),
		"is_new", l.PublishedAt.Valid && now.Sub(l.PublishedAt.Time) < 7*24*time.Hour,
		"locked", locked,
		"unlock_at", iso(until),
		"progress", progress,
		"media", media,
	)
}

// StudentOutlineJSON is Learn_Course: the card, chapters with their lessons, and the lessons outside any chapter.
func StudentOutlineJSON(c *StudentCourse, now time.Time) *jsonx.OrderedMap {
	byChapter := map[uint64][]any{}
	var loose []any
	var next any
	for i, l := range c.Lessons {
		j := StudentLessonJSON(c, l, i+1, now)
		if id := uid(l.ChapterID); id != 0 && l.ChapterID.Valid {
			byChapter[id] = append(byChapter[id], j)
		} else {
			loose = append(loose, j)
		}
		if next == nil && c.LockedUntil(l, now).IsZero() {
			if p, ok := c.Progress[l.ID]; !ok || LessonPercent(int(p.Percent), p.CompletedAt.Valid) < 100 {
				next = jsonx.Obj("id", l.ID, "number", i+1, "title", l.Title)
			}
		}
	}
	chs := make([]any, 0, len(c.Chapters))
	for _, ch := range c.Chapters {
		ls := byChapter[ch.ID]
		if ls == nil {
			ls = []any{}
		}
		chs = append(chs, jsonx.Obj(
			"id", ch.ID,
			"title", ch.Title,
			"locked", ch.UnlockAt.Valid && ch.UnlockAt.Time.After(now),
			"unlock_at", isoN(ch.UnlockAt),
			"lessons", ls,
		))
	}
	if loose == nil {
		loose = []any{}
	}
	if c.Expired(now) {
		next = nil
	}
	return jsonx.Obj(
		"course", StudentCourseJSON(c, now),
		"next_lesson", next,
		"chapters", chs,
		"lessons", loose,
	)
}

// ContinueJSON is the home card «ادامه از جایی که ماندی» (null when nothing to resume).
func ContinueJSON(k *Continue) any {
	if k == nil {
		return nil
	}
	var remaining any
	if k.Lesson.DurationSeconds.Valid {
		remaining = max(int64(k.Lesson.DurationSeconds.Int32)-int64(k.Progress.PositionSeconds), 0)
	}
	a := k.Course.Access
	return jsonx.Obj(
		"course", jsonx.Obj("id", a.CourseID, "title", a.Title, "kind", a.Kind,
			"instructor", instructorRef(a.InstructorID, a.InstructorName, a.InstructorTitle)),
		"lesson", jsonx.Obj("id", k.Lesson.ID, "number", k.Number, "kind", k.Lesson.Kind, "title", k.Lesson.Title,
			"duration_seconds", intN32(k.Lesson.DurationSeconds)),
		"position_seconds", k.Progress.PositionSeconds,
		"remaining_seconds", remaining,
		"percent", LessonPercent(int(k.Progress.Percent), k.Progress.CompletedAt.Valid),
		"course_percent", k.Course.Percent(),
	)
}

// UnlockedJSON is Learn_Unlocked: who opened what, until when.
func UnlockedJSON(u UnlockedGrant, now time.Time) *jsonx.OrderedMap {
	g := u.Grant
	courses := make([]any, 0, len(u.Courses))
	for _, c := range u.Courses {
		ls := u.Lessons[c.ID]
		dur, pages := int64(0), int64(0)
		for _, l := range ls {
			dur += int64(l.DurationSeconds.Int32)
			pages += int64(l.PageCount.Int16)
		}
		courses = append(courses, jsonx.Obj(
			"id", c.ID, "kind", c.Kind, "title", c.Title, "lessons_count", len(ls), "primary_kind", primaryKind(ls),
			"duration_seconds", dur, "page_count", pages,
		))
	}
	var group any
	if g.GroupID.Valid {
		group = jsonx.Obj("id", g.GroupID.Int64, "name", strN(g.GroupName))
	}
	exp := timeOf(g.ExpiresAt)
	var days any
	if !exp.IsZero() {
		days = DaysLeft(exp, now)
	}
	return jsonx.Obj(
		"grant_id", g.ID,
		"instructor", instructorRef(g.InstructorID, g.InstructorName, g.InstructorTitle),
		"group", group,
		"courses", courses,
		"unlimited", exp.IsZero(),
		"expires_at", iso(exp),
		"days_left", days,
		"expired", Expired(exp, now),
		"granted_at", isoN(g.ActivatedAt),
	)
}
