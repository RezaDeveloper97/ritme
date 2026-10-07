package learning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/learning/store"
)

// CourseInput is a validated course (the full state after a create or a partial update).
type CourseInput struct {
	Kind, Title, Description, Status string
}

// LessonInput is a validated lesson (the full state after a create or a partial update).
type LessonInput struct {
	ChapterID       uint64
	Kind, Title     string
	Description     string
	DurationSeconds int // 0 = unknown
	PageCount       int
	SizeBytes       int64
	Status          string
	SortOrder       int // 0 on create = append
}

// ChapterInput is a validated chapter.
type ChapterInput struct {
	Title     string
	SortOrder int // 0 on create = append
	UnlockAt  time.Time
}

// GroupInput is a validated group; CourseIDs replace the group's courses when CourseIDsSet.
type GroupInput struct {
	Name         string
	CourseIDs    []uint64
	CourseIDsSet bool
}

// publishedAt keeps the first publication time; a draft has none.
func publishedAt(status string, cur sql.NullTime, now time.Time) sql.NullTime {
	if status != StatusPublished {
		return sql.NullTime{}
	}
	if cur.Valid {
		return cur
	}
	return nt(now)
}

func n32(v int) sql.NullInt32 {
	return sql.NullInt32{Int32: int32(v), Valid: v > 0} //nolint:gosec // G115: validated ranges
}

func n16(v int) sql.NullInt16 {
	return sql.NullInt16{Int16: int16(v), Valid: v > 0} //nolint:gosec // G115: validated ranges
}

func n64(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: v > 0} }

// ───────────── courses ─────────────

// Courses lists the instructor's courses with lesson counts.
func (s *Service) Courses(ctx context.Context, instructorID uint64) ([]store.ListInstructorCoursesRow, error) {
	rows, err := s.q.ListInstructorCourses(ctx, instructorID)
	if err != nil {
		return nil, fmt.Errorf("learning: courses: %w", err)
	}
	return rows, nil
}

// Course reads one of the instructor's courses.
func (s *Service) Course(ctx context.Context, instructorID, id uint64) (store.LearningCourse, error) {
	r, err := s.q.GetInstructorCourse(ctx, store.GetInstructorCourseParams{ID: id, InstructorID: instructorID})
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrCourseNotFound
	}
	if err != nil {
		return r, fmt.Errorf("learning: course: %w", err)
	}
	return r, nil
}

// CreateCourse adds a course or standalone item.
func (s *Service) CreateCourse(ctx context.Context, instructorID uint64, in CourseInput, now time.Time) (store.LearningCourse, error) {
	n, err := s.q.CountInstructorCourses(ctx, instructorID)
	if err != nil {
		return store.LearningCourse{}, fmt.Errorf("learning: count courses: %w", err)
	}
	if n >= MaxCourses {
		return store.LearningCourse{}, ErrTooMany
	}
	id, err := lastID(s.q.InsertCourse(ctx, store.InsertCourseParams{
		InstructorID: instructorID, Kind: in.Kind, Title: in.Title, Description: ns(in.Description), Status: in.Status,
		PublishedAt: publishedAt(in.Status, sql.NullTime{}, now), Now: nt(now),
	}))
	if err != nil {
		return store.LearningCourse{}, fmt.Errorf("learning: create course: %w", err)
	}
	return s.Course(ctx, instructorID, id)
}

// UpdateCourse saves a course (kind never changes).
func (s *Service) UpdateCourse(ctx context.Context, cur store.LearningCourse, in CourseInput, now time.Time) (store.LearningCourse, error) {
	if _, err := s.q.UpdateCourse(ctx, store.UpdateCourseParams{
		Title: in.Title, Description: ns(in.Description), Status: in.Status,
		PublishedAt: publishedAt(in.Status, cur.PublishedAt, now), Now: nt(now), ID: cur.ID, InstructorID: cur.InstructorID,
	}); err != nil {
		return cur, fmt.Errorf("learning: update course: %w", err)
	}
	return s.Course(ctx, cur.InstructorID, cur.ID)
}

// DeleteCourse removes a course with its chapters, lessons, progress and direct grants.
func (s *Service) DeleteCourse(ctx context.Context, instructorID, id uint64) error {
	n, err := s.q.DeleteCourse(ctx, store.DeleteCourseParams{ID: id, InstructorID: instructorID})
	if err != nil {
		return fmt.Errorf("learning: delete course: %w", err)
	}
	if n == 0 {
		return ErrCourseNotFound
	}
	return nil
}

