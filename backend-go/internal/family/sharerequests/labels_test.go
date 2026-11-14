package sharerequests

import (
	"encoding/json"
	"io/fs"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func flatKeys(prefix string, v any, out map[string]bool) {
	m, ok := v.(map[string]any)
	if !ok {
		out[prefix] = true
		return
	}
	for k, sub := range m {
		flatKeys(prefix+"."+k, sub, out)
	}
}

func TestLang_KeyParity(t *testing.T) {
	dirs, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	var ref []string
	for _, d := range dirs {
		raw, err := fs.ReadFile(langFS, "lang/"+d.Name()+"/share_requests.json")
		require.NoError(t, err)
		var v any
		require.NoError(t, json.Unmarshal(raw, &v))
		keys := map[string]bool{}
		flatKeys("share_requests", v, keys)
		if ref == nil {
			ref = slices.Sorted(maps.Keys(keys))
			continue
		}
		assert.Equal(t, ref, slices.Sorted(maps.Keys(keys)), d.Name())
	}
	for _, k := range []string{"messages.created", "messages.approved", "messages.declined", "messages.not_found",
		"messages.too_many_requests", "messages.link_pending", "messages.validation_failed", "validation.self",
		"validation.domains_required", "validation.domain_invalid", "validation.level_invalid"} {
		assert.Contains(t, ref, "share_requests."+k)
		assert.NotEqual(t, "share_requests."+k, T(k, "en"), k)
	}
}
