package auth

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// User is a users row (App\Models\User) as loaded by the guard. Password and
// remember_token are present but never serialised.
type User = store.User

// Blocked reports whether the admin blocked the account (`$user->blocked_at` is truthy).
func Blocked(u *User) bool { return u != nil && u.BlockedAt.Valid }

// UserJSON is the User model's toArray(): every column in table order except the hidden
// password / remember_token. email_verified_at, mobile_verified_at, created_at and
// updated_at are datetime casts ("2026-09-23T05:30:00.000000Z"); blocked_at has no cast
// and stays the raw DB string ("2026-09-23 09:00:00", Tehran wall-clock).
//
// It returns an ordered map so callers can add relations (e.g. data.user.profile).
func UserJSON(u *User) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", u.ID,
		"name", nullString(u.Name),
		"email", nullString(u.Email),
		"mobile", nullString(u.Mobile),
		"mobile_verified_at", nullDateTime(u.MobileVerifiedAt),
		"blocked_at", rawTimestamp(u.BlockedAt),
		"email_verified_at", nullDateTime(u.EmailVerifiedAt),
		"created_at", nullDateTime(u.CreatedAt),
		"updated_at", nullDateTime(u.UpdatedAt),
	)
}

func nullString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func nullDateTime(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}

func rawTimestamp(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return t.Time.In(civildate.Tehran).Format(time.DateTime)
}

// ProfileQuerier is the one query ProfileCompleted needs (satisfied by *store.Queries).
type ProfileQuerier interface {
	UserHasProfile(ctx context.Context, userID uint64) (bool, error)
}

// ProfileCompleted is User::hasCompletedProfile(): filled($name) && profile()->exists().
func ProfileCompleted(ctx context.Context, q ProfileQuerier, u *User) (bool, error) {
	if !u.Name.Valid || strings.Trim(u.Name.String, " \t\n\r\x00\x0B") == "" {
		return false, nil
	}
	return q.UserHasProfile(ctx, u.ID)
}
