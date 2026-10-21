package fertility_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// T-M9-04: concurrent partial saves of one day (lh from one request, mucus from another) are
// serialised by the per-user lock, so neither read-merge-upsert drops the other's key.
func TestSave_ConcurrentPartialSaves_KeepEveryKey(t *testing.T) {
	db := testdb.New(t)
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', '09120000501', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id) //nolint:gosec // G115: test id
	svc := fertility.NewService(db)
	day := civildate.MustParse("2026-10-01")
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)
	lh, mucus := "positive", "egg_white"

	for round := range 5 {
		_, err := db.Exec(`DELETE FROM fertility_logs WHERE user_id = ?`, uid)
		require.NoError(t, err)
		var wg sync.WaitGroup
		errs := make([]error, 2)
		start := make(chan struct{})
		wg.Go(func() {
			<-start
			_, errs[0] = svc.Save(context.Background(), uid, fertility.DayInput{Date: day, Set: map[string]bool{"lh": true}, LH: &lh}, "fa", now)
		})
		wg.Go(func() {
			<-start
			_, errs[1] = svc.Save(context.Background(), uid, fertility.DayInput{Date: day, Set: map[string]bool{"mucus": true}, Mucus: &mucus}, "fa", now)
		})
		close(start)
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])

		var gotLH, gotMucus string
		require.NoError(t, db.QueryRow(`SELECT COALESCE(lh_test, ''), COALESCE(cervical_mucus, '') FROM fertility_logs WHERE user_id = ? AND log_date = ?`,
			uid, day.String()).Scan(&gotLH, &gotMucus))
		assert.Equal(t, []string{lh, mucus}, []string{gotLH, gotMucus}, "round %d", round)
	}
}
