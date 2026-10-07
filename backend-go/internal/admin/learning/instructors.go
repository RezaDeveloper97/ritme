package learning

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/billing"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	domain "github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// InstructorStatuses is the ?status= filter of the instructors list.
var InstructorStatuses = []string{FilterAll, domain.InstructorPending, domain.InstructorApproved, domain.InstructorRevoked}

func instructorJSON(r store.GetLearningInstructorAdminRow) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"display_name", r.DisplayName,
		"title", httpadmin.NullString(r.Title),
		"bio", httpadmin.NullString(r.Bio),
		"status", r.Status,
		// The account behind the profile: name + masked mobile (the full number stays on /users/:id).
		"user", jsonx.Obj("id", r.UserID, "name", httpadmin.NullString(r.UserName), "mobile", billing.MaskMobile(r.UserMobile)),
		"courses_count", r.CoursesCount,
		"published_courses", r.PublishedCourses,
		"students_count", r.StudentsCount,
		"approved_at", httpadmin.Time(r.ApprovedAt),
		"approved_by", adminRef(r.ApprovedBy, r.ApprovedByName),
		"revoked_at", httpadmin.Time(r.RevokedAt),
		"created_at", httpadmin.Time(r.CreatedAt),
		"updated_at", httpadmin.Time(r.UpdatedAt),
	)
}

// Instructors is GET /learning/instructors?status=all|pending|approved|revoked&q=&page=&per_page= — pending first,
// then newest; q matches the display name, the account name or part of the mobile.
func (h *Handlers) Instructors(c fiber.Ctx) error {
	status := pick(c.Query("status"), InstructorStatuses, FilterAll)
	q, like := search(c)
	_, now := dbNow(c)
	ctx := c.Context()
	total, err := h.q.CountLearningInstructors(ctx, store.CountLearningInstructorsParams{Status: pattern(status), Q: q, QLike: like})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListLearningInstructors(ctx, store.ListLearningInstructorsParams{
		Now: now, Status: pattern(status), Q: q, QLike: like,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	counts, err := h.q.CountLearningInstructorsByStatus(ctx)
	if err != nil {
		return err
	}
	by := map[string]int64{}
	for _, r := range counts {
		by[r.Status] = r.Total
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, instructorJSON(store.GetLearningInstructorAdminRow(r)))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("status", status, "q", q))
	page.Set("counts", countsOf(by, domain.InstructorPending, domain.InstructorApproved, domain.InstructorRevoked))
	page.Set("statuses", InstructorStatuses)
	return httpadmin.OK(c, page)
}

func (h *Handlers) findInstructor(c fiber.Ctx) (store.GetLearningInstructorAdminRow, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.GetLearningInstructorAdminRow{}, httpadmin.NotFound("Instructor")
	}
	_, now := dbNow(c)
	r, err := h.q.GetLearningInstructorAdmin(c.Context(), store.GetLearningInstructorAdminParams{Now: now, ID: id})
	if errors.Is(err, sql.ErrNoRows) {
		return r, httpadmin.NotFound("Instructor")
	}
	return r, err
}

func historyJSON(rows []store.ListLearningModerationRow) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, m := range rows {
		out = append(out, jsonx.Obj(
			"id", m.ID, "action", m.Action, "target_type", m.TargetType, "target_id", m.TargetID,
			"admin", adminRef(m.AdminID, m.AdminName), "note", httpadmin.NullString(m.Note),
			"created_at", httpadmin.Time(m.CreatedAt),
		))
	}
	return out
}

// Instructor is GET /learning/instructors/:id: the profile and the last 20 moderation actions about her (status
// changes and reviews of her lessons).
func (h *Handlers) Instructor(c fiber.Ctx) error {
	r, err := h.findInstructor(c)
	if err != nil {
		return err
	}
	hist, err := h.q.ListLearningModeration(c.Context(), store.ListLearningModerationParams{
		TargetID: r.ID, InstructorID: sql.NullInt64{Int64: int64(r.ID), Valid: true}, //nolint:gosec // G115: ids
	})
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("instructor", instructorJSON(r), "history", historyJSON(hist)))
}

// ApproveInstructor is POST /learning/instructors/:id/approve {note?}: pending or revoked → approved (she reaches
// /api/instructor/v1 and her published courses are shown to students again). Approving an approved instructor
// changes nothing (200, no log row).
func (h *Handlers) ApproveInstructor(c fiber.Ctx) error {
	return h.setInstructor(c, domain.InstructorApproved)
}

// RevokeInstructor is POST /learning/instructors/:id/revoke {note?}: approved or pending → revoked (rejecting an
// application is a revoke). Her panel answers 403 instructor_required and students stop seeing her courses; nothing
// is deleted.
func (h *Handlers) RevokeInstructor(c fiber.Ctx) error {
	return h.setInstructor(c, domain.InstructorRevoked)
}

func (h *Handlers) setInstructor(c fiber.Ctx, to string) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Instructor")
	}
	n, err := note(c, false)
	if err != nil {
		return err
	}
	_, now := dbNow(c)
	admin := adminID(c)
	changed := false
	err = h.tx(c, func(q *store.Queries) error {
		cur, err := q.GetLearningInstructorForUpdate(c.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("Instructor")
		}
		if err != nil || cur.Status == to {
			return err
		}
		changed = true
		action := ActionInstructorApprove
		if to == domain.InstructorApproved {
			err = q.ApproveLearningInstructor(c.Context(), store.ApproveLearningInstructorParams{Now: now, AdminID: admin, ID: id})
		} else {
			action = ActionInstructorRevoke
			err = q.RevokeLearningInstructor(c.Context(), store.RevokeLearningInstructorParams{Now: now, ID: id})
		}
		if err != nil {
			return err
		}
		return q.InsertLearningModeration(c.Context(), store.InsertLearningModerationParams{
			AdminID: admin, Action: action, TargetType: "instructor", TargetID: id,
			InstructorID: sql.NullInt64{Int64: int64(id), Valid: true}, //nolint:gosec // G115: ids
			Note:         n, Now: now,
		})
	})
	if err != nil {
		return err
	}
	msg := "Instructor approved."
	action := ActionInstructorApprove
	if to == domain.InstructorRevoked {
		msg, action = "Instructor revoked.", ActionInstructorRevoke
	}
	if changed {
		httpadmin.Audit(c, h.logger, "learning."+action, "learning_instructor", id, slog.String("status", to))
	} else {
		msg = "No change."
	}
	r, err := h.findInstructor(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("instructor", instructorJSON(r), "changed", changed), msg)
}
