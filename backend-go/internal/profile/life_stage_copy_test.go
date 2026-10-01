package profile

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLossCopyJSON_KeepsEveryKeyInOrder(t *testing.T) {
	out, err := json.Marshal(LossCopyJSON(map[string]any{"confirm": "پایان", "title": "", "body": 3, "other": "x"}))
	require.NoError(t, err)
	assert.JSONEq(t, `{"title":null,"body":null,"confirm":"پایان","cancel":null,"done_title":null,"done_body":null,"done_action":null}`, string(out))
	assert.Equal(t, `{"title":null,"body":null,"confirm":"پایان","cancel":null,"done_title":null,"done_body":null,"done_action":null}`, string(out))

	out, err = json.Marshal(LossCopyJSON(nil))
	require.NoError(t, err)
	assert.Contains(t, string(out), `"title":null`)
}
