package sharelinks_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/sharelinks"
	"github.com/ritme/backend-go/internal/sharelinks/store"
)

// B-N6-04b follow-ups of the doctor report: the report window bounds checkups and labs (L-2), and the active-link
// caps hold under parallel creates (L-3).

// itemDates is the `field` of every row in section key of a record body.
func itemDates(t *testing.T, rec *jsonx.OrderedMap, key, field string) []string {
	t.Helper()
	raw, err := rec.MarshalJSON()
	require.NoError(t, err)
	var body struct {
		Sections []struct {
			Key  string `json:"key"`
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		} `json:"sections"`
	}
	require.NoError(t, json.Unmarshal(raw, &body))
	for _, s := range body.Sections {
		if s.Key == key {
			out := []string{}
			for _, it := range s.Data.Items {
				out = append(out, it[field].(string))
			}
			return out
		}
	}
	t.Fatalf("section %s missing", key)
	return nil
}

func TestReport_WindowBoundsCheckupsAndLabs(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid, _ := e.user(t, "09120000401", false)
	var pap uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM checkup_types WHERE user_id IS NULL ORDER BY sort_order, id LIMIT 1").Scan(&pap))
	for _, d := range []string{"2026-09-10", "2026-08-01", "2026-05-01", "2025-01-15"} {
		e.exec(t, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, created_at, updated_at)
			VALUES (?, ?, ?, 'normal', '2026-09-10 09:00:00', '2026-09-10 09:00:00')`, uid, pap, d)
	}
	// Labs: dated by the sheet date, else the upload day.
	for _, l := range []struct{ taken, created string }{
		{"2026-09-20", "2026-09-21 09:00:00"}, {"", "2026-08-15 09:00:00"}, {"2026-04-01", "2026-10-01 09:00:00"},
		{"", "2025-06-01 09:00:00"},
	} {
		var taken any
		if l.taken != "" {
			taken = l.taken
		}
		e.exec(t, `INSERT INTO lab_reports (user_id, source, category, taken_on, status, interpret_count, created_at, updated_at)
			VALUES (?, 'manual', 'blood', ?, 'ready', 0, ?, ?)`, uid, taken, l.created, l.created)
	}
	labsSvc := labs.NewService(labs.Options{
		DB: e.db, Catalog: catalog.NewReader(catalogstore.New(e.db), nil, 0, quiet), Logger: quiet,
	})
	svc := healthrecord.NewService(e.db, labsSvc)
	sections := []string{healthrecord.SectionCheckups, healthrecord.SectionLabs}

	body, err := svc.BuildReport(ctx, uid, healthrecord.ReportRequest{Range: healthrecord.ReportRange3M, Sections: sections},
		fixed, "en", "fa")
	require.NoError(t, err)
	rec, _ := body.Get("record")
	assert.Equal(t, []string{"2026-09-10", "2026-08-01"}, itemDates(t, rec.(*jsonx.OrderedMap), "checkups", "done_on"),
		"3m window starts 2026-07-08")
	assert.Equal(t, []string{"2026-09-20", "2026-08-15"}, itemDates(t, rec.(*jsonx.OrderedMap), "labs", "date"))

	body, err = svc.BuildReport(ctx, uid, healthrecord.ReportRequest{
		Range: healthrecord.ReportRangeCustom, From: civildate.MustParse("2026-08-01"), Sections: sections,
	}, fixed, "en", "fa")
	require.NoError(t, err)
	rec, _ = body.Get("record")
	assert.Equal(t, []string{"2026-09-10", "2026-08-01"}, itemDates(t, rec.(*jsonx.OrderedMap), "checkups", "done_on"),
		"custom window: done_on >= from (inclusive)")
	assert.Equal(t, []string{"2026-09-20", "2026-08-15"}, itemDates(t, rec.(*jsonx.OrderedMap), "labs", "date"))

	body, err = svc.BuildReport(ctx, uid, healthrecord.ReportRequest{Range: healthrecord.ReportRange1Y, Sections: sections},
		fixed, "en", "fa")
	require.NoError(t, err)
	rec, _ = body.Get("record")
	assert.Equal(t, []string{"2026-09-10", "2026-08-01", "2026-05-01"}, itemDates(t, rec.(*jsonx.OrderedMap), "checkups", "done_on"))
	assert.Equal(t, []string{"2026-09-20", "2026-08-15", "2026-04-01"}, itemDates(t, rec.(*jsonx.OrderedMap), "labs", "date"))

	// The owner's record is not windowed: the newest rows whatever their date.
	own, err := svc.Build(ctx, uid, healthrecord.AudienceOwner, healthrecord.Options{Today: civildate.InTehran(fixed), Locale: "en", DefaultLocale: "fa"})
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-09-10", "2026-08-01", "2026-05-01", "2025-01-15"}, itemDates(t, own.JSON(), "checkups", "done_on"))
	assert.Equal(t, []string{"2026-09-20", "2026-08-15", "2026-04-01", "2025-06-01"}, itemDates(t, own.JSON(), "labs", "date"))
}

// race runs n creates at once and returns how many succeeded and how many hit want.
func race(t *testing.T, n int, create func() error, want error) (int, int) {
	t.Helper()
	var (
		wg              sync.WaitGroup
		mu              sync.Mutex
		ok, limited     int
		start           = make(chan struct{})
		unexpectedError error
	)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := create()
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, want):
				limited++
			default:
				unexpectedError = err
			}
		}()
	}
	close(start)
	wg.Wait()
	require.NoError(t, unexpectedError)
	return ok, limited
}

func TestShareLinks_ActiveLimitHoldsUnderParallelCreates(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid, _ := e.user(t, "09120000402", false)
	for i := range sharelinks.MaxActive - 2 {
		e.exec(t, `INSERT INTO health_share_links (user_id, token_hash, payload, sections, range_from, range_to, expires_at, created_at, updated_at)
			VALUES (?, ?, NULL, '["basics"]', '2026-07-01', '2026-10-06', '2026-10-10 09:00:00', '2026-10-05 09:00:00', '2026-10-05 09:00:00')`,
			uid, strings.Repeat(string(rune('a'+i)), 64))
	}
	req := healthrecord.ReportRequest{Range: healthrecord.ReportRange3M, Sections: []string{"basics"}}
	ok, limited := race(t, 12, func() error {
		_, err := e.svc.Create(ctx, uid, req, fixed, "en", "fa")
		return err
	}, sharelinks.ErrLimit)
	assert.Equal(t, 2, ok, "only the two free slots are taken")
	assert.Equal(t, 10, limited)
	var n int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM health_share_links WHERE user_id = ? AND kind = 'report'", uid).Scan(&n))
	assert.Equal(t, sharelinks.MaxActive, n)
}

func TestSummary_ActiveLimitHoldsUnderParallelCreates(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid, _ := e.user(t, "09120000403", false)
	svc := sharelinks.NewService(store.New(e.db), healthrecord.NewService(e.db, nil), quiet).WithDB(e.db).
		WithCodes([]byte("pepper-0123456789-0123456789-0123456789"), false)
	ok, limited := race(t, 9, func() error {
		_, err := svc.CreateSummary(ctx, uid, summaryReq(), fixed, "en", "fa")
		return err
	}, sharelinks.ErrSummaryLimit)
	assert.Equal(t, sharelinks.MaxActiveSummaries, ok)
	assert.Equal(t, 9-sharelinks.MaxActiveSummaries, limited)
	var n int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM health_share_links WHERE user_id = ? AND kind = 'summary'", uid).Scan(&n))
	assert.Equal(t, sharelinks.MaxActiveSummaries, n)
}
