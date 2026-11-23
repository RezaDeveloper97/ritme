package profile_test

// T-M12-13 (finding T-03): account deletion removes every user row, file and Redis key — proved from the live schema,
// not from a hand-kept table list. The allow-list is docs/quality-2026-11/user-tables.md.

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	cyclecache "github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/idempotency"
	"github.com/ritme/backend-go/internal/profile"
)

const userTablesDoc = "../../../docs/quality-2026-11/user-tables.md"

// ---------------------------------------------------------------------------------------------------------------
// Schema metadata

type colMeta struct {
	name, dataType, colType string
	nullable, hasDefault    bool
	autoInc, generated      bool
	maxLen                  int64
}

type fkMeta struct{ refTable, refCol, rule string }

type schemaMeta struct {
	tables []string
	cols   map[string][]colMeta
	fks    map[string]map[string]fkMeta // table → column → FK
	pk     map[string][]string
	json   map[string]bool // "table.column" with CHECK (json_valid(col))
}

// eachRow runs q and calls scan for every row.
func eachRow(t *testing.T, db *sql.DB, q string, scan func(*sql.Rows)) {
	t.Helper()
	rows, err := db.Query(q)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		scan(rows)
	}
	require.NoError(t, rows.Err())
}

func loadSchema(t *testing.T, db *sql.DB) *schemaMeta {
	t.Helper()
	s := &schemaMeta{cols: map[string][]colMeta{}, fks: map[string]map[string]fkMeta{}, pk: map[string][]string{}, json: map[string]bool{}}
	eachRow(t, db, `SELECT c.table_name, c.column_name, c.data_type, c.column_type, c.is_nullable = 'YES',
			c.column_default IS NOT NULL, c.extra, COALESCE(c.character_maximum_length, 0)
		FROM information_schema.columns c
		JOIN information_schema.tables t ON t.table_schema = c.table_schema AND t.table_name = c.table_name AND t.table_type = 'BASE TABLE'
		WHERE c.table_schema = DATABASE() ORDER BY c.table_name, c.ordinal_position`, func(r *sql.Rows) {
		var tbl, extra string
		var c colMeta
		require.NoError(t, r.Scan(&tbl, &c.name, &c.dataType, &c.colType, &c.nullable, &c.hasDefault, &extra, &c.maxLen))
		c.autoInc = strings.Contains(extra, "auto_increment")
		c.generated = strings.Contains(extra, "GENERATED")
		if _, ok := s.cols[tbl]; !ok {
			s.tables = append(s.tables, tbl)
		}
		s.cols[tbl] = append(s.cols[tbl], c)
	})

	width := map[string]int{} // table/constraint → columns
	eachRow(t, db, `SELECT k.table_name, k.constraint_name, k.column_name, k.referenced_table_name, k.referenced_column_name, r.delete_rule
		FROM information_schema.key_column_usage k
		JOIN information_schema.referential_constraints r ON r.constraint_schema = k.constraint_schema
			AND r.constraint_name = k.constraint_name AND r.table_name = k.table_name
		WHERE k.table_schema = DATABASE() AND k.referenced_table_name IS NOT NULL`, func(r *sql.Rows) {
		var tbl, name, col string
		var fk fkMeta
		require.NoError(t, r.Scan(&tbl, &name, &col, &fk.refTable, &fk.refCol, &fk.rule))
		width[tbl+"/"+name]++
		require.Equal(t, 1, width[tbl+"/"+name], "composite FK %s on %s: teach the seeder", name, tbl)
		if s.fks[tbl] == nil {
			s.fks[tbl] = map[string]fkMeta{}
		}
		s.fks[tbl][col] = fk
	})

	eachRow(t, db, `SELECT table_name, column_name FROM information_schema.key_column_usage
		WHERE table_schema = DATABASE() AND constraint_name = 'PRIMARY' ORDER BY table_name, ordinal_position`, func(r *sql.Rows) {
		var tbl, col string
		require.NoError(t, r.Scan(&tbl, &col))
		s.pk[tbl] = append(s.pk[tbl], col)
	})

	jsonRe := regexp.MustCompile("^json_valid\\(`([^`]+)`\\)$")
	eachRow(t, db, `SELECT table_name, check_clause FROM information_schema.check_constraints WHERE constraint_schema = DATABASE()`, func(r *sql.Rows) {
		var tbl, clause string
		require.NoError(t, r.Scan(&tbl, &clause))
		if m := jsonRe.FindStringSubmatch(clause); m != nil {
			s.json[tbl+"."+m[1]] = true
		}
	})
	return s
}

