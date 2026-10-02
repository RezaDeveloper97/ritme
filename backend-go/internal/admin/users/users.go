// Package users is user management in the admin API (Admin\UserController): list with
// search and status filter, show with stats, update (name, subscription, goal),
// block / unblock and delete. Blocking and deleting revoke every Passport token of the
// user (auth.RevokeUserTokens); revoking on delete is deviation D-03.
package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/auth"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	companionstore "github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Status filter values (?status=).
const (
	StatusAll     = "all"
	StatusActive  = "active"
	StatusBlocked = "blocked"
)

// Handlers serve /users.
type Handlers struct {
	db          *sql.DB
	q           *store.Queries
	logger      *slog.Logger
	storagePath string // STORAGE_PATH: support-report screenshots are removed with the user (B-N1-12)
}

// NewHandlers wires the handlers.
func NewHandlers(db *sql.DB, logger *slog.Logger) *Handlers {
	return &Handlers{db: db, q: store.New(db), logger: logger}
}

// WithStoragePath sets STORAGE_PATH so Destroy also removes the user's support-report screenshots.
func (h *Handlers) WithStoragePath(p string) *Handlers {
	h.storagePath = p
	return h
}

// escapeLike escapes LIKE wildcards so the search is a plain substring match.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

var (
	nullSentinel = time.Date(1971, 1, 1, 0, 0, 0, 0, civildate.Tehran)
	maxTime      = time.Date(2037, 12, 31, 23, 59, 59, 0, civildate.Tehran) // TIMESTAMP ends 2038-01-19
)

// statusRange maps ?status= onto the blocked_at range of CountUsers / ListUsers
// (NULL blocked_at is the 1971-01-01 sentinel there).
func statusRange(status string) (from, to sql.NullTime) {
	valid := func(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }
	switch status {
	case StatusActive:
		return valid(nullSentinel), valid(nullSentinel)
	case StatusBlocked:
		return valid(nullSentinel.AddDate(0, 0, 1)), valid(maxTime)
	default:
		return valid(nullSentinel), valid(maxTime)
	}
}

