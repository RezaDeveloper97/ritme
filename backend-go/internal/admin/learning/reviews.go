package learning

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func reviewItemJSON(r store.GetLearningReviewLessonRow) *jsonx.OrderedMap {
	m := lessonJSON(lessonFields{
		ID: r.ID, ChapterID: r.ChapterID, Kind: r.Kind, Title: r.Title, Description: r.Description,
		DurationSeconds: r.DurationSeconds, PageCount: r.PageCount, SizeBytes: r.SizeBytes, MediaStatus: r.MediaStatus,
		Status: r.Status, PublishedAt: r.PublishedAt, ReviewStatus: r.ReviewStatus, ReviewedAt: r.ReviewedAt,
		ReviewedBy: r.ReviewedBy, ReviewNote: r.ReviewNote, ReviewedByName: r.ReviewedByName,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
	m.Set("course", jsonx.Obj("id", r.CourseID, "title", r.CourseTitle, "kind", r.CourseKind, "status", r.CourseStatus))
	m.Set("instructor", jsonx.Obj("id", r.InstructorID, "display_name", r.InstructorName, "status", r.InstructorStatus))
	return m
}

// reviewCounts is {open, pending, changed, flagged, approved} over the queue's lessons.
func (h *Handlers) reviewCounts(c fiber.Ctx) (*jsonx.OrderedMap, error) {
	rows, err := h.q.CountLearningLessonsByState(c.Context())
	if err != nil {
		return nil, err
	}
	by := map[string]int64{}
	for _, r := range rows {
		by[r.State] += r.Total
	}
	return jsonx.Obj(
		ReviewOpen, by[ReviewPending]+by[ReviewChanged]+by[ReviewFlagged],
		ReviewPending, by[ReviewPending],
		ReviewChanged, by[ReviewChanged],
		ReviewFlagged, by[ReviewFlagged],
		ReviewApproved, by[ReviewApproved],
	), nil
}

// Reviews is GET /learning/reviews?state=open|pending|changed|flagged|approved|all&page=&per_page= — the content
// review queue: published lessons (plus flagged ones, also after a takedown) in the requested state. open (default)
// = pending + changed + flagged, oldest change first; approved / all newest first.
func (h *Handlers) Reviews(c fiber.Ctx) error {
	state := pick(c.Query("state"), ReviewStates, ReviewOpen)
	want := func(s string) bool {
		return state == s || state == FilterAll || (state == ReviewOpen && s != ReviewApproved)
	}
	ctx := c.Context()
	params := store.CountLearningReviewQueueParams{
		WantFlagged: want(ReviewFlagged), WantPending: want(ReviewPending),
		WantChanged: want(ReviewChanged), WantApproved: want(ReviewApproved),
	}
	total, err := h.q.CountLearningReviewQueue(ctx, params)
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListLearningReviewQueue(ctx, store.ListLearningReviewQueueParams{
		WantFlagged: params.WantFlagged, WantPending: params.WantPending,
		WantChanged: params.WantChanged, WantApproved: params.WantApproved,
		OldestFirst: state != ReviewApproved && state != FilterAll,
		Limit:       int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	counts, err := h.reviewCounts(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, reviewItemJSON(store.GetLearningReviewLessonRow(r)))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("state", state))
	page.Set("counts", counts)
	page.Set("states", ReviewStates)
	return httpadmin.OK(c, page)
}

// ApproveLesson is POST /learning/lessons/:id/approve: the lesson is fine as it is now (clears a flag and its note;
// it stays as published or draft as it is — re-publishing is the instructor's call).
func (h *Handlers) ApproveLesson(c fiber.Ctx) error {
	return h.reviewLesson(c, ActionLessonApprove, false)
}

// FlagLesson is POST /learning/lessons/:id/flag {note}: needs attention (the B-N9-10 safety queue picks flagged
// lessons up); students still see it — use unpublish to take it down.
func (h *Handlers) FlagLesson(c fiber.Ctx) error {
	return h.reviewLesson(c, ActionLessonFlag, true)
}

// UnpublishLesson is POST /learning/lessons/:id/unpublish {note}: a takedown — the lesson goes back to draft
// (students stop seeing it at once) and stays flagged with the reason.
func (h *Handlers) UnpublishLesson(c fiber.Ctx) error {
	return h.reviewLesson(c, ActionLessonUnpublish, true)
}

func (h *Handlers) reviewLesson(c fiber.Ctx, action string, noteRequired bool) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Lesson")
	}
	n, err := note(c, noteRequired)
	if err != nil {
		return err
	}
	_, now := dbNow(c)
	admin := adminID(c)
	var instructorID uint64
	err = h.tx(c, func(q *store.Queries) error {
		cur, err := q.GetLearningLessonForUpdate(c.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("Lesson")
		}
		if err != nil {
			return err
		}
		instructorID = cur.InstructorID
		switch action {
		case ActionLessonApprove:
			err = q.ReviewLearningLesson(c.Context(), store.ReviewLearningLessonParams{
				ReviewStatus: ReviewApproved, Now: now, AdminID: admin, ID: id,
			})
		case ActionLessonFlag:
			err = q.ReviewLearningLesson(c.Context(), store.ReviewLearningLessonParams{
				ReviewStatus: ReviewFlagged, Now: now, AdminID: admin, Note: n, ID: id,
			})
		default:
			err = q.UnpublishLearningLesson(c.Context(), store.UnpublishLearningLessonParams{Now: now, AdminID: admin, Note: n, ID: id})
		}
		if err != nil {
			return err
		}
		return q.InsertLearningModeration(c.Context(), store.InsertLearningModerationParams{
			AdminID: admin, Action: action, TargetType: "lesson", TargetID: id,
			InstructorID: sql.NullInt64{Int64: int64(cur.InstructorID), Valid: true}, //nolint:gosec // G115: ids
			Note:         n, Now: now,
		})
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "learning."+action, "learning_lesson", id, slog.Uint64("instructor_id", instructorID))
	r, err := h.q.GetLearningReviewLesson(c.Context(), id)
	if err != nil {
		return err
	}
	msg := map[string]string{
		ActionLessonApprove: "Lesson approved.", ActionLessonFlag: "Lesson flagged.", ActionLessonUnpublish: "Lesson unpublished.",
	}[action]
	return httpadmin.OK(c, jsonx.Obj("lesson", reviewItemJSON(r)), msg)
}
