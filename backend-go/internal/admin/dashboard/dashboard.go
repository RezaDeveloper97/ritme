// Package dashboard is GET /dashboard of the admin API (Admin\DashboardController): the
// headline counts and the most recent sign-ups.
package dashboard

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// RecentUsersLimit is User::latest()->take(8).
const RecentUsersLimit = 8

// Querier is what the dashboard reads.
type Querier interface {
	DashboardCounts(ctx context.Context, arg store.DashboardCountsParams) (store.DashboardCountsRow, error)
	RecentUsers(ctx context.Context, limit int32) ([]store.RecentUsersRow, error)
}

// Handlers serve the dashboard.
type Handlers struct{ q Querier }

// NewHandlers wires the handlers.
func NewHandlers(q Querier) *Handlers { return &Handlers{q: q} }

// Show is GET /dashboard.
func (h *Handlers) Show(c fiber.Ctx) error {
	now := httpadmin.Now(c).In(civildate.Tehran)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, civildate.Tehran)
	counts, err := h.q.DashboardCounts(c.Context(), store.DashboardCountsParams{
		WeekStart:  httpadmin.DBTime(now.AddDate(0, 0, -7)), // now()->subDays(7)
		TodayStart: httpadmin.DBTime(today),                 // now()->startOfDay()
	})
	if err != nil {
		return err
	}
	recent, err := h.q.RecentUsers(c.Context(), RecentUsersLimit)
	if err != nil {
		return err
	}
	users := make([]*jsonx.OrderedMap, 0, len(recent))
	for _, u := range recent {
		users = append(users, jsonx.Obj(
			"id", u.ID,
			"name", httpadmin.NullString(u.Name),
			"mobile", httpadmin.NullString(u.Mobile),
			"email", httpadmin.NullString(u.Email),
			"is_blocked", u.BlockedAt.Valid,
			"blocked_at", httpadmin.Time(u.BlockedAt),
			"created_at", httpadmin.Time(u.CreatedAt),
		))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"stats", jsonx.Obj(
			"users", counts.Users,
			"users_blocked", counts.UsersBlocked,
			"users_new_week", counts.UsersNewWeek,
			"users_new_today", counts.UsersNewToday,
			"articles", counts.Articles,
			"affirmations", counts.Affirmations,
			"challenges", counts.Challenges,
			"task_templates", counts.TaskTemplates,
			"messages", counts.Messages,
			"messages_pending", counts.MessagesPending,
		),
		"recent_users", users,
	))
}