// userColumnName is the task's naming rule for a column that holds a user id.
func userColumnName(col string) bool {
	switch col {
	case "user_id", "owner_id", "member_id", "child_owner_id":
		return true
	}
	return strings.HasSuffix(col, "_user_id")
}

// userRef: the column holds a user id — by name, or by a foreign key to users.
func (s *schemaMeta) userRef(tbl, col string) bool {
	if fk, ok := s.fks[tbl][col]; ok {
		if fk.refTable == "users" {
			return true
		}
		return false // e.g. insurance_claims.member_id → insurance_policy_members
	}
	return userColumnName(col)
}

// ---------------------------------------------------------------------------------------------------------------
// The allow-list (parsed from the doc, the single source)

type allowEntry struct{ kind, reason string }

func loadAllowList(t *testing.T) map[string]allowEntry {
	t.Helper()
	f, err := os.Open(userTablesDoc)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	row := regexp.MustCompile("^\\| `([a-z0-9_]+\\.[a-z0-9_]+)` \\| ([a-z-]+) \\| ([^|]+) \\|")
	out := map[string]allowEntry{}
	inList := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "## ") {
			inList = line == "## Allow-list"
			continue
		}
		if m := row.FindStringSubmatch(line); inList && m != nil {
			out[m[1]] = allowEntry{kind: m[2], reason: strings.TrimSpace(m[3])}
		}
	}
	require.NoError(t, sc.Err())
	require.NotEmpty(t, out, "allow-list table not found in %s", userTablesDoc)
	return out
}

var allowKinds = map[string]bool{"deleted-by-code": true, "same-row": true, "anonymised": true, "kept": true, "pending-decision": true}

// userColumnViolations is the schema rule: a user-named column must cascade from users or be allow-listed.
func userColumnViolations(s *schemaMeta, allow map[string]allowEntry) []string {
	var out []string
	seen := map[string]bool{}
	for _, tbl := range s.tables {
		for _, c := range s.cols[tbl] {
			if !userColumnName(c.name) {
				continue
			}
			key := tbl + "." + c.name
			seen[key] = true
			fk, hasFK := s.fks[tbl][c.name]
			cascades := hasFK && fk.refTable == "users" && fk.rule == "CASCADE"
			_, listed := allow[key]
			switch {
			case cascades && listed:
				out = append(out, key+": cascades from users now — drop it from the allow-list")
			case !cascades && !listed:
				what := "no foreign key"
				if hasFK {
					what = fmt.Sprintf("FK → %s ON DELETE %s", fk.refTable, fk.rule)
				}
				out = append(out, key+": "+what+" — add ON DELETE CASCADE to users or an allow-list row")
			}
		}
	}
	for key, e := range allow {
		if !seen[key] {
			out = append(out, key+": allow-listed but no such user column — stale entry")
		}
		if !allowKinds[e.kind] {
			out = append(out, key+": unknown kind "+e.kind)
		}
		if e.reason == "" {
			out = append(out, key+": no reason")
		}
	}
	sort.Strings(out)
	return out
}

