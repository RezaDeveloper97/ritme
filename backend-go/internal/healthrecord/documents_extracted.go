package healthrecord

import (
	"encoding/json"

	"github.com/ritme/backend-go/db"
)

// ExtractedJobKey is the key of record_documents.extracted that holds the extraction job's bookkeeping (CB-REC-02,
// internal/healthrecord/extract): attempts, lease, the reserved Plus use. It is never shown to a client.
const ExtractedJobKey = "job"

// publicExtracted is the extracted JSON as clients see it: the stored object without its job bookkeeping (nil when
// the document was never sent to extraction). Keys come out sorted (encoding/json), values unchanged.
func publicExtracted(raw db.NullRawJSON) any {
	if !raw.Valid || len(raw.V) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw.V, &m); err != nil || m == nil {
		return nil
	}
	delete(m, ExtractedJobKey)
	return m
}
