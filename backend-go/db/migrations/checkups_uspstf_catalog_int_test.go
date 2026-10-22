package migrations_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cc "github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const checkupsUSPSTFVersion int64 = 100 // 00100_checkups_uspstf_catalog.sql

type catalogRow struct {
	title, why, source          string
	informational               bool
	interval                    int
	intervalMax, ageMin, ageMax sql.NullInt64
}

func readCatalogRow(t *testing.T, conn *sql.DB, key string) (catalogRow, bool) {
	t.Helper()
	var r catalogRow
	var source sql.NullString
	err := conn.QueryRow("SELECT title, IFNULL(why, '{}'), source_note, informational, interval_months, interval_months_max, "+
		"age_min, age_max FROM checkup_types WHERE `key` = ? AND user_id IS NULL", key).
		Scan(&r.title, &r.why, &source, &r.informational, &r.interval, &r.intervalMax, &r.ageMin, &r.ageMax)
	if err == sql.ErrNoRows {
		return r, false
	}
	require.NoError(t, err)
	r.source = source.String
	return r, true
}

func languageCodes(t *testing.T, conn *sql.DB) []string {
	t.Helper()
	rows, err := conn.Query("SELECT code FROM languages ORDER BY sort_order")
	require.NoError(t, err)
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		codes = append(codes, c)
	}
	require.NoError(t, rows.Err())
	require.NotEmpty(t, codes)
	return codes
}

func translations(t *testing.T, raw string) map[string]string {
	t.Helper()
	m := map[string]string{}
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	return m
}

// T-M10-21: the two USPSTF types with copy in every seeded language, breast awareness informational, the Pap copy
// naming HPV, the mammography columns on the registry; Down restores the 00003 catalog and Up runs again.
func TestCheckupsUSPSTFCatalog_UpDown(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	codes := languageCodes(t, conn)
	assertUp := func() {
		for _, key := range []string{"bp_screening", "glucose_screening"} {
			r, ok := readCatalogRow(t, conn, key)
			require.True(t, ok, key)
			for _, code := range codes {
				assert.NotEmpty(t, translations(t, r.title)[code], "%s title %s", key, code)
				assert.NotEmpty(t, translations(t, r.why)[code], "%s why %s", key, code)
			}
			assert.Contains(t, r.source, "USPSTF 2021", key)
			assert.False(t, r.informational)
		}
		aware, _ := readCatalogRow(t, conn, "breast_self_exam")
		assert.True(t, aware.informational, "H-10")
		assert.Equal(t, "آگاهی از سینه", translations(t, aware.title)["fa"])
		pap, _ := readCatalogRow(t, conn, "pap_smear")
		for _, code := range codes {
			assert.Contains(t, translations(t, pap.why)[code], "HPV", code)
		}
		mammo, _ := readCatalogRow(t, conn, "mammography")
		assert.Equal(t, 24, mammo.interval)
		assert.False(t, mammo.intervalMax.Valid)
		assert.Equal(t, int64(74), mammo.ageMax.Int64)
		// Every screening.uspstf type key is a catalog row.
		for _, row := range cc.Rows(cc.KeyScreeningUSPSTF, civildate.Date{}) {
			_, ok := readCatalogRow(t, conn, row.Str("type_key"))
			assert.True(t, ok, "catalog row for %s", row.Str("type_key"))
		}
	}
	assertUp()

	// A user's record of a new type is removed with it on Down (FK cascade).
	uid := newV6User(t, conn, "09129990210")
	var bpID uint64
	require.NoError(t, conn.QueryRow("SELECT id FROM checkup_types WHERE `key` = 'bp_screening'").Scan(&bpID))
	_, err = conn.Exec(`INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, has_attachment, created_at, updated_at)
		VALUES (?, ?, '2026-10-01', 'normal', 0, NOW(), NOW())`, uid, bpID)
	require.NoError(t, err)

	_, err = p.DownTo(ctx, checkupsUSPSTFVersion-1)
	require.NoError(t, err)
	for _, key := range []string{"bp_screening", "glucose_screening"} {
		_, ok := readCatalogRow(t, conn, key)
		assert.False(t, ok, key)
	}
	aware, _ := readCatalogRow(t, conn, "breast_self_exam")
	assert.False(t, aware.informational)
	assert.Equal(t, "خودآزمایی سینه", translations(t, aware.title)["fa"])
	mammo, _ := readCatalogRow(t, conn, "mammography")
	assert.Equal(t, 12, mammo.interval)
	assert.Equal(t, int64(24), mammo.intervalMax.Int64)
	assert.False(t, mammo.ageMax.Valid)
	assert.Equal(t, "Default catalog (T-M4-01). Needs medical review before production.", mammo.source)

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assertUp()
}