// Test 1: every user column cascades from users or is on the documented allow-list.
func TestUserColumns_CascadeOrAllowListed(t *testing.T) {
	db := testdb.New(t)
	allow := loadAllowList(t)
	assert.Empty(t, userColumnViolations(loadSchema(t, db), allow))

	// Boundary cases on a probe table: the checker really flags a missing FK, a SET NULL FK and a stale entry, and
	// leaves alone a name outside the rule.
	t.Run("probe", func(t *testing.T) {
		exec := func(q string) {
			_, err := db.Exec(q)
			require.NoError(t, err, q)
		}
		exec("DROP TABLE IF EXISTS `t13_probe`")
		t.Cleanup(func() { _, _ = db.Exec("DROP TABLE IF EXISTS `t13_probe`") })
		exec("CREATE TABLE `t13_probe` (`id` bigint unsigned PRIMARY KEY AUTO_INCREMENT, `user_id` bigint unsigned NOT NULL, " +
			"`buyer_user_id` bigint unsigned NULL, `user_identifier` bigint unsigned NULL) ENGINE=InnoDB")
		got := userColumnViolations(loadSchema(t, db), allow)
		assert.Contains(t, got, "t13_probe.user_id: no foreign key — add ON DELETE CASCADE to users or an allow-list row")
		assert.Contains(t, got, "t13_probe.buyer_user_id: no foreign key — add ON DELETE CASCADE to users or an allow-list row")
		for _, v := range got {
			assert.NotContains(t, v, "user_identifier", "not a user column by the naming rule")
		}

		exec("ALTER TABLE `t13_probe` ADD CONSTRAINT `t13_probe_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE, " +
			"ADD CONSTRAINT `t13_probe_buyer_user_id_foreign` FOREIGN KEY (`buyer_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL")
		got = userColumnViolations(loadSchema(t, db), allow)
		assert.NotContains(t, strings.Join(got, "\n"), "t13_probe.user_id")
		assert.Contains(t, got, "t13_probe.buyer_user_id: FK → users ON DELETE SET NULL — add ON DELETE CASCADE to users or an allow-list row")

		withStale := map[string]allowEntry{"t13_probe.user_id": {kind: "kept", reason: "x"}, "nope.user_id": {kind: "kept", reason: "x"}}
		for k, v := range allow {
			withStale[k] = v
		}
		got = userColumnViolations(loadSchema(t, db), withStale)
		assert.Contains(t, got, "t13_probe.user_id: cascades from users now — drop it from the allow-list")
		assert.Contains(t, got, "nope.user_id: allow-listed but no such user column — stale entry")
	})
}

// ---------------------------------------------------------------------------------------------------------------
// The generic seeder: one valid row per table, foreign keys followed to real parent rows.

type seededRow struct {
	table   string
	owner   uint64 // the user the row was seeded for (0 = shared catalog row)
	mixed   bool   // a link row: one user column = the deleted user, the others another user
	vals    map[string]any
	parents map[string]*seededRow // FK column → parent row
}

type seeder struct {
	t      *testing.T
	db     *sql.DB
	s      *schemaMeta
	scope  map[string]bool // user tables: a user column, or a FK to a user table (transitively)
	n      int
	memo   map[string]*seededRow // owner/table → row
	active map[string]bool
	rows   []*seededRow
}

func newSeeder(t *testing.T, db *sql.DB, s *schemaMeta) *seeder {
	sd := &seeder{t: t, db: db, s: s, scope: map[string]bool{}, memo: map[string]*seededRow{}, active: map[string]bool{}}
	for _, tbl := range s.tables {
		if tbl == "users" {
			continue
		}
		for _, c := range s.cols[tbl] {
			if s.userRef(tbl, c.name) {
				sd.scope[tbl] = true
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, tbl := range s.tables {
			if sd.scope[tbl] || tbl == "users" {
				continue
			}
			for _, fk := range s.fks[tbl] {
				if sd.scope[fk.refTable] {
					sd.scope[tbl], changed = true, true
					break
				}
			}
		}
	}
	return sd
}

func (sd *seeder) value(tbl string, c colMeta, attempt int) any {
	sd.n++
	n := sd.n
	switch c.dataType {
	case "tinyint", "smallint", "mediumint", "int", "bigint", "decimal", "float", "double":
		return 1 + attempt
	case "date":
		return "2026-09-20"
	case "datetime", "timestamp":
		return "2026-09-20 10:00:00"
	case "time":
		return "10:00:00"
	case "year":
		return 2026
	case "enum", "set":
		inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(c.colType, "enum("), "set("), ")")
		return strings.Trim(strings.SplitN(inner, ",", 2)[0], "'")
	}
	if sd.s.json[tbl+"."+c.name] {
		return "{}"
	}
	v := fmt.Sprintf("%dt13%d", n, attempt)
	if c.maxLen > 0 && int64(len(v)) > c.maxLen {
		v = v[:c.maxLen]
	}
	return v
}

