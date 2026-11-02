package revision_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/revision"
)

const (
	queriesDir = "../../../db/queries"
	sourceDir  = "../../../internal"
	fixtureDir = "testdata/queries"
)

// writeQuery is one `-- name:` block of a db/queries file whose statement writes an engine table.
type writeQuery struct {
	Key    string // "<dir>.<name>", the Manifest key
	Tables []string
}

var (
	nameLine = regexp.MustCompile(`(?m)^--\s*name:\s*(\w+)\s+:\w+`)
	// Write targets. "ON DUPLICATE KEY UPDATE col = …" is not a target (handled in engineTargets).
	insertInto = regexp.MustCompile(`(?is)\b(?:INSERT|REPLACE)\s+(?:IGNORE\s+)?(?:INTO\s+)?\x60?(\w+)\x60?`)
	updateHead = regexp.MustCompile(`(?is)\bUPDATE\s+(.+?)\bSET\b`)
	deleteFrom = regexp.MustCompile(`(?is)\bDELETE\s+(?:\w+\s+)?FROM\s+\x60?(\w+)\x60?`)
	tableRef   = regexp.MustCompile(`(?i)(?:^|\bJOIN\s+|,\s*)\x60?(\w+)\x60?`)
)

// parseQueries reads every *.sql under root and returns its write queries on engine tables.
func parseQueries(t *testing.T, root string) []writeQuery {
	t.Helper()
	return parseQueriesOn(t, root, revision.EngineTables)
}

// parseQueriesOn reads every *.sql under root and returns its write queries on one of tables.
func parseQueriesOn(t *testing.T, root string, tables []string) []writeQuery {
	t.Helper()
	var out []writeQuery
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".sql" {
			return err
		}
		raw, err := os.ReadFile(path) //nolint:gosec // test fixture paths
		if err != nil {
			return err
		}
		dir := filepath.Base(filepath.Dir(path))
		text := string(raw)
		locs := nameLine.FindAllStringSubmatchIndex(text, -1)
		for i, loc := range locs {
			end := len(text)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			name := text[loc[2]:loc[3]]
			hit := targetsOn(statement(text[loc[1]:end]), tables)
			if len(hit) > 0 {
				out = append(out, writeQuery{Key: dir + "." + name, Tables: hit})
			}
		}
		return nil
	})
	require.NoError(t, err)
	return out
}

// statement drops the comment lines of a query block.
func statement(block string) string {
	var lines []string
	for _, l := range strings.Split(block, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "--") {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}

// engineTargets are the engine tables a statement writes (every table of a multi-table UPDATE counts).
func engineTargets(stmt string) []string { return targetsOn(stmt, revision.EngineTables) }

// targetsOn are the tables of list a statement writes.
func targetsOn(stmt string, list []string) []string {
	var targets []string
	for _, m := range insertInto.FindAllStringSubmatch(stmt, -1) {
		targets = append(targets, m[1])
	}
	for _, m := range updateHead.FindAllStringSubmatchIndex(stmt, -1) {
		if strings.HasSuffix(strings.ToUpper(strings.TrimSpace(stmt[:m[0]])), "KEY") {
			continue // ON DUPLICATE KEY UPDATE
		}
		for _, r := range tableRef.FindAllStringSubmatch(strings.TrimSpace(stmt[m[2]:m[3]]), -1) {
			targets = append(targets, r[1])
		}
	}
	for _, m := range deleteFrom.FindAllStringSubmatch(stmt, -1) {
		targets = append(targets, m[1])
	}
	var out []string
	for _, tb := range targets {
		if slices.Contains(list, strings.ToLower(tb)) && !slices.Contains(out, tb) {
			out = append(out, tb)
		}
	}
	return out
}

// testNames are the Test* functions declared in the repository's _test.go files.
func testNames(t *testing.T) map[string]bool {
	t.Helper()
	decl := regexp.MustCompile(`(?m)^func (Test\w+)\(t \*testing\.T\)`)
	names := map[string]bool{}
	err := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(path) //nolint:gosec // repository sources
		if err != nil {
			return err
		}
		for _, m := range decl.FindAllStringSubmatch(string(raw), -1) {
			names[m[1]] = true
		}
		return nil
	})
	require.NoError(t, err)
	return names
}

