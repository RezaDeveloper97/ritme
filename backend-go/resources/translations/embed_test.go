package translations_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/resources/translations"
)

// frontendMessages is frontend/messages, the source of truth for the bundled locales
// (frontend/CLAUDE.md §6.4).
var frontendMessages = filepath.Join("..", "..", "..", "frontend", "messages")

// TestSeedMatchesFrontendMessages fails when the embedded seed drifts from
// frontend/messages: every bundled locale folder there must be here with exactly the same
// files, byte for byte. Fix a failure with
//
//	cp ../frontend/messages/<code>/*.json resources/translations/<code>/
//
// then re-record internal/i18n/testdata/messages_*.json (see TestBundle_MatchesLaravelGoldens).
func TestSeedMatchesFrontendMessages(t *testing.T) {
	locales, err := os.ReadDir(frontendMessages)
	if os.IsNotExist(err) {
		t.Skip("frontend/messages not present (backend-go built on its own)")
	}
	require.NoError(t, err)

	seen := 0
	for _, loc := range locales {
		if !loc.IsDir() {
			continue
		}
		code := loc.Name()
		seen++
		t.Run(code, func(t *testing.T) {
			want := jsonFiles(t, os.DirFS(filepath.Join(frontendMessages, code)), ".")
			got := jsonFiles(t, translations.FS, code)
			assert.Equal(t, sortedKeys(want), sortedKeys(got), "namespace files differ from frontend/messages/%s", code)
			for name, w := range want {
				if g, ok := got[name]; ok {
					assert.Equal(t, string(w), string(g), "%s/%s differs from frontend/messages", code, name)
				}
			}
		})
	}
	require.Positive(t, seen, "no locale folders in frontend/messages")
}

func jsonFiles(t *testing.T, fsys fs.FS, dir string) map[string][]byte {
	t.Helper()
	entries, err := fs.ReadDir(fsys, dir)
	require.NoError(t, err)
	out := map[string][]byte{}
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		require.NoError(t, err)
		out[e.Name()] = b
	}
	return out
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