// Outline reads a course's chapters and every lesson (drafts included: the instructor's view).
func (s *Service) Outline(ctx context.Context, courseID uint64) ([]store.LearningChapter, []store.LearningLesson, error) {
	chapters, err := s.q.ListChapters(ctx, courseID)
	if err != nil {
		return nil, nil, fmt.Errorf("learning: chapters: %w", err)
	}
	lessons, err := s.q.ListLessons(ctx, courseID)
	if err != nil {
		return nil, nil, fmt.Errorf("learning: lessons: %w", err)
	}
	return chapters, lessons, nil
}

// CourseAccess is who can open a course (Ins_Course «چه کسانی دسترسی دارند»).
type CourseAccess struct {
	Groups []store.ListCourseGroupsRow
	Direct int64
}

// Access reads the groups that open a course and its direct active grants.
func (s *Service) Access(ctx context.Context, instructorID, courseID uint64) (CourseAccess, error) {
	groups, err := s.q.ListCourseGroups(ctx, store.ListCourseGroupsParams{CourseID: courseID, InstructorID: instructorID})
	if err != nil {
		return CourseAccess{}, fmt.Errorf("learning: course groups: %w", err)
	}
	direct, err := s.q.CountDirectCourseGrants(ctx, store.CountDirectCourseGrantsParams{CourseID: nid(courseID), InstructorID: instructorID})
	if err != nil {
		return CourseAccess{}, fmt.Errorf("learning: direct grants: %w", err)
	}
	return CourseAccess{Groups: groups, Direct: direct}, nil
}

// ───────────── chapters ─────────────

// Chapter reads one chapter of a course.
func (s *Service) Chapter(ctx context.Context, courseID, id uint64) (store.LearningChapter, error) {
	r, err := s.q.GetChapter(ctx, store.GetChapterParams{ID: id, CourseID: courseID})
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrChapterNotFound
	}
	if err != nil {
		return r, fmt.Errorf("learning: chapter: %w", err)
	}
	return r, nil
}

// CreateChapter appends (or places) a chapter in a course.
func (s *Service) CreateChapter(ctx context.Context, course store.LearningCourse, in ChapterInput, now time.Time) (store.LearningChapter, error) {
	if course.Kind == KindStandalone {
		return store.LearningChapter{}, ErrStandaloneChapter
	}
	n, err := s.q.CountChapters(ctx, course.ID)
	if err != nil {
		return store.LearningChapter{}, fmt.Errorf("learning: count chapters: %w", err)
	}
	if n >= MaxChapters {
		return store.LearningChapter{}, ErrTooMany
	}
	order := in.SortOrder
	if order == 0 {
		mx, err := s.q.MaxChapterSort(ctx, course.ID)
		if err != nil {
			return store.LearningChapter{}, fmt.Errorf("learning: chapter order: %w", err)
		}
		order = int(mx) + 1
	}
	id, err := lastID(s.q.InsertChapter(ctx, store.InsertChapterParams{
		CourseID: course.ID, Title: in.Title, SortOrder: uint16(order), UnlockAt: nt(in.UnlockAt), Now: nt(now), //nolint:gosec // G115: validated range
	}))
	if err != nil {
		return store.LearningChapter{}, fmt.Errorf("learning: create chapter: %w", err)
	}
	return s.Chapter(ctx, course.ID, id)
}

// UpdateChapter saves a chapter.
func (s *Service) UpdateChapter(ctx context.Context, cur store.LearningChapter, in ChapterInput, now time.Time) (store.LearningChapter, error) {
	if _, err := s.q.UpdateChapter(ctx, store.UpdateChapterParams{
		Title: in.Title, SortOrder: uint16(in.SortOrder), UnlockAt: nt(in.UnlockAt), Now: nt(now), //nolint:gosec // G115: validated range
		ID: cur.ID, CourseID: cur.CourseID,
	}); err != nil {
		return cur, fmt.Errorf("learning: update chapter: %w", err)
	}
	return s.Chapter(ctx, cur.CourseID, cur.ID)
}

// DeleteChapter removes a chapter; its lessons stay in the course without a chapter.
func (s *Service) DeleteChapter(ctx context.Context, courseID, id uint64) error {
	n, err := s.q.DeleteChapter(ctx, store.DeleteChapterParams{ID: id, CourseID: courseID})
	if err != nil {
		return fmt.Errorf("learning: delete chapter: %w", err)
	}
	if n == 0 {
		return ErrChapterNotFound
	}
	return nil
}

