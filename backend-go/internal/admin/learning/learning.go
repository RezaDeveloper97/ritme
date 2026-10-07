// Package learning is the admin moderation of courses «مدرسین و دوره‌ها» (bloom B-N8-08, admin-api.md §20) over the
// B-N8-01 courses domain (internal/learning, goose 00046) and the review state of goose 00054:
//
//   - instructors: list (status filter, pending first) and detail with the moderation history; approve / revoke
//     (super admins only — approving lets a user publish health education to students);
//   - courses: list (status, instructor, search) with per-course counts and usage, and a course outline;
//   - content review queue: published lessons in a review state (pending | changed | flagged | approved) with
//     approve / flag / unpublish (any active admin). B-N9-10's safety queue reads the same state;
//   - usage stats: grants, students with access, active students, completion.
//
// Every moderation action writes one learning_moderation_log row (the durable trail) and the slog "admin audit"
// line, with ids only. No health data: students are counted, never listed; progress is aggregated per course; phone
// numbers of grants are never selected and the instructor's own mobile is masked.
package learning

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	domain "github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Review states of a lesson. pending / approved / flagged are stored (learning_lessons.review_status); changed is an
// approved lesson edited after the review (updated_at > reviewed_at). Open = what still needs an admin.
const (
	ReviewPending  = "pending"
	ReviewChanged  = "changed"
	ReviewFlagged  = "flagged"
	ReviewApproved = "approved"
	ReviewOpen     = "open"
	FilterAll      = "all"
)

// ReviewStates is the display order of the review filter.
var ReviewStates = []string{ReviewOpen, ReviewPending, ReviewChanged, ReviewFlagged, ReviewApproved, FilterAll}

// Moderation log actions.
const (
	ActionInstructorApprove = "instructor.approve"
	ActionInstructorRevoke  = "instructor.revoke"
	ActionLessonApprove     = "lesson.approve"
	ActionLessonFlag        = "lesson.flag"
	ActionLessonUnpublish   = "lesson.unpublish"
)