// row seeds one row of tbl. users(col) names the user id for each user column (nil → the column is left out).
func (sd *seeder) row(tbl string, owner uint64, mixed bool, users func(col string) any) *seededRow {
	t := sd.t
	t.Helper()
	r := &seededRow{table: tbl, owner: owner, mixed: mixed, parents: map[string]*seededRow{}}
	for attempt := 0; attempt < 8; attempt++ {
		r.vals = map[string]any{}
		var cols []string
		var args []any
		for _, c := range sd.s.cols[tbl] {
			if c.autoInc || c.generated {
				continue
			}
			var v any
			fk, hasFK := sd.s.fks[tbl][c.name]
			switch {
			case sd.s.userRef(tbl, c.name):
				v = users(c.name)
			case hasFK && fk.refTable == tbl: // self reference: leave empty
				require.True(t, c.nullable, "NOT NULL self reference %s.%s", tbl, c.name)
			case hasFK && (!c.nullable || sd.scope[fk.refTable]):
				pOwner := owner
				if !sd.scope[fk.refTable] {
					pOwner = 0
				}
				p := sd.ensure(fk.refTable, pOwner)
				if p == nil { // a cycle through a nullable column
					require.True(t, c.nullable, "FK cycle at %s.%s", tbl, c.name)
					continue
				}
				r.parents[c.name] = p
				v = p.vals[fk.refCol]
			case c.nullable || c.hasDefault:
				continue
			default:
				v = sd.value(tbl, c, attempt)
			}
			if v == nil {
				continue
			}
			cols = append(cols, "`"+c.name+"`")
			args = append(args, v)
			r.vals[c.name] = v
		}
		q := "INSERT INTO `" + tbl + "` (" + strings.Join(cols, ", ") + ") VALUES (" +
			strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ") + ")"
		if len(cols) == 0 {
			q = "INSERT INTO `" + tbl + "` () VALUES ()"
		}
		res, err := sd.db.Exec(q, args...)
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			continue // a unique key: try other values
		}
		require.NoError(t, err, "seed %s", tbl)
		for _, c := range sd.s.cols[tbl] {
			if c.autoInc {
				id, err := res.LastInsertId()
				require.NoError(t, err)
				r.vals[c.name] = id
			}
		}
		sd.rows = append(sd.rows, r)
		return r
	}
	t.Fatalf("seed %s: unique key collisions", tbl)
	return nil
}

// ensure is the pure row of tbl for owner (every user column = owner), seeded once.
func (sd *seeder) ensure(tbl string, owner uint64) *seededRow {
	key := fmt.Sprintf("%d/%s", owner, tbl)
	if r, ok := sd.memo[key]; ok {
		return r
	}
	if sd.active[key] {
		return nil
	}
	sd.active[key] = true
	defer delete(sd.active, key)
	r := sd.row(tbl, owner, false, func(string) any { return owner })
	sd.memo[key] = r
	return r
}

func (sd *seeder) exists(r *seededRow) bool {
	pk := sd.s.pk[r.table]
	require.NotEmpty(sd.t, pk, "table %s has no primary key", r.table)
	var where []string
	var args []any
	for _, c := range pk {
		where = append(where, "`"+c+"` = ?")
		args = append(args, r.vals[c])
	}
	var n int
	require.NoError(sd.t, sd.db.QueryRow("SELECT COUNT(*) FROM `"+r.table+"` WHERE "+strings.Join(where, " AND "), args...).Scan(&n))
	return n > 0
}

// goneWith: deleting user uid removes this row (a CASCADE FK to users holding uid, a hand delete of a column holding
// uid, or a CASCADE FK to a parent row that goes).
func (sd *seeder) goneWith(r *seededRow, uid uint64, allow map[string]allowEntry) bool {
	for col, v := range r.vals {
		if v != uid {
			continue
		}
		if fk, ok := sd.s.fks[r.table][col]; ok && fk.refTable == "users" && fk.rule == "CASCADE" {
			return true
		}
		if allow[r.table+"."+col].kind == "deleted-by-code" {
			return true
		}
	}
	for col, p := range r.parents {
		if sd.s.fks[r.table][col].rule == "CASCADE" && sd.goneWith(p, uid, allow) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------------------------------------------
// Test 2: DELETE /api/v1/account removes every row, file and key of the user and nothing of anybody else.

type otpRecorder struct{}

func (otpRecorder) Dispatch(context.Context, string, any) error { return nil }

func testRedis(t *testing.T) *cache.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set (run `make test-int`)")
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), "ritme-go-test-"+hex.EncodeToString(b)+":")
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func userKeys(t *testing.T, uid, other uint64) []string {
	t.Helper()
	ck, err := cyclecache.Key{UserID: uid, Version: 1, Locale: "fa", Today: civildate.Date{Year: 2026, Month: 9, Day: 23},
		Scope: "today", Inputs: map[string]int{"x": 1}}.String()
	require.NoError(t, err)
	return []string{
		ck,
		idempotency.Key(uid, "client-key"),
		events.DayKey("shown", uid, 7, "home", "2026-09-23"),
		events.DayKey("eligible", uid, 7, "home", "2026-09-23"),
		events.GoalKey(uid, 7, time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)),
		fmt.Sprintf("seen:%d:2026-09-23", uid),
		fmt.Sprintf("logged:%d:2026-09-23", uid),
		fmt.Sprintf("family:ra:%d:%d:cycle", uid, other), // owner side
		fmt.Sprintf("family:ra:%d:%d:cycle", other, uid), // member side
	}
}

