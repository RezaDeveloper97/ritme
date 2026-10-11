// Package analyticsevents is first-party product analytics (bloom B-N9-08, D-89): the app posts small batches of
// allow-listed events («app_open», «log_saved», …) and the admin KPI screen reads them back as aggregates.
//
// Privacy rules (bloom README):
//   - NO health payload. An event is a name from a fixed list plus, at most, a few allow-listed enumerated props
//     (a paywall surface, a plan length). Any other name or key is rejected (422), and keys that look like health data
//     are named as such in the error. Nothing free-text is accepted, so a cycle/symptom/pregnancy/lab value cannot
//     arrive even by mistake.
//   - Only users who granted the opt-in `anonymous_stats` consent are recorded (an answer of tracking=false tells the
//     client to stop sending).
//   - The rows carry the internal user id only (never exported); they are read only as aggregates with k >= 10.
package analyticsevents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Limits of one request.
const (
	MaxBatch     = 25       // events per request
	MaxBodyBytes = 16 << 10 // request body
	MaxAge       = 72 * time.Hour
	MaxSkew      = 5 * time.Minute
	MaxProps     = 4
)

// prop describes one allowed prop value: either one of Enum (a string) or an integer in [Min, Max].
type prop struct {
	Enum     []string
	Min, Max int
}

// Spec is the allow-list: event name → allowed props. An event with no props lists an empty map.
var Spec = map[string]map[string]prop{
	"app_open":                {},
	"onboarding_started":      {},
	"onboarding_done":         {},
	"log_saved":               {"source": {Enum: []string{"quick", "full"}}},
	"plus_paywall_view":       {"surface": {Enum: []string{"home", "settings", "assistant", "labs", "telemed", "learning", "other"}}},
	"plus_purchase":           {"plan_months": {Min: 1, Max: 12}},
	"assistant_message_sent":  {},
	"lab_upload_started":      {},
	"telemed_booking_started": {},
	"learning_lesson_started": {},
}

// Names lists the allowed event names (sorted), for docs and tests.
func Names() []string {
	out := make([]string, 0, len(Spec))
	for n := range Spec {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// healthWords are fragments that mark a key as health data; such a key is refused with a dedicated message even
// before the allow-list says «unknown».
var healthWords = []string{"cycle", "period", "bleed", "symptom", "mood", "pain", "pregnan", "week", "weight", "bbt",
	"temperature", "lab", "marker", "result", "value", "medic", "drug", "dose", "note", "text", "body", "diagnos",
	"ovulat", "fertil", "flow", "sex", "ttc", "baby", "child", "age", "height", "blood", "hormone"}

// LooksLikeHealth reports whether a prop key contains a health fragment.
func LooksLikeHealth(key string) bool {
	k := strings.ToLower(key)
	for _, w := range healthWords {
		if strings.Contains(k, w) {
			return true
		}
	}
	return false
}

// Event is one validated event ready to store.
type Event struct {
	Name       string
	Props      map[string]any
	OccurredAt time.Time
}

// Problem is one validation failure: the field path ("events.0.props.foo") and a message key.
type Problem struct {
	Field string
	Code  string // unknown_event | unknown_prop | health_prop | bad_prop | bad_time | bad_shape | too_many | empty
}

type rawEvent struct {
	Name       string          `json:"name"`
	Props      json.RawMessage `json:"props"`
	OccurredAt *string         `json:"occurred_at"`
}

type rawBatch struct {
	Events []rawEvent `json:"events"`
}

// Parse validates a request body against the allow-list. now is the server clock.
func Parse(body []byte, now time.Time) ([]Event, []Problem) {
	var b rawBatch
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&b); err != nil {
		return nil, []Problem{{"events", "bad_shape"}}
	}
	if len(b.Events) == 0 {
		return nil, []Problem{{"events", "empty"}}
	}
	if len(b.Events) > MaxBatch {
		return nil, []Problem{{"events", "too_many"}}
	}
	var out []Event
	var probs []Problem
	for i, r := range b.Events {
		f := fmt.Sprintf("events.%d", i)
		spec, ok := Spec[r.Name]
		if !ok {
			probs = append(probs, Problem{f + ".name", "unknown_event"})
			continue
		}
		ev := Event{Name: r.Name, OccurredAt: now}
		if r.OccurredAt != nil {
			t, err := time.Parse(time.RFC3339, *r.OccurredAt)
			if err != nil || t.After(now.Add(MaxSkew)) || t.Before(now.Add(-MaxAge)) {
				probs = append(probs, Problem{f + ".occurred_at", "bad_time"})
				continue
			}
			ev.OccurredAt = t
		}
		ps, bad := parseProps(f, spec, r.Props)
		probs = append(probs, bad...)
		ev.Props = ps
		if len(bad) == 0 {
			out = append(out, ev)
		}
	}
	if len(probs) > 0 {
		return nil, probs
	}
	return out, nil
}

func parseProps(field string, spec map[string]prop, raw json.RawMessage) (map[string]any, []Problem) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, []Problem{{field + ".props", "bad_shape"}}
	}
	if len(m) > MaxProps {
		return nil, []Problem{{field + ".props", "bad_shape"}}
	}
	var probs []Problem
	out := map[string]any{}
	for k, v := range m {
		p, ok := spec[k]
		path := field + ".props." + k
		if !ok {
			code := "unknown_prop"
			if LooksLikeHealth(k) {
				code = "health_prop"
			}
			probs = append(probs, Problem{path, code})
			continue
		}
		if len(p.Enum) > 0 {
			s, isStr := v.(string)
			if !isStr || !slices.Contains(p.Enum, s) {
				probs = append(probs, Problem{path, "bad_prop"})
				continue
			}
			out[k] = s
			continue
		}
		n, isNum := v.(json.Number)
		i, err := int64(0), error(nil)
		if isNum {
			i, err = n.Int64()
		}
		if !isNum || err != nil || int(i) < p.Min || int(i) > p.Max {
			probs = append(probs, Problem{path, "bad_prop"})
			continue
		}
		out[k] = i
	}
	slices.SortFunc(probs, func(a, b Problem) int { return strings.Compare(a.Field, b.Field) })
	if len(out) == 0 {
		out = nil
	}
	return out, probs
}

// Store writes events.
type Store struct{ db *sql.DB }

// NewStore wires the store.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Insert stores the events of one user in a single statement.
func (s *Store) Insert(ctx context.Context, userID uint64, evs []Event, now time.Time) error {
	if len(evs) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO analytics_events (user_id, name, props, occurred_on, occurred_at, created_at) VALUES ")
	args := make([]any, 0, len(evs)*6)
	for i, e := range evs {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("(?,?,?,?,?,?)")
		var props any
		if len(e.Props) > 0 {
			raw, err := json.Marshal(e.Props)
			if err != nil {
				return err
			}
			props = string(raw)
		}
		at := e.OccurredAt.In(civildate.Tehran)
		args = append(args, userID, e.Name, props, at.Format("2006-01-02"), at, now)
	}
	_, err := s.db.ExecContext(ctx, sb.String(), args...)
	return err
}
