package accountexport

import (
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// T-M12-14b: the export is an allow-list. Every column of an exported table is either selected by its sqlc query
// (exportColumns) or named as hidden in tables.go — never both, never neither. A column a later migration adds fails
// here until someone decides which, so a new sensitive column (token, hash, path, sealed blob) cannot leak by default.
func TestAccountExport_ColumnGuard(t *testing.T) {
	db := testdb.New(t)

	schema := map[string][]string{}
	rows, err := db.Query(`SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = DATABASE() ORDER BY table_name, ordinal_position`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tbl, col string
		require.NoError(t, rows.Scan(&tbl, &col))
		schema[tbl] = append(schema[tbl], col)
	}
	require.NoError(t, rows.Err())

	jsonRe := regexp.MustCompile("^json_valid\\(`([^`]+)`\\)$")
	jsonCols := map[string]bool{}
	crows, err := db.Query(`SELECT table_name, check_clause FROM information_schema.check_constraints WHERE constraint_schema = DATABASE()`)
	require.NoError(t, err)
	defer func() { _ = crows.Close() }()
	for crows.Next() {
		var tbl, clause string
		require.NoError(t, crows.Scan(&tbl, &clause))
		if m := jsonRe.FindStringSubmatch(clause); m != nil {
			jsonCols[tbl+"."+m[1]] = true
		}
	}
	require.NoError(t, crows.Err())

	specs := []spec{}
	for _, g := range tableGroups(nil) {
		specs = append(specs, g.parts...)
	}
	specs = append(specs, shopExtra...)
	specs = append(specs, directoryExtra...)
	require.Greater(t, len(specs), 90)

	guard := func(table string, exported, hidden, have []string) []string {
		var bad []string
		known := map[string]string{}
		for _, c := range exported {
			known[c] = "exported"
		}
		for _, c := range hidden {
			if known[c] != "" {
				bad = append(bad, table+"."+c+": both exported and hidden")
			}
			known[c] = "hidden"
		}
		inSchema := map[string]bool{}
		for _, c := range have {
			inSchema[c] = true
			if known[c] == "" {
				bad = append(bad, table+"."+c+": new column is neither exported nor hidden — decide in db/queries/profile/export_*.sql + export_gen.go (select it) or tables.go (hidden)")
			}
		}
		for c := range known {
			if !inSchema[c] {
				bad = append(bad, table+"."+c+": listed but not in the table (stale)")
			}
		}
		sort.Strings(bad)
		return bad
	}

	for _, s := range specs {
		exported := exportColumns[s.table]
		if s.table == dedupeBackupTable {
			exported = dedupeBackupColumns
		}
		require.NotEmpty(t, exported, "%s has no export columns", s.table)
		assert.Empty(t, guard(s.table, exported, s.hidden, schema[s.table]), s.table)
	}
	for key := range jsonTextColumns {
		assert.True(t, jsonCols[key], "%s is in jsonTextColumns but has no json_valid CHECK", key)
	}

	t.Run("checker", func(t *testing.T) {
		got := guard("t", []string{"id", "name"}, []string{"token"}, []string{"id", "name", "token", "secret"})
		assert.Equal(t, 1, len(got))
		assert.Contains(t, got[0], "t.secret: new column is neither exported nor hidden")
		assert.Len(t, guard("t", []string{"id", "token"}, []string{"token"}, []string{"id", "token"}), 1)
	})
}
