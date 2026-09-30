package insights

// On the contract fixtures: the regular persona gets a full history and a ready pattern, the
// irregular persona an irregular verdict, and each user only ever sees their own periods (the
// handlers take no id, the snapshot is loaded by the authenticated user id alone).

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	hlmodel "github.com/ritme/backend-go/internal/healthlog/model"
	hlstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	regularUser   = 1004
	irregularUser = 1005
)

func TestPersonasOnContractFixtures(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))
	svc := cycleservice.New(db, nil)
	ctx := context.Background()
	today := civildate.MustParse("2026-09-23")

	load := func(uid uint64) *cycleservice.Snapshot {
		sn, err := svc.Load(ctx, uid, today.AddDays(-PatternLookback), today, today)
		require.NoError(t, err)
		return sn
	}

	reg := load(regularUser)
	h := BuildHistory(reg.Histories, reg.EngineProfile(), today)
	require.NotEmpty(t, h.Cycles)
	assert.True(t, h.Cycles[0].IsCurrent)
	assert.Equal(t, civildate.MustParse("2026-09-14"), h.Cycles[0].Start)
	require.NotNil(t, h.MedianCycle)
	assert.Equal(t, 28, *h.MedianCycle)
	assert.Equal(t, RegularityRegular, h.Regularity)

	irr := load(irregularUser)
	hi := BuildHistory(irr.Histories, irr.EngineProfile(), today)
	assert.Equal(t, RegularityIrregular, hi.Regularity)

	// Isolation: no start of the other user's history leaks into either result.
	starts := func(sn *cycleservice.Snapshot) map[civildate.Date]bool {
		m := map[civildate.Date]bool{}
		for _, r := range sn.HistoryRows {
			m[r.PeriodStartDate] = true
		}
		return m
	}
	regStarts := starts(reg)
	for _, c := range h.Cycles {
		assert.True(t, regStarts[c.Start], "cycle %s belongs to the regular persona", c.Start)
	}
	for _, r := range reg.HistoryRows {
		assert.Equal(t, uint64(regularUser), r.UserID)
	}
	for _, r := range irr.LogRows {
		assert.Equal(t, uint64(irregularUser), r.UserID)
	}

	logs := map[civildate.Date]DayLog{}
	for _, r := range reg.LogRows {
		logs[r.LogDate] = Weigh(hlmodel.FromRow(hlstore.DailyHealthLog(r)))
	}
	p := BuildPattern(reg.Histories, reg.EngineProfile(), logs, today)
	assert.True(t, p.Ready)
	assert.Equal(t, 28, p.CycleLength)
	_, err := p.JSON().MarshalJSON()
	require.NoError(t, err)
}
