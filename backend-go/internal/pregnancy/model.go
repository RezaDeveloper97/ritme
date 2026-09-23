package pregnancy

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// This file is the small slice of Eloquent the pregnancy models need: a model is an
// ordered map of raw attribute values (what PDO returned, or what the request filled in)
// plus the model's $casts, serialised the way toArray() does. A freshly created model only
// carries the attributes that were set (+ timestamps + id); a loaded one carries every
// column — both shapes are visible in the API.
//
// Raw values: nil, bool, int64, float64, string, civildate.Date, time.Time,
// json.RawMessage (JSON column text), []any / phpval.Map (request arrays).

type cast uint8

const (
	castNone     cast = iota
	castBool          // 'boolean'
	castYMD           // 'date:Y-m-d' → "2026-09-23"
	castDate          // plain 'date' → previous-day UTC "…T20:30:00.000000Z"
	castDateTime      // 'datetime' / timestamps
	castDecimal2      // 'decimal:2' → "65.50"
	castArray         // 'array'
)

// casts is one model's $casts (+ created_at / updated_at).
type casts map[string]cast

// model is an Eloquent model instance.
type model struct {
	casts casts
	attrs *jsonx.OrderedMap
}

func newModel(c casts) *model { return &model{casts: c, attrs: jsonx.NewObject()} }

// set is setAttribute() without the write-side conversions (they are applied when the
// row is written; the cast on read gives the same JSON).
func (m *model) set(key string, v any) *model {
	m.attrs.Set(key, v)
	return m
}

// Get is $model->{$key}: the cast value (implements alerts.Attrs).
func (m *model) Get(key string) (any, bool) {
	v, ok := m.attrs.Get(key)
	if !ok {
		return nil, false
	}
	return castValue(m.casts[key], v), true
}

// raw returns the raw attribute value.
func (m *model) raw(key string) any {
	v, _ := m.attrs.Get(key)
	return v
}

// JSON is toArray().
func (m *model) JSON() *jsonx.OrderedMap {
	out := jsonx.NewObject()
	for _, k := range m.attrs.Keys() {
		v, _ := m.Get(k)
		out.Set(k, v)
	}
	return out
}

// MarshalJSON implements json.Marshaler.
func (m *model) MarshalJSON() ([]byte, error) { return json.Marshal(m.JSON()) }

// castValue is castAttribute() for the cast types the pregnancy models use.
func castValue(c cast, v any) any {
	if v == nil {
		return nil
	}
	switch c {
	case castBool:
		return phpval.Truthy(v)
	case castYMD:
		if d, ok := toDate(v); ok {
			return d
		}
	case castDate:
		if d, ok := toDate(v); ok {
			return jsonx.DateCast(d)
		}
	case castDateTime:
		if t, ok := v.(time.Time); ok {
			return jsonx.DateTime(t)
		}
	case castDecimal2:
		if d, err := jsonx.Decimal(v, 2); err == nil {
			return d
		}
	case castArray:
		b, err := canonicalJSON(v)
		if err == nil {
			return json.RawMessage(b)
		}
	case castNone:
		if f, ok := v.(float64); ok {
			return jsonx.Float(f)
		}
	}
	return v
}

// toDate reads a date attribute (Carbon::parse semantics for strings, Tehran for instants).
func toDate(v any) (civildate.Date, bool) {
	switch x := v.(type) {
	case civildate.Date:
		return x, true
	case time.Time:
		return civildate.InTehran(x), true
	case string:
		t, err := civildate.ParseLenient(x, time.Now().In(civildate.Tehran), civildate.Tehran)
		if err != nil {
			return civildate.Date{}, false
		}
		return civildate.FromTime(t), true
	}
	return civildate.Date{}, false
}

// canonicalJSON is json_decode(json_encode($v), true) re-encoded: what an `array` cast
// column holds. Request arrays with keys 0..n-1 become lists, empty maps `[]`.
func canonicalJSON(v any) ([]byte, error) {
	if raw, ok := v.(json.RawMessage); ok {
		dec, err := phpval.Decode(raw)
		if err != nil {
			return nil, err
		}
		v = dec
	}
	return jsonx.Marshal(phpval.Packed(v), 0)
}

// ---- raw values from sqlc rows ----

func rawBool(b sql.NullBool) any {
	if !b.Valid {
		return nil
	}
	if b.Bool {
		return int64(1)
	}
	return int64(0)
}

func rawFlag(b bool) any { return rawBool(sql.NullBool{Bool: b, Valid: true}) }

func rawString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func rawInt(n sql.NullInt32) any {
	if !n.Valid {
		return nil
	}
	return int64(n.Int32)
}

func rawDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date
}

func rawTime(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return t.Time
}

func rawJSON(j db.NullRawJSON) any {
	if !j.Valid {
		return nil
	}
	return j.V
}

// ---- write-side conversions (what PDO binds after Eloquent's set mutators) ----

// errNotNull is MariaDB's "Column cannot be null" for an explicit null in a NOT NULL column.
type errNotNull string

func (e errNotNull) Error() string {
	return fmt.Sprintf("pregnancy: column %s cannot be null", string(e))
}

func wBool(v any) sql.NullBool {
	if v == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: phpval.Truthy(v), Valid: true}
}

func wFlag(key string, v any) (bool, error) {
	if v == nil {
		return false, errNotNull(key)
	}
	return phpval.Truthy(v), nil
}

func wString(v any) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: phpval.ToString(v), Valid: true}
}

func wInt(v any) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	if b, ok := v.(bool); ok {
		if b {
			return sql.NullInt32{Int32: 1, Valid: true}
		}
		return sql.NullInt32{Valid: true}
	}
	if !phpval.IsNumeric(v) {
		return sql.NullInt32{Valid: true}
	}
	return sql.NullInt32{Int32: int32(math.Round(phpval.ToFloat(v))), Valid: true}
}

func wDate(v any) civildate.NullDate {
	if v == nil {
		return civildate.NullDate{}
	}
	d, ok := toDate(v)
	return civildate.NullDate{Date: d, Valid: ok}
}

func wDecimal(v any) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	d, err := jsonx.Decimal(v, 2)
	if err != nil || !d.Valid() {
		return sql.NullString{}
	}
	return sql.NullString{String: d.String(), Valid: true}
}

func wJSON(v any) db.NullRawJSON {
	if v == nil {
		return db.NullRawJSON{}
	}
	b, err := canonicalJSON(v)
	if err != nil {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: b, Valid: true}
}

func wTime(v any) sql.NullTime {
	t, ok := v.(time.Time)
	if !ok {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// dbNow is freshTimestamp() as stored: Tehran wall-clock, whole seconds.
func dbNow(now time.Time) time.Time { return now.In(civildate.Tehran).Truncate(time.Second) }

type (
	sqlNullBool   = sql.NullBool
	sqlNullString = sql.NullString
)

func sqlTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }
