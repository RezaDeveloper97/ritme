// Command seed fills a LOCAL scratch database with synthetic users for the k6 load tests (T-M9-14) and writes a
// bearer token per user for the scripts.
//
//	go run ./loadtest/seed -dsn 'ritme:ritme@tcp(127.0.0.1:13317)/ritme_loadtest?parseTime=true' -users 2000 \
//	    -tokens-out loadtest/.tokens.json
//
// It refuses any DSN that is not loopback with a database name containing "test" or "local", and always refuses
// ritme_test and the gt_* databases of the integration tests (guard.go). Re-running is idempotent: users that
// already exist (mobile 0988xxxxxxx) are skipped; -reset deletes them first (FKs cascade).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
)

const (
	batchRows        = 400
	personalClientID = "0199c0de-0000-7000-8000-00000000de01"
)

// tokenRecord is one entry of the -tokens-out file the k6 scripts read.
type tokenRecord struct {
	UserID uint64 `json:"user_id"`
	Mobile string `json:"mobile"`
	Mode   Mode   `json:"mode"`
	Token  string `json:"token"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

func run() error {
	dsn := flag.String("dsn", os.Getenv("LOADTEST_DB_DSN"), "MariaDB DSN of the scratch database (default $LOADTEST_DB_DSN)")
	users := flag.Int("users", 2000, "number of synthetic users")
	storage := flag.String("storage", envOr("STORAGE_PATH", "../backend/storage"), "dir with oauth-private.key (Passport)")
	tokensOut := flag.String("tokens-out", "", "write [{user_id,mobile,mode,token}] here (omit = no tokens)")
	tokensN := flag.Int("tokens-n", 0, "mint tokens for the first N users (0 = all)")
	reset := flag.Bool("reset", false, "delete the synthetic users (and their data) first")
	migrate := flag.Bool("migrate", true, "apply the goose migrations to the scratch database first")
	flag.Parse()

	cfg, err := CheckDSN(*dsn)
	if err != nil {
		return err
	}
	cfg.ParseTime = true
	cfg.Loc = civildate.Tehran
	ctx := context.Background()
	pool, err := platformdb.OpenDSN(ctx, cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()

	if *migrate {
		if err := platformdb.Migrate(ctx, pool); err != nil {
			return err
		}
	}
	if *reset {
		if _, err := pool.ExecContext(ctx, "DELETE FROM users WHERE mobile LIKE ?", MobilePrefix+"%"); err != nil {
			return fmt.Errorf("reset: %w", err)
		}
	}
	now := time.Now().In(civildate.Tehran)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if err := ensurePersonalClient(ctx, pool, now); err != nil {
		return err
	}
	start := time.Now()
	created, err := seedUsers(ctx, pool, *users, today, now)
	if err != nil {
		return err
	}
	log.Printf("seed: %d users in db (%d new) in %s", *users, created, time.Since(start).Round(time.Millisecond)) //nolint:gosec // G706: numbers only

	if *tokensOut != "" {
		n := *tokensN
		if n <= 0 || n > *users {
			n = *users
		}
		return mintTokens(ctx, pool, *storage, n, *tokensOut, now)
	}
	return nil
}

// each runs query and calls fn per row; it closes the rows and reports the iteration error.
func each(ctx context.Context, db *sql.DB, fn func(*sql.Rows) error, query string, args ...any) error {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func ensurePersonalClient(ctx context.Context, db *sql.DB, now time.Time) error {
	_, err := db.ExecContext(ctx, `INSERT IGNORE INTO oauth_clients
		(id,name,provider,redirect_uris,grant_types,revoked,created_at,updated_at)
		VALUES (?,?,?,?,?,0,?,?)`,
		personalClientID, "Ritme Personal Access", "users", "[]", `["personal_access"]`, now, now)
	return err
}

// seedUsers inserts the users that do not exist yet, then their data; it returns how many were new.
func seedUsers(ctx context.Context, db *sql.DB, n int, today, now time.Time) (int, error) {
	have := map[string]bool{}
	err := each(ctx, db, func(r *sql.Rows) error {
		var m string
		if err := r.Scan(&m); err != nil {
			return err
		}
		have[m] = true
		return nil
	}, "SELECT mobile FROM users WHERE mobile LIKE ?", MobilePrefix+"%")
	if err != nil {
		return 0, err
	}

	var fresh []int
	for i := 0; i < n; i++ {
		if !have[Mobile(i)] {
			fresh = append(fresh, i)
		}
	}
	for lo := 0; lo < len(fresh); lo += 200 {
		hi := min(lo+200, len(fresh))
		if err := seedChunk(ctx, db, fresh[lo:hi], today, now); err != nil {
			return 0, err
		}
		log.Printf("seed: %d/%d new users", hi, len(fresh))
	}
	return len(fresh), nil
}

type batch struct {
	table string
	cols  []string
	rows  [][]any
}

func (b *batch) add(v ...any) { b.rows = append(b.rows, v) }

func (b *batch) flush(ctx context.Context, db *sql.DB) error {
	for lo := 0; lo < len(b.rows); lo += batchRows {
		chunk := b.rows[lo:min(lo+batchRows, len(b.rows))]
		ph := "(" + strings.TrimSuffix(strings.Repeat("?,", len(b.cols)), ",") + ")"
		var sb strings.Builder
		sb.WriteString("INSERT INTO `" + b.table + "` (`" + strings.Join(b.cols, "`,`") + "`) VALUES ")
		args := make([]any, 0, len(chunk)*len(b.cols))
		for i, r := range chunk {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(ph)
			args = append(args, r...)
		}
		if _, err := db.ExecContext(ctx, sb.String(), args...); err != nil { //nolint:gosec // G202: table/column names are literals in this file
			return fmt.Errorf("insert %s: %w", b.table, err)
		}
	}
	return nil
}

func dayStr(d Day) string { return d.Format("2006-01-02") }

func dayPtr(d *Day) any {
	if d == nil {
		return nil
	}
	return dayStr(*d)
}

func intPtr(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nilIf(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func seedChunk(ctx context.Context, db *sql.DB, idx []int, today Day, now time.Time) error {
	ub := &batch{table: "users", cols: []string{"name", "mobile", "mobile_verified_at", "created_at", "updated_at"}}
	for _, i := range idx {
		ub.add(fmt.Sprintf("Load User %d", i+1), Mobile(i), now, now, now)
	}
	if err := ub.flush(ctx, db); err != nil {
		return err
	}
	ids := map[string]uint64{}
	err := each(ctx, db, func(r *sql.Rows) error {
		var id uint64
		var m string
		if err := r.Scan(&id, &m); err != nil {
			return err
		}
		ids[m] = id
		return nil
	}, "SELECT id, mobile FROM users WHERE mobile BETWEEN ? AND ?", Mobile(idx[0]), Mobile(idx[len(idx)-1]))
	if err != nil {
		return err
	}

	profiles := &batch{table: "user_profiles", cols: []string{"user_id", "birthday", "weight", "height", "period_duration", "cycle_duration", "last_period_start", "user_goal", "pregnancy_intention", "calculation_status", "created_at", "updated_at"}}
	life := &batch{table: "user_life_profiles", cols: []string{"user_id", "gender", "life_mode", "track_contraception", "onboarding_completed_at", "created_at", "updated_at"}}
	periods := &batch{table: "cycle_histories", cols: []string{"user_id", "period_start_date", "period_end_date", "cycle_length", "bleeding_length", "is_confirmed", "is_estimated", "source", "created_at", "updated_at"}}
	logs := &batch{table: "daily_health_logs", cols: []string{"user_id", "log_date", "bleeding_intensity", "headache_intensity", "energy_level", "weight", "basal_body_temperature", "created_at", "updated_at"}}
	fert := &batch{table: "fertility_logs", cols: []string{"user_id", "log_date", "lh_test", "cervical_mucus", "bbt_time", "created_at", "updated_at"}}
	preg := &batch{table: "pregnancy_profiles", cols: []string{"user_id", "pregnancy_mode", "cycle_mode", "lmp_date", "estimated_due_date", "age_source", "confidence_level", "onboarding_completed", "onboarding_completed_at", "created_at", "updated_at"}}
	post := &batch{table: "postpartum_profiles", cols: []string{"user_id", "birth_date", "delivery_type", "baby_count", "source", "created_at", "updated_at"}}
	contra := &batch{table: "contraception_methods", cols: []string{"user_id", "method", "pack_type", "pack_started_on", "created_at", "updated_at"}}
	pills := &batch{table: "contraception_pill_logs", cols: []string{"user_id", "log_date", "status", "logged_at", "created_at", "updated_at"}}

	for _, i := range idx {
		u := Build(i, today)
		id := ids[u.Mobile]
		goal, intention, mode := "non_ttc", any(nil), any(nil)
		switch u.Mode {
		case ModeTTC:
			goal, intention = "ttc", "trying"
		case ModePostpartum:
			mode = "postpartum"
		}
		profiles.add(id, "1995-05-05", 62.5, 165, u.PeriodLen, u.CycleLen, dayStr(u.LastStart), goal, intention, "completed", now, now)
		life.add(id, "female", mode, u.Mode == ModePill, now, now, now)
		for _, p := range u.Periods {
			periods.add(id, dayStr(p.Start), dayPtr(p.End), intPtr(p.CycleLength), intPtr(p.BleedLength), 1, 0, "user_logged", now, now)
		}
		for _, l := range u.Logs {
			var w, bbt any
			if l.HasWeight {
				w = l.Weight
			}
			if l.BBT > 0 {
				bbt = fmt.Sprintf("%.2f", l.BBT)
			}
			logs.add(id, dayStr(l.Date), nilIf(l.Bleeding), nilIf(l.Headache), nilIf(l.Energy), w, bbt, now, now)
		}
		for _, f := range u.Fertility {
			var at any
			if f.HasBBT {
				at = f.BBTAt
			}
			fert.add(id, dayStr(f.Date), f.LH, f.Mucus, at, now, now)
		}
		if u.LMP != nil {
			preg.add(id, 1, 0, dayStr(*u.LMP), dayPtr(u.DueDate), "lmp", "high", 1, now, now, now)
		}
		if u.BirthDate != nil {
			post.add(id, dayStr(*u.BirthDate), "vaginal", 1, "direct", now, now)
		}
		if u.PillStarted != nil {
			contra.add(id, "combined_pill", "21_7", dayStr(*u.PillStarted), now, now)
			for _, d := range u.PillLogs {
				pills.add(id, dayStr(d), "taken", d.Add(8*time.Hour), now, now)
			}
		}
	}
	for _, b := range []*batch{profiles, life, periods, logs, fert, preg, post, contra, pills} {
		if err := b.flush(ctx, db); err != nil {
			return err
		}
	}
	return nil
}

// mintTokens issues a Passport personal access token per user with the production issuer (same JWT, same
// oauth_access_tokens row as POST /auth/verify-otp), straight from the local DB. The OTP endpoint itself cannot
// serve this: one code per mobile per 60 s and 5 requests per minute per IP.
func mintTokens(ctx context.Context, db *sql.DB, storage string, n int, out string, now time.Time) error {
	keys, err := passport.LoadKeys(storage)
	if err != nil {
		return fmt.Errorf("%w (set -storage / STORAGE_PATH; local keys only)", err)
	}
	issuer := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)

	var recs []tokenRecord
	err = each(ctx, db, func(rows *sql.Rows) error {
		var r tokenRecord
		if err := rows.Scan(&r.UserID, &r.Mobile); err != nil {
			return err
		}
		recs = append(recs, r)
		return nil
	}, "SELECT id, mobile FROM users WHERE mobile LIKE ? ORDER BY mobile LIMIT ?", MobilePrefix+"%", n)
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		return errors.New("no synthetic users to mint tokens for")
	}
	for k := range recs {
		iss, err := issuer.Issue(ctx, recs[k].UserID, now)
		if err != nil {
			return err
		}
		recs[k].Token = iss.AccessToken
		var idx int
		_, _ = fmt.Sscanf(strings.TrimPrefix(recs[k].Mobile, MobilePrefix), "%d", &idx)
		recs[k].Mode = ModeOf(idx - 1)
	}
	b, err := json.Marshal(recs)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, b, 0o600); err != nil {
		return err
	}
	log.Printf("seed: %d tokens → %s (mode 0600, git-ignored; never commit)", len(recs), out)
	return nil
}
