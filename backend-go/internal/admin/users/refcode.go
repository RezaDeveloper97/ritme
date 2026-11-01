package users

// Reference codes (T-M10-32, A-09; gap 06 §2.5): the support-facing pseudonym of a user is `RT-` + the first 6 base32
// characters of HMAC-SHA256(user id, ADMIN_REF_SECRET). Without the secret the code cannot be derived from the id or
// enumerated (the old `RT-%04X` was the id in hex). Stored in users.ref_code (UNIQUE) for search; on a collision the
// code grows to 7, then 8 characters. Written at signup (auth signup hook) and by the `admin.ref_code_backfill` job.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"

	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// RefSecretEnv is the environment variable of the HMAC secret.
const RefSecretEnv = "ADMIN_REF_SECRET"

// devRefSecret is the fixed secret of APP_ENV local / testing only (codes there are not private).
const devRefSecret = "ritme-dev-ref-secret" //nolint:gosec // G101: not a credential, local/testing only

// Reference code lengths (A-09): 6 characters, extended on a unique collision.
const (
	refCodeMin = 6
	refCodeMax = 8
)

// ErrRefSecretMissing is returned outside local/testing without ADMIN_REF_SECRET: no code is ever made from a guessable
// secret (fail closed).
var ErrRefSecretMissing = errors.New("users: " + RefSecretEnv + " is required outside local/testing")

// errRefCollision is a code that collided at every length (practically impossible).
var errRefCollision = errors.New("users: reference code collided at every length")

// RefCoder makes and stores reference codes.
type RefCoder struct {
	secret []byte
	db     *sql.DB
	log    *slog.Logger
}

// NewRefCoder returns the coder for appEnv. An empty secret is allowed only in local / testing (a fixed dev secret).
func NewRefCoder(db *sql.DB, appEnv, secret string, logger *slog.Logger) (*RefCoder, error) {
	if secret == "" {
		if appEnv != "local" && appEnv != "testing" {
			return nil, ErrRefSecretMissing
		}
		secret = devRefSecret
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &RefCoder{secret: []byte(secret), db: db, log: logger}, nil
}

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// Code is the reference code of id with n base32 characters (deterministic for one secret).
func (r *RefCoder) Code(id uint64, n int) string {
	m := hmac.New(sha256.New, r.secret)
	m.Write([]byte(strconv.FormatUint(id, 10)))
	return "RT-" + b32.EncodeToString(m.Sum(nil))[:n]
}

// Assign stores id's code when it has none (6 characters, 7 then 8 on a unique collision). Idempotent: a user that
// already has a code keeps it.
func (r *RefCoder) Assign(ctx context.Context, id uint64) error {
	q := authstore.New(r.db)
	for n := refCodeMin; n <= refCodeMax; n++ {
		_, err := q.SetUserRefCode(ctx, authstore.SetUserRefCodeParams{
			RefCode: sql.NullString{String: r.Code(id, n), Valid: true}, ID: id})
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			continue
		}
		if err != nil {
			return fmt.Errorf("users: ref code: %w", err)
		}
		return nil
	}
	return errRefCollision
}

// SignupHook is the auth signup callback (auth.Handlers.OnSignup): the new account gets its code right away.
func (r *RefCoder) SignupHook() func(ctx context.Context, userID uint64, _ string, _ time.Time) error {
	return func(ctx context.Context, userID uint64, _ string, _ time.Time) error { return r.Assign(ctx, userID) }
}

// Backfill job (docs/engine-v6/jobs.md).
const (
	JobRefCodeBackfill  = "admin.ref_code_backfill"
	RefCodeBackfillCron = "35 4 * * *"
	backfillPage        = 1000
)

// Backfill gives every user without a code one, by id pages. A no-op (one query) once every user has a code.
func (r *RefCoder) Backfill(ctx context.Context) error {
	q := authstore.New(r.db)
	var after uint64
	done := 0
	for {
		ids, err := q.ListUserIDsWithoutRefCode(ctx, authstore.ListUserIDsWithoutRefCodeParams{AfterID: after, Limit: backfillPage})
		if err != nil {
			return fmt.Errorf("users: ref code backfill: %w", err)
		}
		for _, id := range ids {
			if err := r.Assign(ctx, id); err != nil {
				return err
			}
			done++
		}
		if len(ids) < backfillPage {
			if done > 0 {
				r.log.InfoContext(ctx, "admin ref codes backfilled", slog.Int("users", done))
			}
			return nil
		}
		after = ids[len(ids)-1]
	}
}

// RegisterJobs puts the backfill on q: once when a process becomes scheduler leader and nightly afterwards (catches a
// signup whose hook failed).
func (r *RefCoder) RegisterJobs(q *queue.Queue) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobRefCodeBackfill, Priority: queue.Low, MaxTries: 1, Timeout: 30 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { return r.Backfill(ctx) }, RefCodeBackfillCron,
		queue.Unique(23*time.Hour), queue.RunAtStart())
}
