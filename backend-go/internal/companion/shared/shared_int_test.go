package shared_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/companion/shared"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// T-M11-06 (D-194): the legacy sections map onto the v6 allow-list DTOs; symptoms is gone (F-03); a link without a
// member (or any snapshot Decide refuses) gets nothing.
func TestReader_LegacySectionsMapOntoProjections(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120006001', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	owner := uint64(id) //nolint:gosec // test id
	_, err = db.Exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at)
		VALUES (?, '2026-09-22', 'mood', 'moods', 'calm', '1', NULL, NULL, 'manual', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, owner)
	require.NoError(t, err)

	r := shared.NewReader(db)
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

	_, err = r.Read(ctx, owner, companion.SectionSymptoms, "en", now)
	require.ErrorIs(t, err, companion.ErrInvalidSection, "F-03: no symptoms projection")
	_, err = r.ReadFor(ctx, companion.Link{OwnerID: owner}, companion.SectionSymptoms, "en", now)
	require.ErrorIs(t, err, companion.ErrInvalidSection)
	_, err = r.Read(ctx, owner, companion.Section("bank"), "en", now)
	require.ErrorIs(t, err, companion.ErrInvalidSection)

	for _, sec := range []companion.Section{companion.SectionCycle, companion.SectionMeds, companion.SectionAppointments, companion.SectionPregnancy} {
		v, err := r.ReadFor(ctx, companion.Link{OwnerID: owner}, sec, "en", now)
		require.ErrorIs(t, err, companion.ErrProjectionDenied, sec)
		assert.Nil(t, v, sec)
	}

	v, err := r.Read(ctx, owner, companion.SectionPregnancy, "en", now)
	require.NoError(t, err)
	raw, _ := json.Marshal(v)
	assert.JSONEq(t, `{"current_week":null,"trimester":null,"edd":null}`, string(raw))
	v, err = r.Read(ctx, owner, companion.SectionCycle, "en", now)
	require.NoError(t, err)
	raw, _ = json.Marshal(v)
	assert.JSONEq(t, `{"main_phase":null}`, string(raw))
	for _, sec := range []companion.Section{companion.SectionMeds, companion.SectionAppointments} {
		v, err = r.Read(ctx, owner, sec, "en", now)
		require.NoError(t, err)
		raw, _ = json.Marshal(v)
		assert.Equal(t, "[]", string(raw))
	}
}
