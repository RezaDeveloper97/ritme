package querybudget_test

import (
	"fmt"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/querybudget"
)

// send is call for any method with a JSON body: the request's query count and body (status 200 required).
func send(t *testing.T, app *fiber.App, method, path, token, body string) (int64, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, requestNow)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, fiber.StatusOK, resp.StatusCode, "%s %s: %s", method, path, raw)
	n, err := strconv.ParseInt(resp.Header.Get(queriesHead), 10, 64)
	require.NoError(t, err, "query count header on %s", path)
	return n, string(raw)
}

// TestWriteBudgets is TestQueryBudgets for querybudget.WriteBudgets (T-M12-31): cold on a fresh app, warm the same
// write again with the caches filled, each against its row; -v prints the counts by sqlc name.
func TestWriteBudgets(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	defer configcache.SetEnabled(false)
	for i, w := range querybudget.WriteBudgets {
		_, tok := fixtureToken(t, db, keysPath, w.Mode, fmt.Sprintf("0912000%04d", 8300+i))
		t.Run(w.Name, func(t *testing.T) {
			configcache.SetEnabled(false)
			app := newApp(t, db, keysPath, w.Engine)
			before := queryNames(t)
			cold, _ := send(t, app, w.Method, w.Path, tok, w.Body)
			coldBy := attribute(before, queryNames(t))
			configcache.SetEnabled(true)
			_, _ = send(t, app, w.Method, w.Path, tok, w.Body)
			before = queryNames(t)
			warm, _ := send(t, app, w.Method, w.Path, tok, w.Body)
			t.Logf("%s %s cold %d: %s", w.Method, w.Path, cold, coldBy)
			t.Logf("%s %s warm %d: %s", w.Method, w.Path, warm, attribute(before, queryNames(t)))
			withinBudget(t, w.Budget, cold, warm)
			if cold < w.Cold || warm < w.Warm {
				t.Logf("%s: measured %d / %d, below its budget %d / %d — lower the row in budgets.go", w.Name, cold, warm, w.Cold, w.Warm)
			}
		})
	}
}

// TestWriteBudgetFailsWhenExceeded: the write row sits at its measured count, so a save with one query more (a
// middleware standing in for a regression, e.g. a per-entry insert coming back) fails TestWriteBudgets' check.
func TestWriteBudgetFailsWhenExceeded(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	w := querybudget.WriteBudgets[0]
	_, tok := fixtureToken(t, db, keysPath, w.Mode, "09120008390")
	regression := func(c fiber.Ctx) error {
		if _, err := db.ExecContext(c, "SELECT 1"); err != nil {
			return err
		}
		return c.Next()
	}
	defer configcache.SetEnabled(false)
	measure := func(extra ...fiber.Handler) (int64, int64) {
		configcache.SetEnabled(false)
		app := newApp(t, db, keysPath, w.Engine, extra...)
		cold, _ := send(t, app, w.Method, w.Path, tok, w.Body)
		configcache.SetEnabled(true)
		_, _ = send(t, app, w.Method, w.Path, tok, w.Body)
		warm, _ := send(t, app, w.Method, w.Path, tok, w.Body)
		return cold, warm
	}
	cold, warm := measure()
	assert.Equal(t, w.Warm, warm, "the row is tight: budget = measured warm count")
	assert.True(t, withinBudget(t, w.Budget, cold, warm))

	cold, warm = measure(regression)
	rec := &failRecorder{}
	assert.False(t, withinBudget(rec, w.Budget, cold, warm), "one query more (%d / %d) must exceed %d / %d", cold, warm, w.Cold, w.Warm)
	assert.True(t, rec.failed)
}
