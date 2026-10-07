package learning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ritme/backend-go/internal/learning/store"
)

// StudentCourse is one course a student can see, with the grant that opens it (the best one when several do) and
// its published outline and her progress.
type StudentCourse struct {
	Access   store.ListUserCourseAccessRow
	Chapters []store.LearningChapter
	Lessons  []store.LearningLesson // published, in reading order (OrderLessons)
	Progress map[uint64]store.LearningProgress
}

// ExpiresAt is the grant's end (zero = unlimited).
func (c *StudentCourse) ExpiresAt() time.Time { return timeOf(c.Access.ExpiresAt) }

// Expired reports whether the access has ended.
func (c *StudentCourse) Expired(now time.Time) bool { return Expired(c.ExpiresAt(), now) }

// Percent is the course completion (mean over published lessons).
func (c *StudentCourse) Percent() int {
	sum := 0
	for _, l := range c.Lessons {
		if p, ok := c.Progress[l.ID]; ok {
			sum += LessonPercent(int(p.Percent), p.CompletedAt.Valid)
		}
	}
	return CoursePercent(sum, len(c.Lessons))
}

// Completed is the number of completed lessons.
func (c *StudentCourse) Completed() int {
	n := 0
	for _, l := range c.Lessons {
		if p, ok := c.Progress[l.ID]; ok && LessonPercent(int(p.Percent), p.CompletedAt.Valid) >= 100 {
			n++
		}
	}
	return n
}

// chapter returns the lesson's chapter (ok=false: none).
func (c *StudentCourse) chapter(l store.LearningLesson) (store.LearningChapter, bool) {
	id := uid(l.ChapterID)
	for _, ch := range c.Chapters {
		if ch.ID == id {
			return ch, true
		}
	}
	return store.LearningChapter{}, false
}

// LockedUntil is when the lesson's chapter opens (zero = open now).
func (c *StudentCourse) LockedUntil(l store.LearningLesson, now time.Time) time.Time {
	ch, ok := c.chapter(l)
	if !ok || !ch.UnlockAt.Valid || !ch.UnlockAt.Time.After(now) {
		return time.Time{}
	}
	return ch.UnlockAt.Time
}

