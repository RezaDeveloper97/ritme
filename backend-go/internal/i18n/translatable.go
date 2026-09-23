package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ritme/backend-go/internal/platform/validation"
)

// Translatable columns are JSON objects keyed by language code: {"fa": "…", "en": "…", "ar": "…"}.
// The helpers below take the raw column and return the raw JSON of the chosen value, so
// whatever Laravel would echo (a string, a nested object, a list) is passed through as is.

type entry struct {
	key string
	val json.RawMessage
}

// entries decodes a JSON object (or list, keyed "0", "1", …) into ordered raw entries.
// ok=false when raw is not an array (a scalar or null), which the helpers return as is.
func entries(raw json.RawMessage) ([]entry, bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, false
	}
	delim, isDelim := tok.(json.Delim)
	if !isDelim {
		return nil, false
	}
	var out []entry
	for i := 0; dec.More(); i++ {
		key := fmt.Sprint(i)
		if delim == '{' {
			kt, err := dec.Token()
			if err != nil {
				return nil, false
			}
			key, _ = kt.(string)
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, false
		}
		out = append(out, entry{key, val})
	}
	if _, err := dec.Token(); err != nil && !errors.Is(err, io.EOF) {
		return nil, false
	}
	return out, true
}

func isNullOrEmpty(v json.RawMessage) bool {
	t := bytes.TrimSpace(v)
	return len(t) == 0 || bytes.Equal(t, []byte("null")) || bytes.Equal(t, []byte(`""`))
}

func isNull(v json.RawMessage) bool {
	t := bytes.TrimSpace(v)
	return len(t) == 0 || bytes.Equal(t, []byte("null"))
}

func lookup(es []entry, key string) (json.RawMessage, bool) {
	for _, e := range es {
		if e.key == key {
			return e.val, true
		}
	}
	return nil, false
}

// Pick is Translatable::pick / HasLocalizedContent::localized: the requested locale,
// then the default language, then the first non-empty translation; null when every
// translation is empty. A non-array value (plain string, number, null) is returned as is.
func Pick(raw json.RawMessage, locale, defaultCode string) json.RawMessage {
	es, ok := entries(raw)
	if !ok {
		if len(bytes.TrimSpace(raw)) == 0 {
			return json.RawMessage("null")
		}
		return raw
	}
	for _, code := range []string{locale, defaultCode} {
		if v, ok := lookup(es, code); ok && !isNullOrEmpty(v) {
			return v
		}
	}
	for _, e := range es {
		if !isNullOrEmpty(e.val) {
			return e.val
		}
	}
	return json.RawMessage("null")
}

// PickString is Pick decoded as a string ("" when the pick is null or not a string).
func PickString(raw json.RawMessage, locale, defaultCode string) string {
	var s string
	_ = json.Unmarshal(Pick(raw, locale, defaultCode), &s)
	return s
}

// PickChain is `$value[$locale] ?? $value['fa'] ?? $value['en'] ?? null` — the
// null-coalescing pick PhaseContent::getLocalizedContent and AbstractHomeSection::pick
// use (an empty string counts as present). Call it as PickChain(raw, locale, LegacyPair...).
// A non-array value is returned as is.
func PickChain(raw json.RawMessage, codes ...string) json.RawMessage {
	es, ok := entries(raw)
	if !ok {
		if len(bytes.TrimSpace(raw)) == 0 {
			return json.RawMessage("null")
		}
		return raw
	}
	for _, code := range codes {
		if v, ok := lookup(es, code); ok && !isNull(v) {
			return v
		}
	}
	return json.RawMessage("null")
}

// PickOrWhole is PregnancyWeeklyContent::getLocalizedContent: the locale's value when
// the key is set (and not null), otherwise the whole column unchanged — so a locale
// without content (e.g. "ar") gets the full {"en": …, "fa": …} object.
func PickOrWhole(raw json.RawMessage, locale string) json.RawMessage {
	if es, ok := entries(raw); ok {
		if v, ok := lookup(es, locale); ok && !isNull(v) {
			return v
		}
	}
	return raw
}

// Clean is Translatable::clean: the object without null/"" translations, or nil
// (SQL NULL) when nothing is left or raw is not an array.
func Clean(raw json.RawMessage) json.RawMessage {
	es, ok := entries(raw)
	if !ok {
		return nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	n := 0
	for _, e := range es {
		if isNullOrEmpty(e.val) {
			continue
		}
		if n > 0 {
			buf.WriteByte(',')
		}
		k, _ := json.Marshal(e.key)
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(bytes.TrimSpace(e.val))
		n++
	}
	if n == 0 {
		return nil
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

// TranslatableRules is Translatable::rules(): the field must be an array (required or
// nullable), the default language's entry is required when the field is, every other
// active language is optional; extra rules ("max:255") apply to each language.
func TranslatableRules(field string, required bool, langs Languages, extra ...string) validation.Rules {
	presence := "nullable"
	if required {
		presence = "required"
	}
	rules := validation.Rules{validation.F(field, []string{presence, "array"})}
	def := langs.DefaultCode()
	for _, code := range langs.Codes() {
		p := "nullable"
		if required && code == def {
			p = "required"
		}
		rules = append(rules, validation.F(field+"."+code, append([]string{p, "string"}, extra...)))
	}
	return rules
}
