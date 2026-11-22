package healthlog_test

// T-M12-31 (P-06): the batched day save (one DELETE of the changed params + one multi-row INSERT, the day read once
// under the user lock) against the old per-param / per-row path, plus read-your-write and the one revision bump.

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// propPool is every value the property test may send per "category.param" (cycle mode): valid ones, clears (null)
// and texts that stress the JSON batch (quotes, <>&, a newline, an emoji, Persian, the 2000-char maximum).
var propPool = map[string][]string{
	"bleeding.flow":            {`"light"`, `"medium"`, `"heavy"`, `null`},
	"bleeding.spotting":        {`true`, `false`, `null`},
	"bleeding.color":           {`"pink"`, `"black"`, `null`},
	"bleeding.clots":           {`true`, `false`, `null`},
	"pain.location":            {`{"abdomen":{"level":"moderate","score":6}}`, `{"head":{"level":"severe","score":9},"abdomen":{"level":"moderate"}}`, `null`},
	"pain.relief":              {`["heat"]`, `["heat","rest"]`, `["painkiller"]`, `null`},
	"mood.moods":               {`["calm"]`, `["calm","energetic"]`, `null`},
	"sleep.duration":           {`"0_3"`, `"6_9"`, `null`},
	"sleep.quality":            {`"great"`, `"poor"`, `null`},
	"appetite_energy.energy":   {`"low"`, `"high"`, `null`},
	"appetite_energy.cravings": {`{"sweet":"yes"}`, `{"salty":"yes","carbs":"yes"}`, `null`},
	"activity.duration":        {`30`, `90`, `null`},
	"measurements.weight":      {`58.4`, `61`, `null`},
	"discharge.amount":         {`"light"`, `"heavy"`, `null`},
	"discharge.consistency":    {`"creamy"`, `"egg_white"`, `null`},
	"urogenital.urination":     {`"increased"`, `"normal"`, `null`},
	"sex.desire":               {`"lower"`, `"higher"`, `null`},
	"note.text":                {`"سلام"`, `"a<b & \"q\"\nline 😀"`, `"` + strings.Repeat("ی", 2000) + `"`, `null`},
}

// randomBody is a PUT /logs/days body of 1..10 distinct params from propPool (sometimes a whole category null)
// and the params to mark as voice-confirmed.
func randomBody(rng *rand.Rand) (string, []string) {
	keys := slices.Sorted(func(yield func(string) bool) {
		for k := range propPool {
			if !yield(k) {
				return
			}
		}
	})
	rng.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	n := 1 + rng.IntN(10)
	byCat := map[string][]string{}
	var cats, voice []string
	for _, k := range keys[:n] {
		cat, param, _ := strings.Cut(k, ".")
		if _, ok := byCat[cat]; !ok {
			cats = append(cats, cat)
		}
		vals := propPool[k]
		byCat[cat] = append(byCat[cat], strconv.Quote(param)+":"+vals[rng.IntN(len(vals))])
		if rng.IntN(4) == 0 {
			voice = append(voice, k)
		}
	}
	parts := make([]string, 0, len(cats))
	for _, cat := range cats {
		if rng.IntN(12) == 0 {
			parts = append(parts, strconv.Quote(cat)+":null")
			continue
		}
		parts = append(parts, strconv.Quote(cat)+":{"+strings.Join(byCat[cat], ",")+"}")
	}
	return `{"categories":{` + strings.Join(parts, ",") + `}}`, voice
}

// parser is the handler's validation for one user (custom items, flow gates) over a given stored day.
func parser(t *testing.T, svc *healthlog.Service, userID uint64, date civildate.Date, body string, voice []string) func([]taxonomy.Entry) ([]taxonomy.Change, error) {
	t.Helper()
	ctx := context.Background()
	custom, err := svc.CustomSet(ctx, userID)
	require.NoError(t, err)
	gates, err := svc.FlowGates(ctx, userID, "cycle", date)
	require.NoError(t, err)
	return func(existing []taxonomy.Entry) ([]taxonomy.Change, error) {
		changes, err := taxonomy.Parse(validation.DecodeBody([]byte(body)), "cycle", "en", existing, custom, gates)
		if err != nil {
			return nil, err
		}
		for i := range changes { // markVoice's effect, without the body field
			if slices.Contains(voice, changes[i].Key()) && len(changes[i].Entries) > 0 {
				changes[i].Source = healthlog.SourceVoice
			}
		}
		return changes, nil
	}
}

