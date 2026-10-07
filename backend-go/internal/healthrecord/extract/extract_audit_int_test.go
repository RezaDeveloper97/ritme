package extract_test

// Security audit of CB-REC-02: malformed review bodies are 422 (never a panic), a pending document refuses a kind
// change or a confirm, and deleting a document whose job still waits in the queue refunds the reserved use once.

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthrecord"
)

func TestAudit_MalformedReviewBodiesAre422(t *testing.T) {
	e := setup(t, true)
	uid, tok := e.user(t, "09120000101", true, true)
	id := e.document(t, uid, tok, healthrecord.KindPrescription, "prescription")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil).status)

	for _, c := range []struct {
		body map[string]any
		key  string
	}{
		{map[string]any{"fields": []any{}}, "fields"},
		{map[string]any{"fields": []any{"x"}}, "fields"},
		{map[string]any{"items": []any{[]any{"x"}}}, "items.0"},
		{map[string]any{"items": []any{map[string]any{"medicine": "A"}, []any{}}}, "items.1"},
	} {
		r := e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, c.body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
		errs, _ := r.body["errors"].(map[string]any)
		assert.Contains(t, errs, c.key, r.raw)
	}
	// The document is untouched and can still be reviewed.
	r := e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, map[string]any{"fields": map[string]any{"doctor": "Dr. C"}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
}

func TestAudit_PendingDocumentAndDeleteRefund(t *testing.T) {
	e := setup(t, false) // async: the job stays queued (the workers are not started)
	uid, tok := e.user(t, "09120000102", true, true)
	id := e.document(t, uid, tok, healthrecord.KindVisit, "visit")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil).status)
	assert.Equal(t, 1, e.usage(t, uid))

	for _, body := range []map[string]any{{"kind": "hospital"}, {"confirm": true}} {
		r := e.do(t, http.MethodPut, docsPath+"/"+id, tok, body)
		require.Equal(t, http.StatusConflict, r.status, r.raw)
		assert.Equal(t, healthrecord.ErrorCodeDocumentBusy, r.body["error_code"])
	}
	r := e.do(t, http.MethodPut, docsPath+"/"+id, tok, map[string]any{"note": "while reading"})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, healthrecord.ReviewPending, r.data()["review_state"], "the locked row's state is kept")

	r = e.do(t, http.MethodDelete, docsPath+"/"+id, tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, e.usage(t, uid), "the queued job's reserved use was given back")

	// A finished document's delete refunds nothing.
	e2 := setup(t, true)
	uid2, tok2 := e2.user(t, "09120000103", true, true)
	id2 := e2.document(t, uid2, tok2, healthrecord.KindVisit, "visit")
	require.Equal(t, http.StatusAccepted, e2.do(t, http.MethodPost, docsPath+"/"+id2+"/extract", tok2, nil).status)
	require.Equal(t, http.StatusOK, e2.do(t, http.MethodDelete, docsPath+"/"+id2, tok2, nil).status)
	assert.Equal(t, 1, e2.usage(t, uid2))
}
