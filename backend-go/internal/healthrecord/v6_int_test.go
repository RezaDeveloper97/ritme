package healthrecord_test

// T-M9-35: the record's cycle section under CYCLE_ENGINE=v6 — last_period_start is the latest actual start (never
// the declared onboarding date), regularity and effective lengths come from the engine. Integration:
// `make test-int PKG=./internal/healthrecord/...`.

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestCycleSectionV6(t *testing.T) {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cq := cyclestore.New(db)
	p, err := state.New(cq, cache.NewFromClient(rdb, "ritme-go-t-m9-35h:"), clock.Real{},
		state.Options{Enabled: true, L1Entries: 10, Build: "test", Logger: quiet})
	require.NoError(t, err)
	src := state.Source{P: p, Revs: cq}
	v1 := healthrecord.NewService(db, nil)
	v6 := healthrecord.NewService(db, nil).WithEngine(src)
	today := civildate.MustParse("2026-09-23")

	section := func(svc *healthrecord.Service, uid uint64) map[string]any {
		t.Helper()
		rec, err := svc.Build(context.Background(), uid, healthrecord.AudienceOwner,
			healthrecord.Options{Today: today, Locale: "en", DefaultLocale: "fa", Sections: []string{healthrecord.SectionCycle}})
		require.NoError(t, err)
		for _, s := range rec.Sections {
			if s.Key == healthrecord.SectionCycle {
				raw, err := json.Marshal(s.Data)
				require.NoError(t, err)
				var m map[string]any
				require.NoError(t, json.Unmarshal(raw, &m))
				return m
			}
		}
		t.Fatalf("no cycle section")
		return nil
	}

	for _, uid := range []uint64{1003, 1004, 1005, 1006, 1007, 1011, 1012} {
		st, err := src.ForUser(context.Background(), uid, today)
		require.NoError(t, err)
		d := engine.DayOf(today)
		now := st.Resolve(d, d)
		got := section(v6, uid)
		assert.Equal(t, now.Metrics.CycleRegularity.String(), got["regularity"], "uid %d", uid)
		cl, _ := st.CycleLength()
		pl, _ := st.PeriodLength()
		assert.EqualValues(t, cl, got["effective_cycle_length"], "uid %d", uid)
		assert.EqualValues(t, pl, got["effective_period_length"], "uid %d", uid)

		var last any
		for _, p := range st.Periods() {
			if p.Start <= d {
				last = p.Start.String()
			}
		}
		assert.Equal(t, last, got["last_period_start"], "uid %d: the latest actual start", uid)
		assert.NotContains(t, section(v1, uid), "effective_cycle_length", "v1 unchanged")
	}
	// The declared onboarding date is never a period start in v6.
	assert.Nil(t, section(v6, 1003)["last_period_start"])
}
