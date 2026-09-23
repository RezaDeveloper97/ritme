package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Cast is an Eloquent attribute cast ($casts value) as it affects toArray().
type Cast string

// The casts the Ritme models use. A column without a cast serialises as the raw PDO value:
// integers as numbers, tinyint(1) as 0/1, DECIMAL as the DB string ("60.50"), DATE as
// "Y-m-d", TIMESTAMP/DATETIME as "Y-m-d H:i:s" (Tehran wall-clock), TIME as "H:i:s" and
// JSON as the stored JSON text (a string). created_at/updated_at are always CastDateTime.
const (
	CastNone     Cast = ""
	CastBoolean  Cast = "boolean"
	CastInteger  Cast = "integer"
	CastFloat    Cast = "float"
	CastDecimal1 Cast = "decimal:1"
	CastDecimal2 Cast = "decimal:2"
	CastDate     Cast = "date"       // midnight Tehran → UTC ("2026-09-22T20:30:00.000000Z")
	CastDateYMD  Cast = "date:Y-m-d" // "2026-09-23"
	CastDateTime Cast = "datetime"   // "2026-09-23T06:30:00.000000Z"
	CastArray    Cast = "array"      // json_decode($v, true) re-encoded
)

// Casts maps column names to their cast.
type Casts map[string]Cast

// Attributes is Model::toArray() for a sqlc row struct generated from `SELECT *`: every
// column in table order (sqlc keeps it), named by its snake_case column name, with the
// model's casts applied. hidden columns are left out ($hidden).
//
// row must be a struct (or pointer to one) whose fields are sqlc's column mapping: the
// field name is the camel-cased column (`user_id` → UserID) and the type one of the sqlc
// types of this repo (ints, string, bool, sql.Null*, time.Time, civildate.Date/NullDate,
// json.RawMessage, db.NullRawJSON). A nil pointer serialises as nil (JSON null).
func Attributes(row any, casts Casts, hidden ...string) any {
	rv := reflect.ValueOf(row)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("model.Attributes: %T is not a struct", row))
	}
	out := jsonx.NewObject()
	rt := rv.Type()
	for i := range rt.NumField() {
		col := ColumnName(rt.Field(i).Name)
		if contains(hidden, col) {
			continue
		}
		cast := casts[col]
		if col == "created_at" || col == "updated_at" {
			cast = CastDateTime
		}
		out.Set(col, castValue(col, cast, sqlValue(rv.Field(i).Interface())))
	}
	return out
}

// List serialises a slice of rows (a Collection's toArray()); nil/empty → [].
func List[T any](rows []T, casts Casts, hidden ...string) []any {
	out := make([]any, 0, len(rows))
	for i := range rows {
		out = append(out, Attributes(&rows[i], casts, hidden...))
	}
	return out
}

// ColumnName turns a sqlc field name back into its column: UserID → user_id,
// BasalBodyTemperature → basal_body_temperature (sqlc's only initialism is "id").
func ColumnName(field string) string {
	field = strings.ReplaceAll(field, "ID", "Id")
	var b strings.Builder
	for i, r := range field {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// sqlValue unwraps a sqlc field into nil, int64, string, bool, time.Time,
// civildate.Date or json.RawMessage.
func sqlValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case int64:
		return x
	case int32:
		return int64(x)
	case int16:
		return int64(x)
	case int8:
		return int64(x)
	case int:
		return int64(x)
	case uint64:
		return int64(x) //nolint:gosec // G115: ids and counters fit int64
	case uint32:
		return int64(x)
	case uint16:
		return int64(x)
	case uint8:
		return int64(x)
	case string, bool, time.Time, civildate.Date:
		return x
	case json.RawMessage:
		return x
	case sql.NullString:
		return nullOr(x.Valid, x.String)
	case sql.NullInt64:
		return nullOr(x.Valid, x.Int64)
	case sql.NullInt32:
		return nullOr(x.Valid, int64(x.Int32))
	case sql.NullInt16:
		return nullOr(x.Valid, int64(x.Int16))
	case sql.NullByte:
		return nullOr(x.Valid, int64(x.Byte))
	case sql.NullBool:
		return nullOr(x.Valid, x.Bool)
	case sql.NullTime:
		return nullOr(x.Valid, x.Time)
	case civildate.NullDate:
		return nullOr(x.Valid, x.Date)
	case sql.Null[json.RawMessage]:
		return nullOr(x.Valid, x.V)
	}
	panic(fmt.Sprintf("model: unsupported column type %T", v))
}

func nullOr(valid bool, v any) any {
	if !valid {
		return nil
	}
	return v
}

// castValue applies Model::castAttribute + serializeDate for toArray().
func castValue(col string, cast Cast, v any) any {
	if v == nil {
		return nil
	}
	switch cast {
	case CastNone:
		return rawValue(v)
	case CastBoolean:
		switch x := v.(type) {
		case bool:
			return x
		case int64:
			return x != 0
		}
		return phpval.Truthy(rawValue(v))
	case CastInteger:
		switch x := v.(type) {
		case int64:
			return x
		case bool:
			return map[bool]int64{true: 1, false: 0}[x]
		}
		return int64(phpval.ToFloat(rawValue(v)))
	case CastFloat:
		return jsonx.Float(phpval.ToFloat(rawValue(v)))
	case CastDecimal1, CastDecimal2:
		scale, _ := strconv.Atoi(strings.TrimPrefix(string(cast), "decimal:"))
		d, err := jsonx.Decimal(rawValue(v), scale)
		if err != nil {
			panic(fmt.Sprintf("model: %s: %v", col, err))
		}
		return d
	case CastDate:
		return jsonx.DateCast(dateOf(v))
	case CastDateYMD:
		return dateOf(v)
	case CastDateTime:
		if t, ok := v.(time.Time); ok {
			return jsonx.DateTime(t)
		}
		return jsonx.DateTime(dateOf(v).TehranMidnight())
	case CastArray:
		raw, ok := v.(json.RawMessage)
		if !ok {
			return nil
		}
		return DecodeJSONColumn(raw)
	}
	panic(fmt.Sprintf("model: %s: unsupported cast %q", col, cast))
}

// rawValue is the uncast PDO value.
func rawValue(v any) any {
	switch x := v.(type) {
	case bool: // tinyint(1) comes back from PDO as an int
		if x {
			return int64(1)
		}
		return int64(0)
	case time.Time:
		return x.In(civildate.Tehran).Format(time.DateTime)
	case civildate.Date:
		return x.String()
	case json.RawMessage:
		return string(x)
	}
	return v
}

func dateOf(v any) civildate.Date {
	switch x := v.(type) {
	case civildate.Date:
		return x
	case time.Time:
		return civildate.InTehran(x)
	}
	panic(fmt.Sprintf("model: %T is not a date", v))
}

// DecodeJSONColumn is the `array` cast: json_decode($raw, true), ready for jsonx (PHP
// arrays keep their key order, an empty object becomes [], floats encode like PHP).
// Undecodable text is null, as json_decode returns.
func DecodeJSONColumn(raw json.RawMessage) any {
	v, err := phpval.Decode(raw)
	if err != nil {
		return nil
	}
	return phpJSON(v)
}

func phpJSON(v any) any {
	switch x := v.(type) {
	case float64:
		return jsonx.Float(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = phpJSON(e)
		}
		return out
	case phpval.Map:
		out := jsonx.NewArray()
		for _, k := range x.Keys() {
			e, _ := x.Get(k)
			out.Set(k, phpJSON(e))
		}
		return out
	}
	return v
}