func TestDeleteAccount_RemovesEveryUserRowFileAndKey(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	rc := testRedis(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	aq := authstore.New(db)
	iss := passport.NewIssuer(key, aq, clock.Real{}, 365)
	guard := auth.NewGuardWith(&key.PublicKey, aq, clock.Real{}, quiet)
	storage := t.TempDir()
	h := profile.NewHandlers(profile.Options{DB: db, Logger: quiet, StoragePath: storage, Cache: rc})
	ah := auth.NewHandlers(aq, iss, otpRecorder{}, clock.Real{}, 30, quiet)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/profile", guard.RequireUser, h.Show)
	app.Delete("/api/v1/account", guard.RequireUser, h.DestroyAccount)
	app.Post("/api/v1/auth/send-otp", ah.SendOTP)
	app.Post("/api/v1/auth/verify-otp", ah.VerifyOTP)
	e := &env{db: db, app: app, iss: iss, storage: storage}

	const mobileA, mobileB = "09120000131", "09120000132"
	a, tokA := e.user(t, mobileA)
	b, tokB := e.user(t, mobileB)
	others := 0
	newOther := func() uint64 {
		others++
		id, _ := e.user(t, fmt.Sprintf("0912100%04d", others))
		return id
	}

	// Rows: the pure row of every user table for her and for the control user, then link rows.
	s := loadSchema(t, db)
	allow := loadAllowList(t)
	sd := newSeeder(t, db, s)
	var scope []string
	for _, tbl := range s.tables {
		if sd.scope[tbl] {
			scope = append(scope, tbl)
		}
	}
	require.Greater(t, len(scope), 100, "the user-table set comes from the schema")
	for _, tbl := range scope {
		sd.ensure(tbl, a)
		sd.ensure(tbl, b)
	}
	links := 0
	for _, tbl := range scope {
		var userFKs []string
		for _, c := range s.cols[tbl] {
			if fk, ok := s.fks[tbl][c.name]; ok && fk.refTable == "users" {
				userFKs = append(userFKs, c.name)
			}
		}
		if len(userFKs) < 2 {
			continue
		}
		for _, col := range userFKs {
			other := newOther()
			sd.row(tbl, other, true, func(c string) any {
				if c == col {
					return a
				}
				return other
			})
			links++
		}
	}
	require.Positive(t, links, "family / companion link rows")
	t.Logf("seeded %d rows in %d user tables (%d link rows)", len(sd.rows), len(scope), links)

	// OTP rows (keyed by number), stored files, Redis keys.
	for _, m := range []string{mobileA, mobileB} {
		_, err := db.Exec(`INSERT INTO otp_verifications (mobile, code, expires_at, attempts, created_at, updated_at)
			VALUES (?, '1234', '2026-09-23 10:05:00', 0, '2026-09-23 10:00:00', '2026-09-23 10:00:00')`, m)
		require.NoError(t, err)
	}
	vault, err := files.NewVault(storage, files.DevKey("t13"), nil, false)
	require.NoError(t, err)
	fsvc := files.NewService(files.Options{DB: db, Vault: vault, BaseURL: "https://api.test"})
	pdf := []byte("%PDF-1.4\n1 0 obj << /Ferritin 9 >> endobj\n%%EOF")
	for _, uid := range []uint64{a, b} {
		_, err := fsvc.Put(ctx, uid, files.PurposeRecordDocument, pdf, time.Now())
		require.NoError(t, err)
	}
	owners, err := files.Owners(storage, files.Registry[files.PurposeRecordDocument])
	require.NoError(t, err)
	require.ElementsMatch(t, []uint64{a, b}, owners)
	keysA, keysB := userKeys(t, a, b), userKeys(t, b, 999999)
	for _, k := range append(append([]string{}, keysA...), keysB...) {
		require.NoError(t, rc.Set(ctx, k, "1", time.Hour))
	}

	// The control user's footprint, per user column.
	countFor := func(uid uint64) map[string]int {
		out := map[string]int{}
		for _, tbl := range s.tables {
			for _, c := range s.cols[tbl] {
				if !s.userRef(tbl, c.name) {
					continue
				}
				var n int
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `"+tbl+"` WHERE `"+c.name+"` = ?", uid).Scan(&n))
				if n > 0 {
					out[tbl+"."+c.name] = n
				}
			}
		}
		return out
	}
	controlBefore := countFor(b)
	require.NotEmpty(t, countFor(a))

	r := e.do(t, http.MethodDelete, "/api/v1/account", tokA, "")
	require.Equal(t, 200, r.status, r.raw)

	// 1. No column that points at users holds her id (except the documented kept / pending ones).
	for col, n := range countFor(a) {
		switch allow[col].kind {
		case "kept", "pending-decision":
			continue
		}
		t.Errorf("%s still holds the deleted user's id in %d row(s)", col, n)
	}
	// 2. Every seeded row: hers that should go is gone, everything else is still there.
	for _, row := range sd.rows {
		gone := sd.goneWith(row, a, allow)
		switch {
		case gone && sd.exists(row):
			t.Errorf("%s: her row survived the delete (owner %d, link %v)", row.table, row.owner, row.mixed)
		case !gone && !sd.exists(row):
			t.Errorf("%s: a row that is not hers was deleted (owner %d, link %v)", row.table, row.owner, row.mixed)
		}
	}
	// 3. The control user is untouched, her token is dead and his works.
	assert.Equal(t, controlBefore, countFor(b), "control user's rows")
	r = e.do(t, http.MethodGet, "/api/v1/profile", tokA, "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "token_revoked", r.body["error_code"])
	assert.Equal(t, 200, e.do(t, http.MethodGet, "/api/v1/profile", tokB, "").status)
	// 4. OTP rows of her number, her files and her Redis keys are gone; his stay.
	var otps int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM otp_verifications WHERE mobile = ?`, mobileA).Scan(&otps))
	assert.Zero(t, otps)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM otp_verifications WHERE mobile = ?`, mobileB).Scan(&otps))
	assert.Equal(t, 1, otps)
	owners, err = files.Owners(storage, files.Registry[files.PurposeRecordDocument])
	require.NoError(t, err)
	assert.Equal(t, []uint64{b}, owners, "her blobs gone, his kept")
	for _, k := range keysA {
		_, err := rc.Get(ctx, k)
		assert.ErrorIs(t, err, cache.ErrMiss, k)
	}
	for _, k := range keysB[:len(keysB)-1] { // the last one names her as the reading member
		_, err := rc.Get(ctx, k)
		assert.NoError(t, err, "control key %s", k)
	}

	// 5. Logging in again with her number starts a new, empty account.
	r = e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"`+mobileA+`"}`)
	require.Equal(t, 200, r.status, r.raw)
	var code string
	require.NoError(t, db.QueryRow(`SELECT code FROM otp_verifications WHERE mobile = ? ORDER BY id DESC LIMIT 1`, mobileA).Scan(&code))
	r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", `{"mobile":"`+mobileA+`","code":"`+code+`"}`)
	require.Equal(t, 200, r.status, r.raw)
	data := r.body["data"].(map[string]any)
	assert.Equal(t, true, data["new_user"])
	assert.Equal(t, false, data["profile_completed"])
	newID := uint64(data["user"].(map[string]any)["id"].(float64))
	assert.NotEqual(t, a, newID)
	fresh := countFor(newID)
	delete(fresh, "oauth_access_tokens.user_id") // the login's own token
	assert.Empty(t, fresh, "no old data reappears under the new account")
	r = e.do(t, http.MethodGet, "/api/v1/profile", data["access_token"].(string), "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Nil(t, r.body["data"].(map[string]any)["profile"])
}