// Limits.
const (
	MaxNoteLen = 300
	MinNoteLen = 3
	// ActiveDays is the window of «active students» (opened a lesson).
	ActiveDays = 30
)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve /api/admin/v1/learning/*.
type Handlers struct {
	db     *sql.DB
	q      *store.Queries
	logger *slog.Logger
}

// NewHandlers wires the handlers.
func NewHandlers(db *sql.DB, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{db: db, q: store.New(db), logger: logger}
}

// Routes mounts the module.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	a, s := kit.Admin, kit.Super
	p := "/learning"
	route(fiber.MethodGet, p+"/stats", a(h.Stats))
	route(fiber.MethodGet, p+"/instructors", a(h.Instructors))
	route(fiber.MethodGet, p+"/instructors/:id", a(h.Instructor))
	route(fiber.MethodPost, p+"/instructors/:id/approve", s(h.ApproveInstructor))
	route(fiber.MethodPost, p+"/instructors/:id/revoke", s(h.RevokeInstructor))
	route(fiber.MethodGet, p+"/courses", a(h.Courses))
	route(fiber.MethodGet, p+"/courses/:id", a(h.Course))
	route(fiber.MethodGet, p+"/reviews", a(h.Reviews))
	route(fiber.MethodPost, p+"/lessons/:id/approve", a(h.ApproveLesson))
	route(fiber.MethodPost, p+"/lessons/:id/flag", a(h.FlagLesson))
	route(fiber.MethodPost, p+"/lessons/:id/unpublish", a(h.UnpublishLesson))
}

// ───────────── helpers ─────────────

func dbNow(c fiber.Ctx) (time.Time, sql.NullTime) {
	now := httpadmin.Now(c)
	return now, httpadmin.DBTime(now)
}

func adminID(c fiber.Ctx) sql.NullInt64 {
	if a := httpadmin.CurrentAdmin(c); a != nil {
		return sql.NullInt64{Int64: int64(a.ID), Valid: true} //nolint:gosec // G115: auto-increment ids
	}
	return sql.NullInt64{}
}

func nullID(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func nullInt[T ~int16 | ~int32 | ~int64](valid bool, v T) any {
	if !valid {
		return nil
	}
	return int64(v)
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// search is the trimmed ?q= (Persian digits → ASCII) and its LIKE pattern.
func search(c fiber.Ctx) (q, like string) {
	q = strings.TrimSpace(c.Query("q"))
	q = strings.Map(func(r rune) rune {
		switch {
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		}
		return r
	}, q)
	if len([]rune(q)) > 100 {
		q = string([]rune(q)[:100])
	}
	return q, form.Contains(q)
}

func pick(v string, allowed []string, def string) string {
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	return def
}

func pattern(v string) string {
	if v == FilterAll {
		return "%"
	}
	return v
}

// countsOf is {all, <key>: n…} from per-status totals (keys missing in the map count 0).
func countsOf(by map[string]int64, keys ...string) *jsonx.OrderedMap {
	var all int64
	for _, n := range by {
		all += n
	}
	m := jsonx.Obj(FilterAll, all)
	for _, k := range keys {
		m.Set(k, by[k])
	}
	return m
}

func adminRef(id sql.NullInt64, name sql.NullString) any {
	if !id.Valid {
		return nil
	}
	return jsonx.Obj("id", id.Int64, "name", httpadmin.NullString(name))
}

// ReviewState is a lesson's review state from its stored review fields (see the constants).
func ReviewState(status string, reviewedAt, updatedAt sql.NullTime) string {
	switch status {
	case ReviewFlagged:
		return ReviewFlagged
	case ReviewApproved:
		if !reviewedAt.Valid || (updatedAt.Valid && updatedAt.Time.After(reviewedAt.Time)) {
			return ReviewChanged
		}
		return ReviewApproved
	}
	return ReviewPending
}

func note(c fiber.Ctx, required bool) (sql.NullString, error) {
	rule := "nullable|string|max:" + fmt.Sprint(MaxNoteLen)
	if required {
		rule = fmt.Sprintf("required|string|min:%d|max:%d", MinNoteLen, MaxNoteLen)
	}
	data, err := form.Validate(c, validation.Rules{validation.F("note", rule)})
	if err != nil {
		return sql.NullString{}, err
	}
	return ns(strings.TrimSpace(httpadmin.String(data, "note"))), nil
}

func (h *Handlers) tx(c fiber.Ctx, fn func(q *store.Queries) error) error {
	t, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	if err := fn(h.q.WithTx(t)); err != nil {
		_ = t.Rollback()
		return err
	}
	return t.Commit()
}

// ───────────── completion ─────────────

// usage is the students / completion of one course (or of every course together).
type usage struct {
	Students  int // (course, student) pairs with a running grant
	Completed int // of those, at 100 %
	sum       int // summed course percents
}

// Percent is the mean course completion over the students with access (0 when none).
func (u usage) Percent() int {
	if u.Students == 0 {
		return 0
	}
	return domain.CoursePercent(u.sum, u.Students)
}

type pairKey struct{ course, user uint64 }

// aggregate folds access pairs, progress sums and published-lesson counts into usage per course (+ the total).
func aggregate(pairs []pairKey, sums map[pairKey]int64, lessons map[uint64]int64) (map[uint64]usage, usage) {
	per := map[uint64]usage{}
	var total usage
	for _, p := range pairs {
		pct := domain.CoursePercent(int(sums[p]), int(lessons[p.course]))
		u := per[p.course]
		u.Students++
		u.sum += pct
		if pct >= 100 {
			u.Completed++
		}
		per[p.course] = u
		total.Students++
		total.sum += pct
		if pct >= 100 {
			total.Completed++
		}
	}
	return per, total
}

func (h *Handlers) lessonCounts(c fiber.Ctx) (map[uint64]int64, error) {
	rows, err := h.q.CountLearningPublishedLessons(c.Context())
	if err != nil {
		return nil, err
	}
	m := make(map[uint64]int64, len(rows))
	for _, r := range rows {
		m[r.CourseID] = r.Total
	}
	return m, nil
}

// courseUsage is usage for the given courses.
func (h *Handlers) courseUsage(c fiber.Ctx, ids []uint64, now sql.NullTime) (map[uint64]usage, error) {
	if len(ids) == 0 {
		return map[uint64]usage{}, nil
	}
	pr, err := h.q.ListLearningAccessPairs(c.Context(), store.ListLearningAccessPairsParams{Now: now, CourseIds: ids})
	if err != nil {
		return nil, err
	}
	sr, err := h.q.SumLearningProgress(c.Context(), ids)
	if err != nil {
		return nil, err
	}
	lessons, err := h.lessonCounts(c)
	if err != nil {
		return nil, err
	}
	pairs := make([]pairKey, 0, len(pr))
	for _, p := range pr {
		pairs = append(pairs, pairKey{p.CourseID, uint64(p.UserID.Int64)}) //nolint:gosec // G115: ids
	}
	sums := make(map[pairKey]int64, len(sr))
	for _, s := range sr {
		sums[pairKey{s.CourseID, s.UserID}] = s.PercentSum
	}
	per, _ := aggregate(pairs, sums, lessons)
	return per, nil
}
