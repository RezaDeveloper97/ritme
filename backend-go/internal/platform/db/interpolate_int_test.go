package db_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// T-M9-09: with interpolateParams a parameterised query is one round-trip (no COM_STMT_PREPARE), and the inlined
// arguments round-trip byte for byte — quotes, backslashes, Persian text, emoji, NUL, JSON — and DATETIMEs stay
// Tehran wall-clock.
func TestInterpolateParams_OneRoundTripAndSafeEscaping(t *testing.T) {
	ctx := context.Background()
	conn, err := testdb.New(t).Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	prepares := func() int {
		var name, v string
		require.NoError(t, conn.QueryRowContext(ctx, "SHOW SESSION STATUS LIKE 'Com_stmt_prepare'").Scan(&name, &v))
		n, err := strconv.Atoi(v)
		require.NoError(t, err)
		return n
	}

	_, err = conn.ExecContext(ctx, `CREATE TEMPORARY TABLE interp_probe (
		id INT PRIMARY KEY, s TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci, j JSON, at DATETIME, n BIGINT UNSIGNED)`)
	require.NoError(t, err)

	tricky := []string{
		`it's`, `"double"`, `back\slash`, `\'; DROP TABLE users; --`, "nul\x00byte", "سلام «ریتمی»\u200cها", "🌸 emoji",
		"\\\\'\\\"", "%_?", "خط\nجدید\r\ttab",
	}
	at := time.Date(2026, 3, 21, 0, 30, 15, 0, time.FixedZone("Asia/Tehran", 3*3600+1800))
	before := prepares()
	for i, s := range tricky {
		js, _ := json.Marshal(map[string]string{"v": s})
		_, err := conn.ExecContext(ctx, "INSERT INTO interp_probe (id, s, j, at, n) VALUES (?, ?, ?, ?, ?)",
			i, s, json.RawMessage(js), at, uint64(1)<<63)
		require.NoError(t, err)
	}
	for i, s := range tricky {
		var got string
		var js []byte
		var gotAt time.Time
		var n uint64
		require.NoError(t, conn.QueryRowContext(ctx, "SELECT s, j, at, n FROM interp_probe WHERE id = ? AND s = ?", i, s).
			Scan(&got, &js, &gotAt, &n))
		assert.Equal(t, s, got)
		var m map[string]string
		require.NoError(t, json.Unmarshal(js, &m))
		assert.Equal(t, s, m["v"])
		assert.Equal(t, "2026-03-21 00:30:15", gotAt.Format(time.DateTime), "wall-clock kept")
		assert.Equal(t, uint64(1)<<63, n)
	}
	assert.Equal(t, before, prepares(), "no server-side prepare: every parameterised query was interpolated")
}
