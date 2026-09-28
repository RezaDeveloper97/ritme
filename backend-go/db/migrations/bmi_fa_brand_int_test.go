// Package migrations_test checks goose data migrations against a real MariaDB (the SQL files
// themselves are embedded by package db). Run with `make test-int PKG=./db/migrations/...`.
package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/content"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const bmiBrandVersion int64 = 6 // 00006_bmi_fa_brand.sql

// bmiTexts is the fa code default (T-M2-30, «ریتمی») and the text Laravel seeded ("Ritme").
func bmiTexts(t *testing.T, item string) (seeded, branded string) {
	t.Helper()
	v, ok := content.Default("bmi_message", item, "fa").Field("message")
	require.True(t, ok)
	branded, _ = v.(string)
	require.Contains(t, branded, "ریتمی")
	return strings.ReplaceAll(branded, "ریتمی", "Ritme"), branded
}

// laravelPayload is json_encode(['message' => $s]) without JSON_UNESCAPED_UNICODE, the way the
// Laravel seeder stored the rows (\uXXXX escapes).
func laravelPayload(s string) string {
	var b strings.Builder
	b.WriteString(`{"message":"`)
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r > 0x7e || r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteString(`"}`)
	return b.String()
}

func insertBmi(t *testing.T, db *sql.DB, item, locale, message string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		"INSERT INTO message_contents (`group`, item_key, locale, label, payload, created_at, updated_at) "+
			"VALUES ('bmi_message', ?, ?, ?, ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')",
		item, locale, "bmi_message / "+item, laravelPayload(message))
	require.NoError(t, err)
}

func bmiMessage(t *testing.T, db *sql.DB, item, locale string) string {
	t.Helper()
	var msg string
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT JSON_VALUE(payload, '$.message') FROM message_contents WHERE `group` = 'bmi_message' AND item_key = ? AND locale = ?",
		item, locale).Scan(&msg))
	return msg
}

// section returns the statements of the file's "-- +goose Up" or "-- +goose Down" block.
func section(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile("00006_bmi_fa_brand.sql")
	require.NoError(t, err)
	s := string(raw)
	up, down := strings.Index(s, "-- +goose Up"), strings.Index(s, "-- +goose Down")
	require.True(t, up >= 0 && down > up)
	body := s[up:down]
	if name == "Down" {
		body = s[down:]
	}
	var out []string
	for _, stmt := range strings.Split(body, ";\n") {
		if _, update, ok := strings.Cut(stmt, "UPDATE"); ok {
			out = append(out, "UPDATE"+update)
		}
	}
	require.Len(t, out, 2)
	return out
}

func TestBmiFaBrandMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	_, err = p.DownTo(ctx, bmiBrandVersion-1)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	normalSeeded, normalBranded := bmiTexts(t, "normal")
	obeseSeeded, obeseBranded := bmiTexts(t, "obese")
	obeseEdited := obeseSeeded + " (admin)"
	// Equal to the seeded text under utf8mb4_unicode_ci (ZWNJ is ignorable there) but not byte for
	// byte: must be treated as an admin edit.
	normalNoZWNJ := strings.ReplaceAll(normalSeeded, "\u200c", "")
	require.NotEqual(t, normalSeeded, normalNoZWNJ)

	insertBmi(t, db, "normal", "fa", normalSeeded)
	insertBmi(t, db, "obese", "fa", obeseEdited)
	insertBmi(t, db, "normal", "en", "… Ritme aims to help you maintain it.")
	insertBmi(t, db, "normal", "ar", normalNoZWNJ) // same guard text, other locale: untouched

	_, err = p.UpByOne(ctx)
	require.NoError(t, err)
	assert.Equal(t, normalBranded, bmiMessage(t, db, "normal", "fa"), "seeded default is rebranded")
	assert.Equal(t, obeseEdited, bmiMessage(t, db, "obese", "fa"), "admin edit is kept")
	assert.Equal(t, "… Ritme aims to help you maintain it.", bmiMessage(t, db, "normal", "en"), "en keeps Ritme")
	assert.Equal(t, normalNoZWNJ, bmiMessage(t, db, "normal", "ar"))

	var keys string
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT JSON_KEYS(payload) FROM message_contents WHERE `group` = 'bmi_message' AND item_key = 'normal' AND locale = 'fa'").Scan(&keys))
	assert.JSONEq(t, `["message"]`, keys)

	// Idempotent: running the Up statements again changes nothing.
	for _, stmt := range section(t, "Up") {
		res, err := db.ExecContext(ctx, stmt)
		require.NoError(t, err)
		n, err := res.RowsAffected()
		require.NoError(t, err)
		assert.Zero(t, n, "second run is a no-op")
	}

	// The seeded obese row (no admin edit) is rebranded as well.
	_, err = db.ExecContext(ctx, "UPDATE message_contents SET payload = ? WHERE `group` = 'bmi_message' AND item_key = 'obese' AND locale = 'fa'",
		laravelPayload(obeseSeeded))
	require.NoError(t, err)
	for _, stmt := range section(t, "Up") {
		_, err := db.ExecContext(ctx, stmt)
		require.NoError(t, err)
	}
	assert.Equal(t, obeseBranded, bmiMessage(t, db, "obese", "fa"))

	// Down restores the seeded text with the same guard: a row edited after Up stays.
	_, err = db.ExecContext(ctx, "UPDATE message_contents SET payload = JSON_SET(payload, '$.message', 'ویرایش ادمین') WHERE `group` = 'bmi_message' AND item_key = 'obese' AND locale = 'fa'")
	require.NoError(t, err)
	_, err = p.Down(ctx)
	require.NoError(t, err)
	v, err := p.GetDBVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, bmiBrandVersion-1, v)
	assert.Equal(t, normalSeeded, bmiMessage(t, db, "normal", "fa"), "Down restores the seeded text")
	assert.Equal(t, "ویرایش ادمین", bmiMessage(t, db, "obese", "fa"), "Down keeps an admin edit")
	assert.Equal(t, normalNoZWNJ, bmiMessage(t, db, "normal", "ar"))

	// Up again after Down.
	_, err = p.UpByOne(ctx)
	require.NoError(t, err)
	assert.Equal(t, normalBranded, bmiMessage(t, db, "normal", "fa"))
}