// List is GET /users?q=&status=all|active|blocked&page=&per_page=.
func (h *Handlers) List(c fiber.Ctx) error {
	search := strings.TrimSpace(c.Query("q"))
	status := c.Query("status")
	if status != StatusActive && status != StatusBlocked {
		status = StatusAll
	}
	pattern := sql.NullString{String: "%", Valid: true}
	if search != "" {
		pattern.String = "%" + escapeLike(search) + "%"
	}
	from, to := statusRange(status)
	total, err := h.q.CountUsers(c.Context(), store.CountUsersParams{Pattern: pattern, BlockedFrom: from, BlockedTo: to})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListUsers(c.Context(), store.ListUsersParams{
		Pattern: pattern, BlockedFrom: from, BlockedTo: to,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, u := range rows {
		items = append(items, jsonx.Obj(
			"id", u.ID,
			"name", httpadmin.NullString(u.Name),
			"mobile", httpadmin.NullString(u.Mobile),
			"email", httpadmin.NullString(u.Email),
			"is_blocked", u.BlockedAt.Valid,
			"blocked_at", httpadmin.Time(u.BlockedAt),
			"subscription_type", orDefault(u.SubscriptionType, string(enums.SubscriptionTypeFree)),
			"user_goal", orDefault(u.UserGoal, string(enums.UserGoalNonTtc)),
			"created_at", httpadmin.Time(u.CreatedAt),
		))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("q", search, "status", status))
	return httpadmin.OK(c, page)
}

// orDefault is `$user->profile?->x ?? default`.
func orDefault(s sql.NullString, def string) string {
	if s.Valid {
		return s.String
	}
	return def
}

func (h *Handlers) userID(c fiber.Ctx) (uint64, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return 0, httpadmin.NotFound("User")
	}
	return id, nil
}

// Show is GET /users/:id: the user, the profile (null when none), stats and the options
// for the edit form.
func (h *Handlers) Show(c fiber.Ctx) error {
	id, err := h.userID(c)
	if err != nil {
		return err
	}
	body, err := h.detail(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, body)
}

func (h *Handlers) detail(c fiber.Ctx, id uint64) (*jsonx.OrderedMap, error) {
	u, err := h.q.GetUserDetail(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpadmin.NotFound("User")
	}
	if err != nil {
		return nil, err
	}
	stats, err := h.q.UserStats(c.Context(), store.UserStatsParams{UserID: id})
	if err != nil {
		return nil, err
	}
	var profile any
	if u.ProfileID.Valid {
		profile = jsonx.Obj(
			"birthday", nullDate(u.Birthday),
			"last_period_start", nullDate(u.LastPeriodStart),
			"cycle_duration", nullInt(u.CycleDuration),
			"period_duration", nullInt(u.PeriodDuration),
			"subscription_type", httpadmin.NullString(u.SubscriptionType),
			"user_goal", httpadmin.NullString(u.UserGoal),
			"pregnancy_intention", httpadmin.NullString(u.PregnancyIntention),
		)
	}
	locale := i18n.Locale(c)
	return jsonx.Obj(
		"user", jsonx.Obj(
			"id", u.ID,
			"name", httpadmin.NullString(u.Name),
			"mobile", httpadmin.NullString(u.Mobile),
			"email", httpadmin.NullString(u.Email),
			"is_blocked", u.BlockedAt.Valid,
			"blocked_at", httpadmin.Time(u.BlockedAt),
			"mobile_verified_at", httpadmin.Time(u.MobileVerifiedAt),
			"subscription_type", orDefault(u.SubscriptionType, string(enums.SubscriptionTypeFree)),
			"user_goal", orDefault(u.UserGoal, string(enums.UserGoalNonTtc)),
			"created_at", httpadmin.Time(u.CreatedAt),
			"updated_at", httpadmin.Time(u.UpdatedAt),
		),
		"profile", profile,
		"stats", jsonx.Obj(
			"health_logs", stats.HealthLogs,
			"reminders", stats.Reminders,
			"notifications", stats.Notifications,
		),
		"options", jsonx.Obj(
			"subscription_types", enums.SubscriptionTypeOptions(locale),
			"user_goals", enums.UserGoalOptions(locale),
		),
	), nil
}

func nullDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date
}

func nullInt(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return n.Int16
}

// Update is PUT /users/:id {name, subscription_type, user_goal}. Subscription and goal
// live on the profile, which is created on demand (updateOrCreate).
func (h *Handlers) Update(c fiber.Ctx) error {
	id, err := h.userID(c)
	if err != nil {
		return err
	}
	if err := h.mustExist(c.Context(), id); err != nil {
		return err
	}
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("name", "nullable|string|max:255"),
		validation.F("subscription_type", "required", validation.In(enums.SubscriptionTypeValues()...)),
		validation.F("user_goal", "required", validation.In(enums.UserGoalValues()...)),
	})
	if err != nil {
		return err
	}
	// name: absent = unchanged, null/"" = cleared (ConvertEmptyStringsToNull).
	nameValue, setName := data.Get("name")
	name := sql.NullString{}
	if setName && nameValue != nil {
		name = sql.NullString{String: httpadmin.String(data, "name"), Valid: true}
	}
	now := httpadmin.DBTime(httpadmin.Now(c))
	sub, goal := httpadmin.String(data, "subscription_type"), httpadmin.String(data, "user_goal")
	err = h.inTx(c.Context(), func(q *store.Queries, _ *sql.Tx) error {
		if setName {
			if err := q.UpdateUserName(c.Context(), store.UpdateUserNameParams{Name: name, Now: now, ID: id}); err != nil {
				return err
			}
		}
		pid, err := q.FirstProfileID(c.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			return q.CreateProfilePlan(c.Context(), store.CreateProfilePlanParams{
				UserID: id, SubscriptionType: sub, UserGoal: goal, Now: now})
		}
		if err != nil {
			return err
		}
		return q.UpdateProfilePlan(c.Context(), store.UpdateProfilePlanParams{
			SubscriptionType: sub, UserGoal: goal, Now: now, ID: pid})
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "user.update", "user", id,
		slog.String("subscription_type", sub), slog.String("user_goal", goal))
	body, err := h.detail(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, body, "User updated.")
}