// ───────────── lessons ─────────────

// Lesson reads one lesson of a course.
func (s *Service) Lesson(ctx context.Context, courseID, id uint64) (store.LearningLesson, error) {
	r, err := s.q.GetLesson(ctx, store.GetLessonParams{ID: id, CourseID: courseID})
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrLessonNotFound
	}
	if err != nil {
		return r, fmt.Errorf("learning: lesson: %w", err)
	}
	return r, nil
}

func (s *Service) checkChapter(ctx context.Context, course store.LearningCourse, chapterID uint64) error {
	if chapterID == 0 {
		return nil
	}
	if course.Kind == KindStandalone {
		return ErrStandaloneChapter
	}
	if _, err := s.Chapter(ctx, course.ID, chapterID); err != nil {
		if errors.Is(err, ErrChapterNotFound) {
			return ErrChapterInvalid
		}
		return err
	}
	return nil
}

// CreateLesson adds a lesson (a standalone item holds exactly one).
func (s *Service) CreateLesson(ctx context.Context, course store.LearningCourse, in LessonInput, now time.Time) (store.LearningLesson, error) {
	if err := s.checkChapter(ctx, course, in.ChapterID); err != nil {
		return store.LearningLesson{}, err
	}
	n, err := s.q.CountLessons(ctx, course.ID)
	if err != nil {
		return store.LearningLesson{}, fmt.Errorf("learning: count lessons: %w", err)
	}
	if course.Kind == KindStandalone && n >= 1 {
		return store.LearningLesson{}, ErrStandaloneSingle
	}
	if n >= MaxLessons {
		return store.LearningLesson{}, ErrTooMany
	}
	order := in.SortOrder
	if order == 0 {
		mx, err := s.q.MaxLessonSort(ctx, course.ID)
		if err != nil {
			return store.LearningLesson{}, fmt.Errorf("learning: lesson order: %w", err)
		}
		order = int(mx) + 1
	}
	id, err := lastID(s.q.InsertLesson(ctx, store.InsertLessonParams{
		CourseID: course.ID, ChapterID: nid(in.ChapterID), Kind: in.Kind, Title: in.Title, Description: ns(in.Description),
		DurationSeconds: n32(in.DurationSeconds), PageCount: n16(in.PageCount), SizeBytes: n64(in.SizeBytes),
		Status: in.Status, PublishedAt: publishedAt(in.Status, sql.NullTime{}, now), SortOrder: uint16(order), //nolint:gosec // G115: validated range
		Now: nt(now),
	}))
	if err != nil {
		return store.LearningLesson{}, fmt.Errorf("learning: create lesson: %w", err)
	}
	return s.Lesson(ctx, course.ID, id)
}

// UpdateLesson saves a lesson.
func (s *Service) UpdateLesson(ctx context.Context, course store.LearningCourse, cur store.LearningLesson, in LessonInput, now time.Time) (store.LearningLesson, error) {
	if err := s.checkChapter(ctx, course, in.ChapterID); err != nil {
		return cur, err
	}
	if _, err := s.q.UpdateLesson(ctx, store.UpdateLessonParams{
		ChapterID: nid(in.ChapterID), Kind: in.Kind, Title: in.Title, Description: ns(in.Description),
		DurationSeconds: n32(in.DurationSeconds), PageCount: n16(in.PageCount), SizeBytes: n64(in.SizeBytes),
		Status: in.Status, PublishedAt: publishedAt(in.Status, cur.PublishedAt, now), SortOrder: uint16(in.SortOrder), //nolint:gosec // G115: validated range
		Now: nt(now), ID: cur.ID, CourseID: course.ID,
	}); err != nil {
		return cur, fmt.Errorf("learning: update lesson: %w", err)
	}
	return s.Lesson(ctx, course.ID, cur.ID)
}

// DeleteLesson removes a lesson and its progress rows.
func (s *Service) DeleteLesson(ctx context.Context, courseID, id uint64) error {
	n, err := s.q.DeleteLesson(ctx, store.DeleteLessonParams{ID: id, CourseID: courseID})
	if err != nil {
		return fmt.Errorf("learning: delete lesson: %w", err)
	}
	if n == 0 {
		return ErrLessonNotFound
	}
	return nil
}

// ───────────── groups ─────────────

