package db_test

import (
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db"
)

// T-M9-09: the DSN builder turns on client-side interpolation over a safe utf8mb4 connection, keeps Tehran
// wall-clock parsing and never sets the session time_zone.
func TestDSN_InterpolateParams(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tehran")
	require.NoError(t, err)
	dsn := db.DSN(config.DB{Host: "mysql", Port: 3306, Database: "ritme_salamat", Username: "ritme", Password: "p@ss"}, loc)

	parsed, err := mysql.ParseDSN(dsn) // ParseDSN rejects interpolateParams with an unsafe collation
	require.NoError(t, err)
	assert.True(t, parsed.InterpolateParams)
	assert.True(t, parsed.ParseTime)
	assert.Equal(t, "Asia/Tehran", parsed.Loc.String())
	assert.Equal(t, db.Collation, parsed.Collation)
	assert.Contains(t, dsn, "charset=utf8mb4") // ParseDSN moves charset out of Params
	assert.NotContains(t, parsed.Params, "time_zone")
	assert.Equal(t, "ritme_salamat", parsed.DBName)
	assert.Equal(t, "mysql:3306", parsed.Addr)
}

// The driver itself refuses interpolation with a collation whose multi-byte sequences can hide a backslash; utf8mb4
// is accepted.
func TestDSN_UnsafeCollationRefused(t *testing.T) {
	_, err := mysql.ParseDSN("u:p@tcp(h:3306)/d?interpolateParams=true&collation=gbk_chinese_ci")
	require.Error(t, err)
	_, err = mysql.ParseDSN("u:p@tcp(h:3306)/d?interpolateParams=true&collation=" + db.Collation)
	require.NoError(t, err)
}
