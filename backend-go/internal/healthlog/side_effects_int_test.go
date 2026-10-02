package healthlog_test

// POST /health-logs side effects against Laravel: testdata/sideeffects/steps.json is replayed
// through the Go validation + service on the contract fixtures, and the resulting
// cycle_histories / user_profiles / daily_health_logs rows (snapshot.sql) plus each step's
// status and warning must equal what the same steps did on the Laravel contract stack
// (laravel.jsonl, recorded by capture_laravel.sh) — after applyD55, the documented Go fix of the
// period reconciliation for back-dated and in-period bleeding days (deviations.md D-55, B-N3-14b).

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

func TestMain(m *testing.M) { testdb.Main(m) }

type sideEffectStep struct {
	Doc    string          `json:"doc"`
	User   uint64          `json:"user"`
	Locale string          `json:"locale"`
	Now    string          `json:"now"`
	Body   json.RawMessage `json:"body"`
}

func TestStoreSideEffectsMatchLaravel(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	dir := filepath.Join("testdata", "sideeffects")

	raw, err := os.ReadFile(filepath.Join(dir, "steps.json"))
	require.NoError(t, err)
	var steps []sideEffectStep
	require.NoError(t, json.Unmarshal(raw, &steps))

	svc := healthlog.NewService(db)
	ctx := context.Background()
	var got []string
	for i, st := range steps {
		now, err := civildate.ParseLenient(st.Now, time.Now(), civildate.Tehran)
		require.NoError(t, err)
		status, warning := 0, any(nil)
		attrs, err := healthlog.ValidateStore(validation.DecodeBody(st.Body), st.Locale, now)
		require.NoError(t, err, "step %d (%s)", i, st.Doc)
		res, err := svc.Store(ctx, st.User, attrs, st.Locale, now)
		require.NoError(t, err, "step %d (%s)", i, st.Doc)
		status = 200
		if res.Created {
			status = 201
		}
		if res.Warning != nil {
			warning = res.Warning
		}
		line, err := jsonx.Marshal(jsonx.Obj("step", i, "status", status, "warning", warning), jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
		require.NoError(t, err)
		got = append(got, string(line))
	}

	snapshot, err := os.ReadFile(filepath.Join(dir, "snapshot.sql"))
	require.NoError(t, err)
	for _, q := range strings.Split(strings.TrimSpace(string(snapshot)), ";\n") {
		func() {
			rows, err := db.QueryContext(ctx, strings.TrimSuffix(q, ";"))
			require.NoError(t, err)
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var s string
				require.NoError(t, rows.Scan(&s))
				got = append(got, s)
			}
			require.NoError(t, rows.Err())
		}()
	}

	want := applyD55(t, readLines(t, filepath.Join(dir, "laravel.jsonl")))
	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, normJSON(t, want[i]), normJSON(t, got[i]), "line %d", i+1)
	}
}

// d55 maps the Laravel lines D-55 changes to Go's ("" = the row no longer exists):
//   - step 4 (regular, back-dated 09-01 between the 08-17 and 09-14 periods): a closed one-day period with a
//     positive cycle_length (15) instead of an open row with −20, and the LMP stays on the latest start (09-21)
//     instead of moving back to 09-01;
//   - step 3 (regular, new period 09-21): the confirmed 09-14..09-18 period keeps its logged end (Laravel cut
//     it to the last bleeding log, 09-16);
//   - step 7 (open_period_day3, bleeding 09-22 inside the open period started 09-21): no new 1-day period,
//     the LMP stays 09-21.
var d55 = map[string]string{
	`["cycle_histories", 101808, 1004, "2026-09-01", null, -20, null, 0, 0, "user_logged", null, "2026-09-23 10:00:00", "2026-09-23 10:00:00"]`:               `["cycle_histories", 101808, 1004, "2026-09-01", "2026-09-01", 15, 1, 0, 0, "user_logged", null, "2026-09-23 10:00:00", "2026-09-23 10:00:00"]`,
	`["cycle_histories", 100406, 1004, "2026-09-14", "2026-09-16", null, 3, 1, 0, "user_logged", null, "2026-09-23 09:00:00", "2026-09-23 10:20:00"]`:         `["cycle_histories", 100406, 1004, "2026-09-14", "2026-09-18", null, 5, 1, 0, "user_logged", null, "2026-09-23 09:00:00", "2026-09-23 09:00:00"]`,
	`["cycle_histories", 101810, 1007, "2026-09-22", null, 1, null, 0, 0, "user_logged", null, "2026-09-23 10:00:00", "2026-09-23 10:00:00"]`:                 "",
	`["user_profiles", 1004, 1004, "2026-09-01", "completed", "2026-09-23 10:00:00", "2026-09-23 10:00:00", 4, "2026-09-23 09:00:00", "2026-09-23 10:00:00"]`: `["user_profiles", 1004, 1004, "2026-09-21", "completed", "2026-09-23 10:00:00", "2026-09-23 10:00:00", 4, "2026-09-23 09:00:00", "2026-09-23 10:00:00"]`,
	`["user_profiles", 1007, 1007, "2026-09-22", "completed", "2026-09-23 11:00:00", "2026-09-23 11:00:00", 4, "2026-09-23 09:00:00", "2026-09-23 11:00:00"]`: `["user_profiles", 1007, 1007, "2026-09-21", "completed", "2026-09-23 11:00:00", "2026-09-23 11:00:00", 4, "2026-09-23 09:00:00", "2026-09-23 11:00:00"]`,
}

func applyD55(t *testing.T, lines []string) []string {
	t.Helper()
	out := make([]string, 0, len(lines))
	seen := 0
	for _, l := range lines {
		repl, ok := d55[l]
		if !ok {
			out = append(out, l)
			continue
		}
		seen++
		if repl != "" {
			out = append(out, repl)
		}
	}
	require.Equal(t, len(d55), seen, "every D-55 line is in laravel.jsonl")
	return out
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // test fixture
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if s := strings.TrimSpace(sc.Text()); s != "" {
			out = append(out, s)
		}
	}
	require.NoError(t, sc.Err())
	return out
}

// normJSON re-indents a JSON line (MariaDB's JSON_ARRAY spacing, key order kept).
func normJSON(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, []byte(s)), s)
	return buf.String()
}
