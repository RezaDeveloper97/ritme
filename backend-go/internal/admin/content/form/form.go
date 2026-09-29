// Package form holds the request/response helpers shared by the admin content, message
// and language endpoints (T-M2-21): validation with extra checks the rule engine does not
// have (unique, url, gte, required_with, uploads), translatable columns and JSON output.
package form

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Add records one extra validation message for a field. A field that already failed a
// rule keeps its first message only (Laravel reports one message per failed rule, and
// the extra checks stand in for the rules that come after the engine's).
type Add func(field, msg string)

// Check is an extra validation step over the raw input (the rule engine's input). It
// returns an error only for infrastructure failures (DB), never for invalid input.
type Check func(in phpval.Map, add Add) error

// Validate runs the Laravel rules plus the extra checks and returns the validated data,
// or the admin 422 with every message (rules first, then the checks).
func Validate(c fiber.Ctx, rules validation.Rules, checks ...Check) (phpval.Map, error) {
	return ValidateInput(c, validation.Input(c), rules, checks...)
}

// ValidateInput is Validate over a given input instead of the request's (e.g. a stored
// payload merged with a partial update, T-M7-06).
func ValidateInput(c fiber.Ctx, in phpval.Map, rules validation.Rules, checks ...Check) (phpval.Map, error) {
	v := validation.Make(lang.Default(), i18n.Locale(c), in, rules, validation.Now(httpadmin.Now(c)))
	ve := httpx.NewValidationError()
	if v.Fails() {
		for _, f := range v.Fields() {
			for _, m := range v.Messages(f) {
				ve.Add(f, m)
			}
		}
	}
	add := func(field, msg string) {
		if len(ve.Messages(field)) == 0 {
			ve.Add(field, msg)
		}
	}
	for _, check := range checks {
		if err := check(in, add); err != nil {
			return nil, err
		}
	}
	if !ve.Empty() {
		return nil, httpadmin.Invalid(ve)
	}
	return v.Validated(), nil
}

// Msg is a validation message for field in the request locale ("validation.unique", …).
func Msg(c fiber.Ctx, key, field string, params ...string) string {
	p := map[string]string{"attribute": httpadmin.AttributeName(c, field)}
	for i := 0; i+1 < len(params); i += 2 {
		p[params[i]] = params[i+1]
	}
	return httpadmin.Trans(c, key, p)
}

// Translatable is Translatable::rules() over the request's active languages.
func Translatable(c fiber.Ctx, field string, required bool, extra ...string) validation.Rules {
	return i18n.TranslatableRules(field, required, i18n.LanguagesOf(c), extra...)
}

// ---------------------------------------------------------------------------
// Reading validated data

// Has reports whether key is present and not null.
func Has(data phpval.Map, key string) bool {
	v, ok := data.Get(key)
	return ok && v != nil
}

// Str is a nullable string column (null when missing or null).
func Str(data phpval.Map, key string) sql.NullString {
	if !Has(data, key) {
		return sql.NullString{}
	}
	return sql.NullString{String: httpadmin.String(data, key), Valid: true}
}

// Int is an integer value (`$data[key] ?? def`), clamped to the int32 range the columns use.
func Int(data phpval.Map, key string, def int64) int64 {
	if !Has(data, key) {
		return def
	}
	v, _ := data.Get(key)
	f := phpval.ToFloat(v)
	return int64(math.Max(math.MinInt32, math.Min(math.MaxInt32, math.Trunc(f))))
}

// Int32 is Int as int32.
func Int32(data phpval.Map, key string, def int64) int32 {
	return int32(Int(data, key, def)) //nolint:gosec // G115: clamped by Int
}

// NullInt16 is a nullable small integer column.
func NullInt16(data phpval.Map, key string) sql.NullInt16 {
	if !Has(data, key) {
		return sql.NullInt16{}
	}
	n := Int(data, key, 0)
	return sql.NullInt16{Int16: int16(max(math.MinInt16, min(math.MaxInt16, n))), Valid: true} //nolint:gosec // G115: clamped
}

