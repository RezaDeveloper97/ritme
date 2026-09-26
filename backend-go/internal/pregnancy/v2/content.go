package v2

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Lang is the request locale and the site's default language (the fallback of every
// translatable column and message row).
type Lang struct{ Locale, Default string }

func (l Lang) codes() []string {
	if l.Locale == l.Default {
		return []string{l.Locale}
	}
	return []string{l.Locale, l.Default}
}

func (l Lang) pick(raw json.RawMessage) string { return i18n.PickString(raw, l.Locale, l.Default) }

func (l Lang) pickNull(raw db.NullRawJSON) any {
	if !raw.Valid {
		return nil
	}
	if s := l.pick(raw.V); s != "" {
		return s
	}
	return nil
}

// payload is the request-locale message row, else the default-language one; nil when none.
func payload(rows []store.ListV2MessagePayloadsRow, l Lang) map[string]any {
	for _, code := range l.codes() {
		for _, r := range rows {
			if r.Locale != code {
				continue
			}
			var m map[string]any
			if json.Unmarshal(r.Payload, &m) == nil {
				return m
			}
		}
	}
	return nil
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

// fill replaces the {name} placeholders of a message template.
func fill(tpl string, kv ...string) string {
	for i := 0; i+1 < len(kv); i += 2 {
		tpl = strings.ReplaceAll(tpl, "{"+kv[i]+"}", kv[i+1])
	}
	return tpl
}

// tipJSON is a pregnancy_week_tip row as {week, title, body, read_minutes, link}; nil when missing.
func tipJSON(week int, rows []store.ListV2MessagePayloadsRow, l Lang) any {
	p := payload(rows, l)
	if p == nil || (str(p, "title") == "" && str(p, "body") == "") {
		return nil
	}
	var minutes any
	if n, ok := p["read_minutes"].(float64); ok {
		minutes = int(n)
	}
	var link any
	if s := str(p, "article_url"); s != "" {
		link = s
	}
	return jsonx.Obj("week", week, "title", str(p, "title"), "body", str(p, "body"),
		"read_minutes", minutes, "link", link)
}

// weekTask is one {key, text} of a week's checklist.
type weekTask struct {
	Key  string          `json:"key"`
	Text json.RawMessage `json:"text"`
}

func decodeList[T any](raw db.NullRawJSON) []T {
	var out []T
	if raw.Valid {
		_ = json.Unmarshal(raw.V, &out)
	}
	return out
}

// doneKeys is a user-state row's done_task_keys as a set (and the stored order).
func doneKeys(raw db.NullRawJSON) []string {
	keys := decodeList[string](raw)
	if keys == nil {
		return []string{}
	}
	return keys
}

// tasksJSON is the week's checklist with done state.
func tasksJSON(d *store.PregnancyWeekDetail, done []string, l Lang) []any {
	out := []any{}
	if d == nil {
		return out
	}
	set := map[string]bool{}
	for _, k := range done {
		set[k] = true
	}
	for _, t := range decodeList[weekTask](d.Tasks) {
		if t.Key == "" {
			continue
		}
		out = append(out, jsonx.Obj("key", t.Key, "text", l.pick(t.Text), "done", set[t.Key]))
	}
	return out
}

// taskKeys are the known task keys of a week.
func taskKeys(d *store.PregnancyWeekDetail) map[string]bool {
	keys := map[string]bool{}
	if d != nil {
		for _, t := range decodeList[weekTask](d.Tasks) {
			keys[t.Key] = true
		}
	}
	return keys
}

// detailsJSON is the admin-defined week content in the request language.
func detailsJSON(d *store.PregnancyWeekDetail, l Lang) *jsonx.OrderedMap {
	if d == nil {
		return jsonx.Obj("size_label", nil, "illustration_key", nil, "length_cm", nil, "weight_g", nil,
			"heart_rate", nil, "headline", nil, "highlights", []any{}, "body_symptoms", []any{},
			"body_symptom_items", []any{}, "body_text", nil, "warning", nil, "reviewer_name", nil,
			"reviewed_at", nil, "sources", []any{})
	}
	type highlight struct {
		Icon  string          `json:"icon"`
		Title json.RawMessage `json:"title"`
		Body  json.RawMessage `json:"body"`
	}
	type symptom struct {
		Key   string          `json:"key"`
		Label json.RawMessage `json:"label"`
	}
	type source struct {
		Title json.RawMessage `json:"title"`
		URL   *string         `json:"url"`
	}
	highlights := []any{}
	for _, h := range decodeList[highlight](d.Highlights) {
		highlights = append(highlights, jsonx.Obj("icon", h.Icon, "title", l.pick(h.Title), "body", l.pick(h.Body)))
	}
	labels, items := []any{}, []any{}
	for _, s := range decodeList[symptom](d.BodySymptoms) {
		label := l.pick(s.Label)
		labels = append(labels, label)
		items = append(items, jsonx.Obj("key", s.Key, "label", label))
	}
	sources := []any{}
	for _, s := range decodeList[source](d.Sources) {
		sources = append(sources, jsonx.Obj("title", l.pick(s.Title), "url", s.URL))
	}
	var reviewedAt any
	if d.ReviewedAt.Valid {
		reviewedAt = d.ReviewedAt.Date.String()
	}
	return jsonx.Obj(
		"size_label", l.pickNull(d.SizeLabel),
		"illustration_key", nullStr(d.IllustrationKey.String, d.IllustrationKey.Valid),
		"length_cm", nullStr(d.LengthCm.String, d.LengthCm.Valid),
		"weight_g", nullStr(d.WeightG.String, d.WeightG.Valid),
		"heart_rate", nullStr(d.HeartRate.String, d.HeartRate.Valid),
		"headline", l.pickNull(d.Headline),
		"highlights", highlights,
		"body_symptoms", labels,
		"body_symptom_items", items,
		"body_text", l.pickNull(d.BodyText),
		"warning", l.pickNull(d.Warning),
		"reviewer_name", l.pickNull(d.ReviewerName),
		"reviewed_at", reviewedAt,
		"sources", sources,
	)
}

func nullStr(s string, valid bool) any {
	if !valid || s == "" {
		return nil
	}
	return s
}

// sizeLine is «جنین الان تقریباً به اندازهٔ یک تمشک است.»; nil when the week has no size.
func sizeLine(d *store.PregnancyWeekDetail, relation string, l Lang) any {
	if d == nil {
		return nil
	}
	size, _ := l.pickNull(d.SizeLabel).(string)
	if size == "" {
		return nil
	}
	return tr("size_line."+relation, l.Locale, "size", size)
}

func itoa(n int) string { return strconv.Itoa(n) }