// Groups lists the instructor's groups with member counts and, per group id, their courses.
func (s *Service) Groups(ctx context.Context, instructorID uint64) ([]store.ListGroupsRow, map[uint64][]store.ListGroupCoursesRow, error) {
	rows, err := s.q.ListGroups(ctx, instructorID)
	if err != nil {
		return nil, nil, fmt.Errorf("learning: groups: %w", err)
	}
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	courses, err := s.groupCourses(ctx, ids)
	return rows, courses, err
}

func (s *Service) groupCourses(ctx context.Context, ids []uint64) (map[uint64][]store.ListGroupCoursesRow, error) {
	out := map[uint64][]store.ListGroupCoursesRow{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.q.ListGroupCourses(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("learning: group courses: %w", err)
	}
	for _, r := range rows {
		out[r.GroupID] = append(out[r.GroupID], r)
	}
	return out, nil
}

// Group reads one of the instructor's groups and its courses.
func (s *Service) Group(ctx context.Context, instructorID, id uint64) (store.LearningGroup, []store.ListGroupCoursesRow, error) {
	g, err := s.q.GetGroup(ctx, store.GetGroupParams{ID: id, InstructorID: instructorID})
	if errors.Is(err, sql.ErrNoRows) {
		return g, nil, ErrGroupNotFound
	}
	if err != nil {
		return g, nil, fmt.Errorf("learning: group: %w", err)
	}
	courses, err := s.groupCourses(ctx, []uint64{id})
	return g, courses[id], err
}

func (s *Service) checkCourses(ctx context.Context, q *store.Queries, instructorID uint64, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	n, err := q.CountInstructorCoursesIn(ctx, store.CountInstructorCoursesInParams{InstructorID: instructorID, Ids: ids})
	if err != nil {
		return fmt.Errorf("learning: check courses: %w", err)
	}
	if int(n) != len(ids) {
		return ErrCourseIDsInvalid
	}
	return nil
}

func setGroupCourses(ctx context.Context, q *store.Queries, groupID uint64, ids []uint64, now time.Time) error {
	if err := q.DeleteGroupCourses(ctx, groupID); err != nil {
		return fmt.Errorf("learning: group courses: %w", err)
	}
	for _, id := range ids {
		if err := q.InsertGroupCourse(ctx, store.InsertGroupCourseParams{GroupID: groupID, CourseID: id, Now: nt(now)}); err != nil {
			return fmt.Errorf("learning: group course: %w", err)
		}
	}
	return nil
}

// CreateGroup adds a group with its courses.
func (s *Service) CreateGroup(ctx context.Context, instructorID uint64, in GroupInput, now time.Time) (uint64, error) {
	var id uint64
	err := s.tx(ctx, func(q *store.Queries) error {
		n, err := q.CountGroups(ctx, instructorID)
		if err != nil {
			return fmt.Errorf("learning: count groups: %w", err)
		}
		if n >= MaxGroups {
			return ErrTooMany
		}
		if err := s.checkCourses(ctx, q, instructorID, in.CourseIDs); err != nil {
			return err
		}
		id, err = lastID(q.InsertGroup(ctx, store.InsertGroupParams{InstructorID: instructorID, Name: in.Name, Now: nt(now)}))
		if err != nil {
			return fmt.Errorf("learning: create group: %w", err)
		}
		return setGroupCourses(ctx, q, id, in.CourseIDs, now)
	})
	return id, err
}

// UpdateGroup renames a group and, when given, replaces its courses (members get the new set at once).
func (s *Service) UpdateGroup(ctx context.Context, g store.LearningGroup, in GroupInput, now time.Time) error {
	return s.tx(ctx, func(q *store.Queries) error {
		if _, err := q.UpdateGroup(ctx, store.UpdateGroupParams{Name: in.Name, Now: nt(now), ID: g.ID, InstructorID: g.InstructorID}); err != nil {
			return fmt.Errorf("learning: update group: %w", err)
		}
		if !in.CourseIDsSet {
			return nil
		}
		if err := s.checkCourses(ctx, q, g.InstructorID, in.CourseIDs); err != nil {
			return err
		}
		return setGroupCourses(ctx, q, g.ID, in.CourseIDs, now)
	})
}

// DeleteGroup removes a group and the grants given through it.
func (s *Service) DeleteGroup(ctx context.Context, instructorID, id uint64) error {
	n, err := s.q.DeleteGroup(ctx, store.DeleteGroupParams{ID: id, InstructorID: instructorID})
	if err != nil {
		return fmt.Errorf("learning: delete group: %w", err)
	}
	if n == 0 {
		return ErrGroupNotFound
	}
	return nil
}