// JSON encodes a value for a JSON column as raw UTF-8 (json_encode with JSON_UNESCAPED_UNICODE |
// JSON_UNESCAPED_SLASHES), the way the seed rows are stored. Plain json_encode wrote every Persian
// letter as \uXXXX: the same meaning, but ~6× the bytes and invisible to a `LIKE '%فارسی%'` on the
// DB (QA 2026-09-29-c L8). Rows Laravel stored escaped stay as they are; readers decode JSON, and
// the admin search matches both spellings (Contains + ContainsJSON).
func JSON(v any) json.RawMessage {
	b, err := jsonx.Marshal(v, jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}

// NullJSON is a nullable JSON column: SQL NULL when key is missing or null.
func NullJSON(data phpval.Map, key string) db.NullRawJSON {
	if !Has(data, key) {
		return db.NullRawJSON{}
	}
	v, _ := data.Get(key)
	return db.NullRawJSON{V: JSON(v), Valid: true}
}

// ReqJSON is a NOT NULL JSON column (validated as required).
func ReqJSON(data phpval.Map, key string) json.RawMessage {
	v, _ := data.Get(key)
	return JSON(v)
}

// Clean is Translatable::clean on a validated translatable value: the languages with
// text, or SQL NULL when none has any.
func Clean(data phpval.Map, key string) db.NullRawJSON {
	col := NullJSON(data, key)
	if !col.Valid {
		return col
	}
	cleaned := i18n.Clean(col.V)
	if cleaned == nil {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: cleaned, Valid: true}
}

// ---------------------------------------------------------------------------
// Output

// Raw decodes a JSON column for output (null when empty or invalid).
func Raw(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	v, err := phpval.Decode(raw)
	if err != nil {
		return nil
	}
	return phpval.Packed(v)
}

// NullRaw is Raw for a nullable JSON column.
func NullRaw(col db.NullRawJSON) any {
	if !col.Valid {
		return nil
	}
	return Raw(col.V)
}

// NullInt is a nullable integer or null.
func NullInt(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return n.Int16
}

// ---------------------------------------------------------------------------
// Rules the engine lacks

// IsURL is Laravel's `url` rule for the common cases: an absolute http(s)/ftp-style URL
// with a host and no whitespace.
func IsURL(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t\r\n") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "ftp", "ftps":
		return true
	}
	return false
}

// ParseTime parses a datetime input the way Carbon::parse reads the admin forms'
// values ("2026-09-23", "2026-09-23 10:00[:00]", "2026-09-23T10:00[:00]", RFC 3339),
// as Asia/Tehran wall-clock.
func ParseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		"2006-01-02", "2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02T15:04", "2006-01-02T15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, s, civildate.Tehran); err == nil {
			return t, true
		}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(civildate.Tehran), true
	}
	return time.Time{}, false
}

// EscapeLike escapes LIKE wildcards so a value matches literally.
func EscapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Exact is a LIKE pattern matching s exactly ("%" when s is empty: no filter).
func Exact(s string) string {
	if s == "" {
		return "%"
	}
	return EscapeLike(s)
}

// Contains is a LIKE pattern for a substring search ("%" when s is empty).
func Contains(s string) string {
	if s == "" {
		return "%"
	}
	return "%" + EscapeLike(s) + "%"
}

// ContainsJSON is Contains for text as PHP's json_encode writes it inside a JSON column
// (non-ASCII as \uXXXX, "/" as \/), so a Persian search also finds rows Laravel stored.
func ContainsJSON(s string) string {
	if s == "" {
		return "%"
	}
	b, err := jsonx.Marshal(s, 0)
	if err != nil || len(b) < 2 {
		return Contains(s)
	}
	return "%" + EscapeLike(string(b[1:len(b)-1])) + "%"
}

// BoolRange maps a status filter onto [min, max] for `col >= min AND col <= max`.
func BoolRange(only *bool) (lo, hi bool) {
	if only == nil {
		return false, true
	}
	return *only, *only
}

// ---------------------------------------------------------------------------
// Updates: Eloquent's update($validated) only writes the keys it was given, so an
// optional field missing from the request keeps its stored value. Present-but-null
// (or an empty string, which ConvertEmptyStringsToNull turns into null) clears it.

// KeepJSON is NullJSON when key was sent, else cur.
func KeepJSON(data phpval.Map, key string, cur db.NullRawJSON) db.NullRawJSON {
	if _, ok := data.Get(key); ok {
		return NullJSON(data, key)
	}
	return cur
}

// KeepStr is Str when key was sent, else cur.
func KeepStr(data phpval.Map, key string, cur sql.NullString) sql.NullString {
	if _, ok := data.Get(key); ok {
		return Str(data, key)
	}
	return cur
}

// KeepInt16 is NullInt16 when key was sent, else cur.
func KeepInt16(data phpval.Map, key string, cur sql.NullInt16) sql.NullInt16 {
	if _, ok := data.Get(key); ok {
		return NullInt16(data, key)
	}
	return cur
}

// KeepTime is the parsed datetime when key was sent (null when empty), else cur.
func KeepTime(data phpval.Map, key string, cur sql.NullTime) sql.NullTime {
	if _, ok := data.Get(key); !ok {
		return cur
	}
	if t, ok := ParseTime(Str(data, key).String); ok {
		return sql.NullTime{Time: t, Valid: true}
	}
	return sql.NullTime{}
}
