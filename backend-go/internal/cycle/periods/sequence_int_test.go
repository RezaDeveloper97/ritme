package periods

// Period-log write sequence against Laravel: testdata/sequence/steps.json (start → end → edit →
// delete → store … across personas) is replayed through the Go period service on the contract
// fixtures, and each step's status / code / warning types plus the resulting cycle_histories and
// user_profiles rows (snapshot.sql) must equal what the same requests did on the Laravel
// contract stack (laravel.jsonl, recorded by capture_laravel.sh).

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

type sequenceStep struct {
	Doc    string            `json:"doc"`
	User   uint64            `json:"user"`
	Locale string            `json:"locale"`
	Now    string            `json:"now"`
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Target string            `json:"target"`
	Body   map[string]string `json:"body"`
}

func TestPeriodSequenceMatchesLaravel(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))
	dir := filepath.Join("testdata", "sequence")

	raw, err := os.ReadFile(filepath.Join(dir, "steps.json"))
	require.NoError(t, err)
	var steps []sequenceStep
	require.NoError(t, json.Unmarshal(raw, &steps))

	svc := NewService(db)
	ctx := context.Background()
	date := func(s string) *civildate.Date {
		if s == "" {
			return nil
		}
		d := civildate.MustParse(s)
		return &d
	}

	var got []string
	for i, st := range steps {
		now, err := time.Parse(time.RFC3339, st.Now)
		require.NoError(t, err)
		now = now.In(civildate.Tehran)
		var id uint64
		if st.Target != "" {
			require.NoError(t, db.QueryRowContext(ctx,
				"SELECT id FROM cycle_histories WHERE user_id = ? AND period_start_date = ?", st.User, st.Target).Scan(&id))
		}

		var res *Result
		switch {
		case st.Method == http.MethodPost && st.Path == "/cycle/period/start":
			start := civildate.InTehran(now)
			if d := date(st.Body["date"]); d != nil {
				start = *d
			}
			res, err = svc.Start(ctx, st.User, start, now)
		case st.Method == http.MethodPost && st.Path == "/cycle/period/end":
			res, err = svc.End(ctx, st.User, date(st.Body["date"]), now)
		case st.Method == http.MethodPost && st.Path == "/cycle/period":
			res, err = svc.Store(ctx, st.User, *date(st.Body["start_date"]), date(st.Body["end_date"]), now)
		case st.Method == http.MethodPut:
			res, err = svc.Update(ctx, st.User, id, *date(st.Body["start_date"]), date(st.Body["end_date"]), now)
		case st.Method == http.MethodDelete:
			err = svc.Destroy(ctx, st.User, id, now)
		default:
			t.Fatalf("step %d: unsupported %s %s", i, st.Method, st.Path)
		}

		status, code, types := 200, any(nil), []string{}
		var refused *Refusal
		switch {
		case errors.As(err, &refused):
			status = refused.Status
			if refused.Code != "" {
				code = refused.Code
			}
		case err != nil:
			require.NoError(t, err, "step %d (%s)", i, st.Doc)
		case res != nil:
			for _, w := range Warnings(intPtr(res.Row.CycleLength), intPtr(res.Row.BleedingLength), st.Locale) {
				typ, _ := w.(*jsonx.OrderedMap).Get("type")
				types = append(types, typ.(string))
			}
		}
		line, err := jsonx.Marshal(jsonx.Obj("step", i, "status", status, "code", code, "warnings", types), jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
		require.NoError(t, err)
		got = append(got, string(line))
	}

	snapshot, err := os.ReadFile(filepath.Join(dir, "snapshot.sql"))
	require.NoError(t, err)
	for _, q := range strings.Split(strings.TrimSpace(string(snapshot)), ";\n") {
		got = append(got, queryStrings(t, db, strings.TrimSuffix(q, ";"))...)
	}

	want := readLines(t, filepath.Join(dir, "laravel.jsonl"))
	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, normJSON(t, want[i]), normJSON(t, got[i]), "line %d", i+1)
	}
}

func queryStrings(t *testing.T, db *sql.DB, q string) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), q)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
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

// normJSON compacts a JSON line (MariaDB's JSON_ARRAY spacing, key order kept).
func normJSON(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, []byte(s)), s)
	return buf.String()
}