// OrderLessons sorts lessons in reading order: by chapter order, then lesson order; lessons without a chapter last.
func OrderLessons(chapters []store.LearningChapter, lessons []store.LearningLesson) []store.LearningLesson {
	rank := make(map[uint64]int, len(chapters))
	for i, ch := range chapters {
		rank[ch.ID] = i
	}
	out := append([]store.LearningLesson(nil), lessons...)
	key := func(l store.LearningLesson) int {
		if r, ok := rank[uid(l.ChapterID)]; ok && l.ChapterID.Valid {
			return r
		}
		return len(chapters)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if ki, kj := key(out[i]), key(out[j]); ki != kj {
			return ki < kj
		}
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// better reports whether grant row a opens the course better than b: live beats expired, then unlimited, then the
// later end.
func better(a, b store.ListUserCourseAccessRow, now time.Time) bool {
	ea, eb := timeOf(a.ExpiresAt), timeOf(b.ExpiresAt)
	if xa, xb := Expired(ea, now), Expired(eb, now); xa != xb {
		return !xa
	}
	if ea.IsZero() != eb.IsZero() {
		return ea.IsZero()
	}
	return ea.After(eb)
}

// StudentCourses lists the courses open to userID (expired ones included), most recently opened first, with
// outlines and progress.
func (s *Service) StudentCourses(ctx context.Context, userID uint64, now time.Time) ([]*StudentCourse, error) {
	rows, err := s.q.ListUserCourseAccess(ctx, nid(userID))
	if err != nil {
		return nil, fmt.Errorf("learning: my courses: %w", err)
	}
	var order []uint64
	best := map[uint64]store.ListUserCourseAccessRow{}
	for _, r := range rows {
		cur, ok := best[r.CourseID]
		if !ok {
			order = append(order, r.CourseID)
		}
		if !ok || better(r, cur, now) {
			best[r.CourseID] = r
		}
	}
	if len(order) == 0 {
		return []*StudentCourse{}, nil
	}
	chapters, err := s.q.ListChaptersForCourses(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("learning: my chapters: %w", err)
	}
	lessons, err := s.q.ListPublishedLessonsForCourses(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("learning: my lessons: %w", err)
	}
	progress, err := s.q.ListUserProgressForCourses(ctx, store.ListUserProgressForCoursesParams{UserID: userID, CourseIds: order})
	if err != nil {
		return nil, fmt.Errorf("learning: my progress: %w", err)
	}
	byID := make(map[uint64]*StudentCourse, len(order))
	out := make([]*StudentCourse, 0, len(order))
	for _, id := range order {
		c := &StudentCourse{Access: best[id], Progress: map[uint64]store.LearningProgress{}}
		byID[id] = c
		out = append(out, c)
	}
	for _, ch := range chapters {
		byID[ch.CourseID].Chapters = append(byID[ch.CourseID].Chapters, ch)
	}
	for _, l := range lessons {
		byID[l.CourseID].Lessons = append(byID[l.CourseID].Lessons, l)
	}
	for _, p := range progress {
		byID[p.CourseID].Progress[p.LessonID] = p
	}
	for _, c := range out {
		c.Lessons = OrderLessons(c.Chapters, c.Lessons)
	}
	return out, nil
}

// StudentCourse reads one course open to userID; ErrCourseNotFound when no grant opens it.
func (s *Service) StudentCourse(ctx context.Context, userID, courseID uint64, now time.Time) (*StudentCourse, error) {
	all, err := s.StudentCourses(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	for _, c := range all {
		if c.Access.CourseID == courseID {
			return c, nil
		}
	}
	return nil, ErrCourseNotFound
}

// Continue is the home card «ادامه از جایی که ماندی»: the most recently watched unfinished lesson of a live course.
type Continue struct {
	Course   *StudentCourse
	Lesson   store.LearningLesson
	Number   int
	Progress store.LearningProgress
}

// FindContinue picks the continue card from the student's courses (nil when there is nothing to resume).
func FindContinue(courses []*StudentCourse, now time.Time) *Continue {
	var out *Continue
	for _, c := range courses {
		if c.Expired(now) {
			continue
		}
		for i, l := range c.Lessons {
			p, ok := c.Progress[l.ID]
			if !ok || LessonPercent(int(p.Percent), p.CompletedAt.Valid) >= 100 || !c.LockedUntil(l, now).IsZero() {
				continue
			}
			if out == nil || timeOf(p.LastSeenAt).After(timeOf(out.Progress.LastSeenAt)) {
				out = &Continue{Course: c, Lesson: l, Number: i + 1, Progress: p}
			}
		}
	}
	return out
}

// StudentLesson is a lesson opened by its student.
type StudentLesson struct {
	Course *StudentCourse
	Lesson store.LearningLesson
	Number int
	NextID uint64
}

// OpenLesson checks that userID may open lessonID now: a published lesson of a course a live grant opens, in an
// unlocked chapter. ErrLessonNotFound (no access), ErrAccessExpired, *ChapterLockedError.
func (s *Service) OpenLesson(ctx context.Context, userID, lessonID uint64, now time.Time) (StudentLesson, error) {
	l, err := s.q.GetLessonByID(ctx, lessonID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && l.Status != StatusPublished) {
		return StudentLesson{}, ErrLessonNotFound
	}
	if err != nil {
		return StudentLesson{}, fmt.Errorf("learning: lesson: %w", err)
	}
	c, err := s.StudentCourse(ctx, userID, l.CourseID, now)
	if errors.Is(err, ErrCourseNotFound) {
		return StudentLesson{}, ErrLessonNotFound
	}
	if err != nil {
		return StudentLesson{}, err
	}
	if c.Expired(now) {
		return StudentLesson{}, ErrAccessExpired
	}
	if until := c.LockedUntil(l, now); !until.IsZero() {
		return StudentLesson{}, &ChapterLockedError{UnlockAt: until}
	}
	out := StudentLesson{Course: c, Lesson: l}
	for i, x := range c.Lessons {
		if x.ID == l.ID {
			out.Number = i + 1
			if i+1 < len(c.Lessons) {
				out.NextID = c.Lessons[i+1].ID
			}
		}
	}
	return out, nil
}

// ProgressInput is a validated progress report.
type ProgressInput struct {
	Position   int
	Percent    int // -1 = derive from the lesson duration
	Completed  bool
	CompleteIt bool // `completed` was sent true
}

// SaveProgress records the student's position in an open lesson. The percent never goes down (a rewind keeps it);
// a lesson becomes complete at CompletePercent or when the client says so, and stays complete.
func (s *Service) SaveProgress(ctx context.Context, userID uint64, open StudentLesson, in ProgressInput, now time.Time) (store.LearningProgress, error) {
	l := open.Lesson
	percent := in.Percent
	if percent < 0 {
		percent = 0
		if l.DurationSeconds.Valid && l.DurationSeconds.Int32 > 0 {
			percent = min(100, in.Position*100/int(l.DurationSeconds.Int32))
		}
	}
	completedAt := sql.NullTime{}
	if p, ok := open.Course.Progress[l.ID]; ok {
		percent = max(percent, int(p.Percent))
		completedAt = p.CompletedAt
	}
	if in.CompleteIt {
		percent = 100
	}
	if !completedAt.Valid && percent >= CompletePercent {
		completedAt = nt(now)
	}
	if err := s.q.UpsertProgress(ctx, store.UpsertProgressParams{
		UserID: userID, LessonID: l.ID, CourseID: l.CourseID, PositionSeconds: uint32(max(in.Position, 0)), //nolint:gosec // G115: validated range
		Percent: uint8(min(max(percent, 0), 100)), CompletedAt: completedAt, Now: nt(now), //nolint:gosec // G115: clamped
	}); err != nil {
		return store.LearningProgress{}, fmt.Errorf("learning: save progress: %w", err)
	}
	p, err := s.q.GetProgress(ctx, store.GetProgressParams{UserID: userID, LessonID: l.ID})
	if err != nil {
		return p, fmt.Errorf("learning: read progress: %w", err)
	}
	return p, nil
}

// UnlockedGrant is the Learn_Unlocked screen: one of the student's active grants and the published courses it opens.
type UnlockedGrant struct {
	Grant   store.GetUserGrantRow
	Courses []store.LearningCourse
	Lessons map[uint64][]store.LearningLesson
}

// Unlocked reads one of userID's active grants (ErrGrantNotFound otherwise).
func (s *Service) Unlocked(ctx context.Context, userID, grantID uint64) (UnlockedGrant, error) {
	g, err := s.q.GetUserGrant(ctx, store.GetUserGrantParams{ID: grantID, UserID: nid(userID)})
	if errors.Is(err, sql.ErrNoRows) {
		return UnlockedGrant{}, ErrGrantNotFound
	}
	if err != nil {
		return UnlockedGrant{}, fmt.Errorf("learning: grant: %w", err)
	}
	courses, err := s.q.ListGrantCourses(ctx, store.ListGrantCoursesParams{CourseID: g.CourseID, GroupID: g.GroupID})
	if err != nil {
		return UnlockedGrant{}, fmt.Errorf("learning: grant courses: %w", err)
	}
	out := UnlockedGrant{Grant: g, Courses: courses, Lessons: map[uint64][]store.LearningLesson{}}
	if len(courses) == 0 {
		return out, nil
	}
	ids := make([]uint64, 0, len(courses))
	for _, c := range courses {
		ids = append(ids, c.ID)
	}
	lessons, err := s.q.ListPublishedLessonsForCourses(ctx, ids)
	if err != nil {
		return UnlockedGrant{}, fmt.Errorf("learning: grant lessons: %w", err)
	}
	for _, l := range lessons {
		out.Lessons[l.CourseID] = append(out.Lessons[l.CourseID], l)
	}
	return out, nil
}
