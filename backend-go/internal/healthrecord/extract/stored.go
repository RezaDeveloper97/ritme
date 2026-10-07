package extract

import (
	"encoding/json"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Extraction statuses (Stored.Status; the document's review_state says the same for the whole document).
const (
	StatusPending = "pending"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// Dating offer outcomes kept on the document (Stored.Dating).
const (
	DatingApplied   = "applied"
	DatingDismissed = "dismissed"
)

// Job states (Job.State).
const (
	jobQueued  = "queued"
	jobRunning = "running"
)

// Value is one value read from the document with the provider's confidence (0..1).
type Value struct {
	Value      any     `json:"value"`
	Confidence float64 `json:"confidence"`
}

// Stored is record_documents.extracted. Everything but Job is shown to the owner (GET /health-record/documents/{id}
// → `extracted`); Job is the queue bookkeeping (healthrecord.ExtractedJobKey).
type Stored struct {
	Schema        string             `json:"schema"`
	Status        string             `json:"status"`
	ErrorCode     *string            `json:"error_code"`
	Fields        map[string]Value   `json:"fields"`
	Items         []map[string]Value `json:"items"`
	Reviewed      map[string]any     `json:"reviewed"`       // the values the user confirmed (null until reviewed)
	ReviewedItems []map[string]any   `json:"reviewed_items"` // prescription rows as confirmed (null until reviewed)
	RequestedAt   string             `json:"requested_at"`
	ExtractedAt   *string            `json:"extracted_at"`
	ReviewedAt    *string            `json:"reviewed_at"`
	Dating        *string            `json:"dating"`        // applied | dismissed | null (the pregnancy dating offer)
	Job           *Job               `json:"job,omitempty"` // healthrecord.ExtractedJobKey
}

// Job is the extraction job of a pending document (lease and retries decided in Go; no queue table).
type Job struct {
	State       string `json:"state"`
	Attempts    int    `json:"attempts"`     // the claim token: writes of an attempt only land while it still matches
	AvailableAt int64  `json:"available_at"` // unix seconds
	LockedUntil int64  `json:"locked_until"` // unix seconds (running)
	Locale      string `json:"locale"`
	ReservedAt  string `json:"reserved_at"` // the reserved Plus use (RFC 3339), "" when none was reserved
}

func isoTime(t time.Time) string { return t.In(civildate.Tehran).Format(time.RFC3339) }

func strPtr(s string) *string { return &s }

func parseStored(raw db.NullRawJSON) *Stored {
	if !raw.Valid || len(raw.V) == 0 {
		return nil
	}
	var s Stored
	if err := json.Unmarshal(raw.V, &s); err != nil || s.Schema == "" {
		return nil
	}
	return &s
}

func (s *Stored) raw() (db.NullRawJSON, error) {
	if s.Fields == nil {
		s.Fields = map[string]Value{}
	}
	if s.Items == nil {
		s.Items = []map[string]Value{}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return db.NullRawJSON{}, err
	}
	return db.NullRawJSON{V: b, Valid: true}, nil
}

// values are the document's values as they stand: the reviewed ones once the user confirmed, else what was read.
func (s *Stored) values() map[string]any {
	if s.Reviewed != nil {
		return s.Reviewed
	}
	out := make(map[string]any, len(s.Fields))
	for k, v := range s.Fields {
		out[k] = v.Value
	}
	return out
}

func (s *Stored) reservedAt() (time.Time, bool) {
	if s.Job == nil || s.Job.ReservedAt == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s.Job.ReservedAt)
	return t.In(civildate.Tehran), err == nil
}