// dump is every stored row of the user the save path writes — entries (id order), the legacy rows and the periods
// — as text (NULL spelled out), so two users' states compare with one Equal.
func dump(t *testing.T, db *sql.DB, userID uint64) []string {
	t.Helper()
	var out []string
	for _, q := range []string{
		"SELECT log_date, category, param, item, value_code, value_num, value_text, source FROM health_log_entries WHERE user_id = ? ORDER BY log_date, id",
		"SELECT * FROM daily_health_logs WHERE user_id = ? ORDER BY log_date",
		"SELECT period_start_date, period_end_date, cycle_length, bleeding_length, is_confirmed, source FROM cycle_histories WHERE user_id = ? ORDER BY period_start_date",
	} {
		out = append(out, dumpQuery(t, db, q, userID)...)
	}
	return out
}

// dumpQuery is one query's rows as "col=value | …" lines (id and user_id left out: they differ by construction).
func dumpQuery(t *testing.T, db *sql.DB, q string, userID uint64) []string {
	t.Helper()
	rows, err := db.Query(q, userID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	require.NoError(t, err)
	var out []string
	for rows.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		var line []string
		for i, c := range cols {
			if c == "id" || c == "user_id" {
				continue
			}
			if vals[i].Valid {
				line = append(line, c+"="+vals[i].String)
			} else {
				line = append(line, c+"=NULL")
			}
		}
		out = append(out, strings.Join(line, " | "))
	}
	require.NoError(t, rows.Err())
	return out
}

// TestSaveDay_MatchesReferencePath is the property test: many random edits (seeded; LOG_PROP_SEED replays one) over a
// few days, each applied to user A by the batched SaveDayWith and to user B by the pre-T-M12-31 path
// (SaveDayReference, test-only). After every edit both must agree on the outcome (saved / the same validation
// error), the returned day, and every stored row: entries in id order with their source, the legacy write-back and
// the bleeding-log periods.
func TestSaveDay_MatchesReferencePath(t *testing.T) {
	e := setupLogs(t)
	a, _ := e.user(t, "09120003101", "cycle")
	b, _ := e.user(t, "09120003102", "cycle")
	svc := healthlog.NewService(e.db)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	dates := []civildate.Date{civildate.MustParse("2026-09-18"), civildate.MustParse("2026-09-20"), civildate.MustParse("2026-09-22")}

	seed := uint64(time.Now().UnixNano()) //nolint:gosec // a test seed
	if s := os.Getenv("LOG_PROP_SEED"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		require.NoError(t, err)
		seed = v
	}
	t.Logf("seed %d (LOG_PROP_SEED=%d replays it)", seed, seed)
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // deterministic test data

	const iterations = 150
	saved := 0
	for i := range iterations {
		date := dates[rng.IntN(len(dates))]
		body, voice := randomBody(rng)
		dayA, errA := svc.SaveDayWith(ctx, a, date, parser(t, svc, a, date, body, voice), "en", now)

		existingB, err := svc.Day(ctx, b, date)
		require.NoError(t, err)
		changesB, errB := parser(t, svc, b, date, body, voice)(existingB)
		var dayB []taxonomy.Entry
		if errB == nil {
			dayB, errB = svc.SaveDayReference(ctx, b, date, changesB, "en", now)
		}

		msg := fmt.Sprintf("iteration %d (seed %d) %s %s voice=%v", i, seed, date, body, voice)
		if errB != nil {
			require.Error(t, errA, msg)
			require.Equal(t, errB.Error(), errA.Error(), msg)
		} else {
			require.NoError(t, errA, msg)
			saved++
			require.Equal(t, dayB, dayA, "returned day: "+msg)
		}
		require.Equal(t, dump(t, e.db, b), dump(t, e.db, a), "stored rows: "+msg)
	}
	t.Logf("%d of %d random edits saved (the rest: the same validation error on both paths)", saved, iterations)
	assert.Greater(t, saved, iterations*2/3, "the pool must mostly produce valid saves, or the property proves little")
	assert.NotEmpty(t, dump(t, e.db, a))
}

