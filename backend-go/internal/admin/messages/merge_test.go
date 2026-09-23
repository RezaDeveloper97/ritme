package messages

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestMergePayload(t *testing.T) {
	orig, err := phpval.Decode([]byte(`{"title":"a","tips":["x"],"n":3}`))
	require.NoError(t, err)
	in, err := phpval.Decode([]byte(`{"title":"b","tips":"l1\r\nl2\r\r\n","n":["no"],"extra":"z"}`))
	require.NoError(t, err)
	b, err := json.Marshal(mergePayload(orig, in))
	require.NoError(t, err)
	assert.JSONEq(t, `{"title":"b","tips":["l1","l2"],"n":3}`, string(b))

	b, err = json.Marshal(mergePayload(orig, nil))
	require.NoError(t, err)
	assert.JSONEq(t, `{"title":"a","tips":[],"n":3}`, string(b), "a list field with no input empties (Laravel)")
}