// Block is POST /users/:id/block: sets blocked_at and revokes every token, so the block
// takes effect on the user's next request.
func (h *Handlers) Block(c fiber.Ctx) error {
	return h.setBlocked(c, true)
}

// Unblock is POST /users/:id/unblock (tokens stay revoked; the user logs in again).
func (h *Handlers) Unblock(c fiber.Ctx) error {
	return h.setBlocked(c, false)
}

func (h *Handlers) setBlocked(c fiber.Ctx, blocked bool) error {
	id, err := h.userID(c)
	if err != nil {
		return err
	}
	if err := h.mustExist(c.Context(), id); err != nil {
		return err
	}
	nowTime := httpadmin.Now(c)
	now := httpadmin.DBTime(nowTime)
	blockedAt := sql.NullTime{}
	if blocked {
		blockedAt = now
	}
	var revoked int64
	err = h.inTx(c.Context(), func(q *store.Queries, tx *sql.Tx) error {
		if err := q.SetUserBlockedAt(c.Context(), store.SetUserBlockedAtParams{BlockedAt: blockedAt, Now: now, ID: id}); err != nil {
			return err
		}
		if !blocked {
			return nil
		}
		n, err := auth.RevokeUserTokens(c.Context(), authstore.New(tx), id, nowTime)
		revoked = n
		return err
	})
	if err != nil {
		return err
	}
	action, msg := "user.unblock", "User unblocked."
	if blocked {
		action, msg = "user.block", "User blocked."
	}
	httpadmin.Audit(c, h.logger, action, "user", id, slog.Int64("revoked_tokens", revoked))
	return httpadmin.OK(c, jsonx.Obj(
		"id", id,
		"is_blocked", blocked,
		"blocked_at", httpadmin.Time(blockedAt),
		"revoked_tokens", revoked,
	), msg)
}

// Destroy is DELETE /users/:id. Tokens are revoked first (D-03: Laravel only deletes the
// row; oauth_access_tokens has no foreign key, so the rows would stay unrevoked).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	id, err := h.userID(c)
	if err != nil {
		return err
	}
	nowTime := httpadmin.Now(c)
	// The files outlive the rows (support_reports goes by ON DELETE CASCADE): list them first.
	shots, err := profilestore.New(h.db).ListUserSupportScreenshots(c.Context(), id)
	if err != nil {
		return fmt.Errorf("users: support screenshots: %w", err)
	}
	var revoked int64
	err = h.inTx(c.Context(), func(q *store.Queries, tx *sql.Tx) error {
		n, err := auth.RevokeUserTokens(c.Context(), authstore.New(tx), id, nowTime)
		if err != nil {
			return err
		}
		revoked = n
		// CB-LOSS-01: the pregnancy notices her companions got (their inboxes, no FK to her).
		if err := companionstore.New(tx).DeletePregnancyNoticesForOwner(c.Context(), id); err != nil {
			return err
		}
		res, err := q.DeleteUser(c.Context(), id)
		if err != nil {
			return err
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return httpadmin.NotFound("User")
		}
		return nil
	})
	if err != nil {
		return err
	}
	profile.RemoveSupportFiles(h.storagePath, shots, h.logger)
	httpadmin.Audit(c, h.logger, "user.delete", "user", id, slog.Int64("revoked_tokens", revoked))
	return httpadmin.OK(c, jsonx.Obj("id", id, "revoked_tokens", revoked), "User deleted.")
}

func (h *Handlers) mustExist(ctx context.Context, id uint64) error {
	found, err := h.q.UserExists(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return httpadmin.NotFound("User")
	}
	return nil
}

func (h *Handlers) inTx(ctx context.Context, fn func(q *store.Queries, tx *sql.Tx) error) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("users: begin: %w", err)
	}
	if err := fn(h.q.WithTx(tx), tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("users: commit: %w", err)
	}
	return nil
}
