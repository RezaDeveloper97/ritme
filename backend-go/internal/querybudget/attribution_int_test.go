package querybudget_test

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/querybudget"
)

// Query attribution (T-M12-30b): the per-request counter says how many queries a route made; the
// db_query_duration_seconds{query} histogram says which. Requests run one at a time here, so the difference of the
// histogram's sample counts around one call is that call's queries by sqlc name ("raw" = hand-written SQL).

// queryNames is the process's executed-query count per sqlc name so far.
func queryNames(t *testing.T) map[string]uint64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	out := map[string]uint64{}
	for _, mf := range mfs {
		if mf.GetName() != "db_query_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "query" {
					out[lp.GetValue()] = m.GetHistogram().GetSampleCount()
				}
			}
		}
	}
	return out
}

// attribute is "Name ×n" for each query name whose count grew from before to after, by name.
func attribute(before, after map[string]uint64) string {
	var parts []string
	for _, name := range slices.Sorted(maps.Keys(after)) {
		if d := after[name] - before[name]; d > 0 {
			parts = append(parts, fmt.Sprintf("%s ×%d", name, d))
		}
	}
	return strings.Join(parts, ", ")
}

// fixtureToken seeds the benchfixtures user of mode and returns a personal access token for her.
func fixtureToken(t *testing.T, db *sql.DB, keysPath string, mode enums.LifeMode, mobile string) (uint64, string) {
	t.Helper()
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	exec(t, db, `INSERT IGNORE INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`, clientID)
	var uid uint64
	if u, ok := benchfixtures.UserFor(mode); ok {
		uid = seedUser(t, db, u, mobile)
	} else {
		uid = seedBare(t, db, mode, mobile)
	}
	tok, err := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365).Issue(context.Background(), uid, time.Now())
	require.NoError(t, err)
	return uid, tok.AccessToken
}

func budgetOf(t *testing.T, name string) querybudget.Budget {
	t.Helper()
	i := slices.IndexFunc(querybudget.Budgets, func(b querybudget.Budget) bool { return b.Name == name })
	require.GreaterOrEqual(t, i, 0, "no budget row %q", name)
	return querybudget.Budgets[i]
}

// TestQueryAttribution logs the cold and warm queries of the rows T-M12-30b works on, by sqlc name (-v). It asserts
// nothing beyond what TestQueryBudgets does; it is the profile the budget rows' comments quote.
func TestQueryAttribution(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	defer configcache.SetEnabled(false)
	for i, name := range []string{"pregnancy v2 today", "todo", "todo v6", "care today", "checkups home", "home notifications"} {
		b := budgetOf(t, name)
		_, tok := fixtureToken(t, db, keysPath, b.Mode, fmt.Sprintf("0912000%04d", 8200+i))
		t.Run(name, func(t *testing.T) {
			configcache.SetEnabled(false)
			app := newApp(t, db, keysPath, b.Engine)
			before := queryNames(t)
			cold, _ := call(t, app, b.Path, tok)
			coldBy := attribute(before, queryNames(t))
			configcache.SetEnabled(true)
			_, _ = call(t, app, b.Path, tok)
			before = queryNames(t)
			warm, _ := call(t, app, b.Path, tok)
			t.Logf("%s cold %d: %s", b.Path, cold, coldBy)
			t.Logf("%s warm %d: %s", b.Path, warm, attribute(before, queryNames(t)))
		})
	}
}