// problems checks the queries against the manifest: unlisted writes, stale entries, malformed entries, missing tests.
func problems(queries []writeQuery, manifest map[string]revision.Entry, tests map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, q := range queries {
		seen[q.Key] = true
		if _, ok := manifest[q.Key]; !ok {
			out = append(out, "unlisted write on "+strings.Join(q.Tables, ",")+": "+q.Key)
		}
	}
	for key, e := range manifest {
		switch {
		case !seen[key]:
			out = append(out, "stale entry (no such write query): "+key)
		case (e.Bumps == "") == (e.Exempt == ""):
			out = append(out, "entry needs exactly one of Bumps / Exempt: "+key)
		case e.Bumps != "" && !tests[e.Bumps]:
			out = append(out, "entry names a test that does not exist: "+key+" → "+e.Bumps)
		}
	}
	sort.Strings(out)
	return out
}

// T-M9-30 (README §3): every sqlc write query on an engine-input table is in the manifest, and each Bumps entry
// names an existing integration test.
func TestManifest_CoversEveryEngineWrite(t *testing.T) {
	queries := parseQueries(t, queriesDir)
	require.NotEmpty(t, queries)
	assert.Empty(t, problems(queries, revision.Manifest, testNames(t)))
}

// The manifest test catches a new writer: a fake query file with an `UPDATE cycle_histories` (and an aliased
// multi-table DELETE) fails it; reads and non-engine upserts do not.
func TestManifest_FailsOnUnlistedWrite(t *testing.T) {
	fake := parseQueries(t, fixtureDir)
	keys := []string{}
	for _, q := range fake {
		keys = append(keys, q.Key)
	}
	assert.ElementsMatch(t, []string{"fake.FakeTouchHistories", "fake.FakeDeleteAliased"}, keys)

	all := append(parseQueries(t, queriesDir), fake...)
	got := problems(all, revision.Manifest, testNames(t))
	assert.Equal(t, []string{
		"unlisted write on cycle_histories: fake.FakeTouchHistories",
		"unlisted write on cycle_skips: fake.FakeDeleteAliased",
	}, got)

	// A Bumps entry naming a test that does not exist fails too.
	m := map[string]revision.Entry{}
	for k, v := range revision.Manifest {
		m[k] = v
	}
	m["cycle.InsertPeriod"] = revision.Entry{Bumps: "TestBump_DoesNotExist"}
	assert.Contains(t, problems(parseQueries(t, queriesDir), m, testNames(t)),
		"entry names a test that does not exist: cycle.InsertPeriod → TestBump_DoesNotExist")
}

// T-M10-28 (D-177): every sqlc write query on an analysis-only input table is in AnalysisManifest, and each Bumps
// entry names an existing integration test.
func TestAnalysisManifest_CoversEveryAnalysisWrite(t *testing.T) {
	queries := parseQueriesOn(t, queriesDir, revision.AnalysisTables)
	require.NotEmpty(t, queries)
	assert.Empty(t, problems(queries, revision.AnalysisManifest, testNames(t)))
	for _, tb := range revision.AnalysisTables {
		assert.NotContains(t, revision.EngineTables, tb, "a table is engine or analysis-only, not both")
	}
}

// The analysis manifest catches a new writer too: the fixture's `INSERT INTO vital_readings` without a bump fails it.
func TestAnalysisManifest_FailsOnUnlistedWrite(t *testing.T) {
	fake := parseQueriesOn(t, fixtureDir, revision.AnalysisTables)
	keys := []string{}
	for _, q := range fake {
		keys = append(keys, q.Key)
	}
	assert.ElementsMatch(t, []string{"fake.FakeInsertReading"}, keys)

	all := append(parseQueriesOn(t, queriesDir, revision.AnalysisTables), fake...)
	assert.Equal(t, []string{"unlisted write on vital_readings: fake.FakeInsertReading"},
		problems(all, revision.AnalysisManifest, testNames(t)))
}

func TestEngineTargets(t *testing.T) {
	cases := map[string][]string{
		"INSERT INTO `cycle_histories` (a) VALUES (?)":                                     {"cycle_histories"},
		"INSERT IGNORE INTO health_log_entries SELECT 1":                                   {"health_log_entries"},
		"UPDATE `user_profiles` p JOIN `users` u ON u.id = p.user_id SET p.x = 1":          {"user_profiles"},
		"INSERT INTO `user_revisions` (a) VALUES (1) ON DUPLICATE KEY UPDATE a = a + 1":    nil,
		"INSERT INTO `user_profiles` (a) VALUES (1) ON DUPLICATE KEY UPDATE a = VALUES(a)": {"user_profiles"},
		"DELETE FROM `fertility_logs` WHERE user_id = ?":                                   {"fertility_logs"},
		"SELECT * FROM `cycle_histories`":                                                  nil,
	}
	for stmt, want := range cases {
		assert.Equal(t, want, engineTargets(stmt), stmt)
	}
}
