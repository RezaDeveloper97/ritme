package cycletips

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// go test ./internal/messages/cycletips -run TestSeedSQL -seed-out=<file> writes the VALUES rows of the goose seed
// from Defs (base language fa, like the other rule seeds). Without the flag it does nothing.
var seedOut = flag.String("seed-out", "", "write the seed VALUES rows of Defs to this file")

func sqlString(s string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), "'", "''") + "'"
}

func TestSeedSQL(t *testing.T) {
	if *seedOut == "" {
		t.Skip("no -seed-out")
	}
	var b strings.Builder
	for i, d := range Defs {
		content, err := d.ContentJSON("fa")
		require.NoError(t, err)
		topics := "NULL"
		if js := d.TopicsJSON(); js != "" {
			topics = sqlString(js)
		}
		b.WriteString("(" + strings.Join([]string{
			sqlString(d.Key), sqlString(d.Name), sqlString(Level), sqlString(Kind), strconv.Itoa(d.Prio), sqlString(Slot),
			sqlString(content), sqlString(d.ConditionsJSON()), "'[]'", topics,
			"0, 50, NULL, NULL, 0, 100, 'approved', 1, 1, 1, 0, NOW(), NOW()",
		}, ", ") + ")")
		if i < len(Defs)-1 {
			b.WriteString(",\n")
		}
	}
	require.NoError(t, os.WriteFile(*seedOut, []byte(b.String()+"\n"), 0o600))
}
