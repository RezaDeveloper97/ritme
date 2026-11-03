package messages

import (
	"database/sql"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M9-49b: the fact profile row carries the condition programs, the condition day entries of a lookback window and
// the open inbox rows as JSON lists, so a warm feed reads none of them again. Every decoder reports ok = false on a
// value it cannot read; the caller then falls back to its own query (the same answer, one round trip more).

// condLookbackDays is how far back the fact profile row carries the condition day entries: a current cycle window
// starting earlier is read with healthlog.Range as before.
const condLookbackDays = 120

// jsonBytes is a JSON aggregate column (nil = SQL NULL).
func jsonBytes(v any) ([]byte, bool) {
	switch t := v.(type) {
	case nil:
		return nil, true
	case []byte:
		return t, true
	case string:
		return []byte(t), true
	default:
		return nil, false
	}
}

// parsePrograms is condition_programs: the enrolled program codes.
func parsePrograms(v any) ([]string, bool) {
	b, ok := jsonBytes(v)
	if !ok {
		return nil, false
	}
	if b == nil {
		return []string{}, true
	}
	var out []string
	if json.Unmarshal(b, &out) != nil {
		return nil, false
	}
	return out, true
}

// parseCondEntries is cond_entries: the entries grouped by day, oldest day first (healthlog.Range's shape). Only the
// columns the condition day rules read are carried (no value_text).
func parseCondEntries(v any) ([]healthlog.DayEntries, bool) {
	b, ok := jsonBytes(v)
	if !ok {
		return nil, false
	}
	if b == nil {
		return []healthlog.DayEntries{}, true
	}
	var raw [][]*string
	if json.Unmarshal(b, &raw) != nil {
		return nil, false
	}
	byDay := map[civildate.Date][]taxonomy.Entry{}
	for _, r := range raw {
		if len(r) != 6 || r[0] == nil || r[1] == nil || r[2] == nil || r[3] == nil {
			return nil, false
		}
		d, err := civildate.Parse(*r[0])
		if err != nil {
			return nil, false
		}
		byDay[d] = append(byDay[d], taxonomy.Entry{Category: *r[1], Param: *r[2], Item: *r[3],
			Code: nullStr(r[4]), Num: nullStr(r[5])})
	}
	out := make([]healthlog.DayEntries, 0, len(byDay))
	for d, es := range byDay {
		out = append(out, healthlog.DayEntries{Date: d, Entries: es})
	}
	slices.SortFunc(out, func(a, b healthlog.DayEntries) int { return a.Date.Compare(b.Date) })
	return out, true
}

func nullStr(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

// inboxRowJSON is one inbox_rows element.
type inboxRowJSON struct {
	ID          uint64          `json:"id"`
	UserID      uint64          `json:"user_id"`
	RuleID      uint64          `json:"rule_id"`
	Version     uint32          `json:"version"`
	RuleKey     *string         `json:"rule_key"`
	Kind        string          `json:"kind"`
	Level       string          `json:"level"`
	State       string          `json:"state"`
	EvidenceKey *string         `json:"evidence_key"`
	HitCount    uint32          `json:"hit_count"`
	Evidence    json.RawMessage `json:"evidence"`
	FirstAt     *string         `json:"first_at"`
	LastAt      *string         `json:"last_at"`
	AckedAt     *string         `json:"acked_at"`
	NotedAt     *string         `json:"noted_at"`
}

// parseInboxRows is inbox_rows as ListUserOpenInbox rows (ordered by id). Timestamps are Tehran wall-clock text, as
// the driver scans them (platform/db: loc=Asia/Tehran).
func parseInboxRows(v any) ([]store.ListUserOpenInboxRow, bool) {
	b, ok := jsonBytes(v)
	if !ok {
		return nil, false
	}
	if b == nil {
		return []store.ListUserOpenInboxRow{}, true
	}
	var raw []inboxRowJSON
	if json.Unmarshal(b, &raw) != nil {
		return nil, false
	}
	out := make([]store.ListUserOpenInboxRow, len(raw))
	for i, r := range raw {
		row := store.ListUserOpenInboxRow{ID: r.ID, UserID: r.UserID, RuleID: r.RuleID, Version: r.Version,
			RuleKey: nullStr(r.RuleKey), Kind: r.Kind, Level: r.Level, State: r.State, EvidenceKey: nullStr(r.EvidenceKey),
			HitCount: r.HitCount}
		if ev := strings.TrimSpace(string(r.Evidence)); ev != "" && ev != "null" {
			e := json.RawMessage(ev)
			if ev[0] == '"' { // a JSON column embedded as a string
				var s string
				if json.Unmarshal(e, &s) != nil {
					return nil, false
				}
				e = json.RawMessage(s)
			}
			row.Evidence = db.NullRawJSON{V: e, Valid: true}
		}
		for _, t := range []struct {
			src *string
			dst *sql.NullTime
		}{{r.FirstAt, &row.FirstAt}, {r.LastAt, &row.LastAt}, {r.AckedAt, &row.AckedAt}, {r.NotedAt, &row.NotedAt}} {
			if t.src == nil {
				continue
			}
			ts, err := time.ParseInLocation(time.DateTime, *t.src, civildate.Tehran)
			if err != nil {
				return nil, false
			}
			*t.dst = sql.NullTime{Time: ts, Valid: true}
		}
		out[i] = row
	}
	slices.SortFunc(out, func(a, b store.ListUserOpenInboxRow) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		}
		return 0
	})
	return out, true
}