// TestSaveDay_DuplicateParamLastWins (boundary): two changes to one param in one call leave only the last one's rows
// — what the per-param delete + insert left — and the returned day says the same as the table.
func TestSaveDay_DuplicateParamLastWins(t *testing.T) {
	e := setupLogs(t)
	uid, _ := e.user(t, "09120003103", "cycle")
	svc := healthlog.NewService(e.db)
	date := civildate.MustParse("2026-09-22")
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	moods := func(codes ...string) taxonomy.Change {
		ch := taxonomy.Change{Category: "mood", Param: "moods"}
		for _, c := range codes {
			ch.Entries = append(ch.Entries, taxonomy.Entry{Category: "mood", Param: "moods", Item: c})
		}
		return ch
	}
	day, err := svc.SaveDay(context.Background(), uid, date, []taxonomy.Change{moods("calm", "sad"), moods("calm")}, "en", now)
	require.NoError(t, err)
	stored, err := svc.Day(context.Background(), uid, date)
	require.NoError(t, err)
	assert.Equal(t, stored, day)
	require.Len(t, stored, 1)
	assert.Equal(t, "calm", stored[0].Item)

	// An empty change list writes nothing (no DELETE / INSERT) and keeps the day.
	day, err = svc.SaveDay(context.Background(), uid, date, nil, "en", now)
	require.NoError(t, err)
	assert.Equal(t, stored, day)
}

func engineRev(t *testing.T, db *sql.DB, userID uint64) int64 {
	t.Helper()
	var rev int64
	err := db.QueryRow("SELECT engine_rev FROM user_revisions WHERE user_id = ?", userID).Scan(&rev)
	if err == sql.ErrNoRows {
		return 0
	}
	require.NoError(t, err)
	return rev
}

// TestLogs_SaveBumpsRevisionOnceAndReadsBack: every PUT /logs/days/{date} bumps the engine revision exactly once (the
// engine caches key on it — no stale read after the write) and the next GET returns what was saved; a refused body
// (validated under the lock now) writes nothing and bumps nothing.
func TestLogs_SaveBumpsRevisionOnceAndReadsBack(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120003104", "cycle")
	path := "/api/v1/logs/days/2026-09-22"

	for i, body := range []string{
		`{"categories":{"mood":{"moods":["calm"]},"note":{"text":"one"},"sleep":{"quality":"good"},"pain":{"relief":["heat","rest"]}}}`,
		`{"categories":{"mood":{"moods":["calm","energetic"]},"note":{"text":"two"}}}`,
		`{"categories":{"pain":null,"note":{"text":"three"}}}`,
	} {
		before := engineRev(t, e.db, uid)
		r := e.do(t, "PUT", path, tok, "en", body)
		require.Equal(t, 200, r.status, r.raw)
		assert.Equal(t, before+1, engineRev(t, e.db, uid), "save %d bumps the revision exactly once", i)
		g := e.do(t, "GET", path, tok, "en", "")
		require.Equal(t, 200, g.status)
		assert.JSONEq(t, mustJSON(t, r.data()), mustJSON(t, g.data()), "save %d reads back", i)
	}
	g := e.do(t, "GET", path, tok, "en", "")
	assert.Equal(t, map[string]any{"text": "three"}, g.categories()["note"])
	assert.Equal(t, map[string]any{"moods": []any{"calm", "energetic"}}, g.categories()["mood"])
	assert.NotContains(t, g.categories(), "pain")
	assert.Equal(t, map[string]any{"quality": "good"}, g.categories()["sleep"])

	// Negative: one invalid param refuses the whole body — no row changes, no bump.
	before, rows := engineRev(t, e.db, uid), e.count(t, "SELECT COUNT(*) FROM health_log_entries WHERE user_id = ?", uid)
	r := e.do(t, "PUT", path, tok, "en", `{"categories":{"note":{"text":"four"},"measurements":{"weight":500}}}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, before, engineRev(t, e.db, uid), "a refused save bumps nothing")
	assert.Equal(t, rows, e.count(t, "SELECT COUNT(*) FROM health_log_entries WHERE user_id = ?", uid))
	assert.Equal(t, map[string]any{"text": "three"}, e.do(t, "GET", path, tok, "en", "").categories()["note"])
}
