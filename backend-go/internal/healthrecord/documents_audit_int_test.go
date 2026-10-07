package healthrecord_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Security audit of CB-REC-02 (HIGH-1): list items that are not objects are a 422 on that item, never a panic.
func TestRecordExtras_NonObjectItemsAre422(t *testing.T) {
	e := setupDocs(t)
	_, tok := e.user(t, "09120000201")
	for _, c := range []struct {
		body any
		key  string
	}{
		{map[string]any{"surgeries": []any{[]any{"x"}}}, "surgeries.0.title"},
		{map[string]any{"family_history": []any{[]any{}}}, "family_history.0"},
	} {
		r := e.do(t, http.MethodPut, "/api/v1/health-record/extras", tok, c.body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
		assert.NotEmpty(t, errorKeys(r), r.raw)
	}
	r := e.do(t, http.MethodGet, "/api/v1/health-record/extras", tok, nil)
	assert.Equal(t, http.StatusOK, r.status, r.raw)
}
