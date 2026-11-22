package users_test

// T-M12-13b: DELETE /users/:id (admin) leaves the same state as DELETE /api/v1/account — tokens and refresh tokens
// deleted (not only revoked), the OTP rows of her number, her stored files and her per-user Redis keys gone; the
// control user untouched; the admin audit row kept.

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/platform/cache"
)

func TestDestroy_SameStateAsSelfDelete(t *testing.T) {
	e := admintest.New(t)
	ctx := t.Context()
	h := users.NewHandlers(e.DB, admintest.Quiet).WithStoragePath(e.Storage).WithCache(e.Cache)
	e.Route()(fiber.MethodDelete, "/users/:id", e.Kit.Admin(h.Destroy))

	const her, him uint64 = 40, 41
	const mobileHer, mobileHim = "09120000401", "09120000411"
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES
		(?, 'Her', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00'), (?, 'Him', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`,
		her, mobileHer, him, mobileHim)
	for _, u := range []uint64{her, him} {
		for i := range 2 {
			tok := fmt.Sprintf("t13b-%d-%d", u, i)
			e.Exec(`INSERT INTO oauth_access_tokens (id, user_id, client_id, revoked, created_at, updated_at)
				VALUES (?, ?, 'c', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, tok, u)
			e.Exec(`INSERT INTO oauth_refresh_tokens (id, access_token_id, revoked, expires_at) VALUES (?, ?, 0, NULL)`, "r-"+tok, tok)
		}
	}
	for _, m := range []string{mobileHer, mobileHim} {
		e.Exec(`INSERT INTO otp_verifications (mobile, code, expires_at, attempts, created_at, updated_at)
			VALUES (?, '1234', '2026-09-23 10:05:00', 0, '2026-09-23 10:00:00', '2026-09-23 10:00:00')`, m)
	}
	e.Exec(`INSERT INTO user_profiles (user_id, subscription_type, created_at, updated_at) VALUES (?, 'free', NOW(), NOW()), (?, 'free', NOW(), NOW())`, her, him)

	// Stored files of both.
	vault, err := files.NewVault(e.Storage, files.DevKey("t13b"), nil, false)
	require.NoError(t, err)
	fsvc := files.NewService(files.Options{DB: e.DB, Vault: vault, BaseURL: "https://api.test"})
	pdf := []byte("%PDF-1.4\n1 0 obj << /Ferritin 9 >> endobj\n%%EOF")
	for _, uid := range []uint64{her, him} {
		_, err := fsvc.Put(ctx, uid, files.PurposeRecordDocument, pdf, time.Now())
		require.NoError(t, err)
	}

	// One Redis key per pattern self-delete purges, for both users.
	keys := func(uid, other uint64) []string {
		return []string{
			fmt.Sprintf("cycle-engine:%d:v1:fa", uid),
			fmt.Sprintf("idem:%d:client-key", uid),
			fmt.Sprintf("msg:shown:%d:7:home:2026-09-23", uid),
			fmt.Sprintf("msg:eligible:%d:7:home:2026-09-23", uid),
			fmt.Sprintf("msg:goal:%d:7", uid),
			fmt.Sprintf("seen:%d:2026-09-23", uid),
			fmt.Sprintf("logged:%d:2026-09-23", uid),
			fmt.Sprintf("family:ra:%d:%d:cycle", uid, other), // owner side
			fmt.Sprintf("family:ra:%d:%d:cycle", other, uid), // member side
		}
	}
	keysHer, keysHim := keys(her, 999998), keys(him, 999999)
	for _, k := range append(append([]string{}, keysHer...), keysHim...) {
		require.NoError(t, e.Cache.Set(ctx, k, "1", time.Hour))
	}

	r := e.As(admintest.SuperID).JSON(fiber.MethodDelete, fmt.Sprintf("/users/%d", her), nil)
	require.Equal(t, 200, r.Status, string(r.Raw))
	assert.EqualValues(t, 2, r.Data()["revoked_tokens"], "the live sessions it ended")

	// Rows: hers gone (tokens deleted, not revoked), his untouched.
	assert.Zero(t, e.Int(`SELECT COUNT(*) FROM users WHERE id = ?`, her))
	assert.Zero(t, e.Int(`SELECT COUNT(*) FROM user_profiles WHERE user_id = ?`, her))
	assert.Zero(t, e.Int(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id = ?`, her), "access tokens deleted, not only revoked")
	assert.Zero(t, e.Int(`SELECT COUNT(*) FROM oauth_refresh_tokens WHERE id LIKE ?`, fmt.Sprintf("r-t13b-%d-%%", her)))
	assert.Zero(t, e.Int(`SELECT COUNT(*) FROM otp_verifications WHERE mobile = ?`, mobileHer))
	assert.Equal(t, 2, e.Int(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id = ? AND revoked = 0`, him))
	assert.Equal(t, 2, e.Int(`SELECT COUNT(*) FROM oauth_refresh_tokens WHERE id LIKE ?`, fmt.Sprintf("r-t13b-%d-%%", him)))
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM otp_verifications WHERE mobile = ?`, mobileHim))
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM user_profiles WHERE user_id = ?`, him))

	// Files: hers gone, his kept.
	owners, err := files.Owners(e.Storage, files.Registry[files.PurposeRecordDocument])
	require.NoError(t, err)
	assert.Equal(t, []uint64{him}, owners)

	// Redis: hers gone (both family:ra sides), his kept (except the key naming her as the reading member).
	for _, k := range keysHer {
		_, err := e.Cache.Get(ctx, k)
		assert.ErrorIs(t, err, cache.ErrMiss, k)
	}
	for _, k := range keysHim {
		_, err := e.Cache.Get(ctx, k)
		assert.NoError(t, err, "control key %s", k)
	}

	// The admin audit row is kept.
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'user.delete' AND target_id = ? AND admin_id = ?`,
		her, admintest.SuperID))

	// A second delete is a 404.
	assert.Equal(t, 404, e.As(admintest.SuperID).JSON(fiber.MethodDelete, fmt.Sprintf("/users/%d", her), nil).Status)
}
