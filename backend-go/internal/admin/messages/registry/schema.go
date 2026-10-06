// Package registry is the admin-side registry of message_contents groups (T-M7-06,
// docs/pregnancy-v2/README.md § Message engine integration, point 4): which (group, item_key)
// pairs exist and the shape of each payload. POST /messages may only create rows for a
// registered pair, the alert-rule editor reads its typed params from here, and the list
// endpoint reports the registered rows that are still missing per language.
//
// Two kinds of groups:
//   - the smart-message groups of the code fallback (messages/content defaults.json): keys and a
//     payload shape derived from the embedded default copy (strings and string lists);
//   - the pregnancy v2 groups (pregnancy_week_tip, pregnancy_alert, pregnancy_setup) with an
//     explicit schema — typed params, enums, nested objects.
package registry

import (
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Kind is a payload field type.
type Kind string

// Field kinds.
const (
	KindText       Kind = "text"        // string (nullable: may be null)
	KindTextList   Kind = "text_list"   // list of strings
	KindInt        Kind = "integer"     // integer in [Min, Max]
	KindBool       Kind = "boolean"     // true / false
	KindEnum       Kind = "enum"        // one of Values
	KindEnumList   Kind = "enum_list"   // distinct values of Values
	KindURL        Kind = "url"         // absolute http(s) URL or an in-app path "/…" (nullable)
	KindObject     Kind = "object"      // nested Fields
	KindObjectList Kind = "object_list" // list of objects with Fields; a `key` field is distinct
)

// Field is one payload key.
type Field struct {
	Key      string
	Kind     Kind
	Nullable bool
	Min, Max int      // integer bounds
	MaxLen   int      // text / url length (0 = 2000)
	MinItems int      // lists
	MaxItems int      // lists (0 = 20)
	Values   []string // enum values
	Fields   []Field  // object / object_list item fields
}

const (
	defaultMaxLen   = 2000
	defaultMaxItems = 20
	urlMaxLen       = 1000
)

func (f Field) maxLen() int {
	switch {
	case f.MaxLen > 0:
		return f.MaxLen
	case f.Kind == KindURL:
		return urlMaxLen
	}
	return defaultMaxLen
}

func (f Field) maxItems() int {
	if f.MaxItems > 0 {
		return f.MaxItems
	}
	return defaultMaxItems
}

func presence(nullable bool) string {
	if nullable {
		return "nullable"
	}
	return "required"
}

func listRule(f Field) string {
	r := "present|array|max:" + strconv.Itoa(f.maxItems())
	if f.Nullable {
		r = "nullable|array|max:" + strconv.Itoa(f.maxItems())
	}
	if f.MinItems > 0 {
		r += "|min:" + strconv.Itoa(f.MinItems)
	}
	return r
}

// Rules are the Laravel rules of fields under prefix ("payload." / "texts.fa." / "").
func Rules(prefix string, fields []Field) validation.Rules {
	var out validation.Rules
	for _, f := range fields {
		attr := prefix + f.Key
		switch f.Kind {
		case KindText:
			out = append(out, validation.F(attr, presence(f.Nullable)+"|string|max:"+strconv.Itoa(f.maxLen())))
		case KindURL:
			out = append(out, validation.F(attr, "nullable|string|max:"+strconv.Itoa(f.maxLen())))
		case KindTextList:
			out = append(out,
				validation.F(attr, listRule(f)),
				validation.F(attr+".*", "required|string|max:"+strconv.Itoa(f.maxLen())))
		case KindInt:
			out = append(out, validation.F(attr, presence(f.Nullable)+"|integer|min:"+strconv.Itoa(f.Min)+"|max:"+strconv.Itoa(f.Max)))
		case KindBool:
			out = append(out, validation.F(attr, presence(f.Nullable)+"|boolean"))
		case KindEnum:
			out = append(out, validation.F(attr, presence(f.Nullable)+"|string", validation.In(f.Values...)))
		case KindEnumList:
			out = append(out,
				validation.F(attr, listRule(f)),
				validation.F(attr+".*", "required|string", validation.In(f.Values...)))
		case KindObject:
			p := presence(f.Nullable)
			if !f.Nullable && len(f.Fields) == 0 {
				p = "present" // `required` fails on an empty array; a param-less rule stores {}
			}
			out = append(out, validation.F(attr, p+"|array"))
			out = append(out, Rules(attr+".", f.Fields)...)
		case KindObjectList:
			out = append(out,
				validation.F(attr, listRule(f)),
				validation.F(attr+".*", "required|array"))
			out = append(out, Rules(attr+".*.", f.Fields)...)
		}
	}
	return out
}

// Check is the part of the validation the rule engine lacks: URL shape, distinct enum-list
// values and distinct `key`s in object lists. prefix is the rules' prefix; value the object
// the fields live in (the raw input at that prefix).
func Check(prefix string, fields []Field, value any, add form.Add, msg func(rule, field string) string) {
	for _, f := range fields {
		v, ok := phpval.Get(value, f.Key)
		if !ok || v == nil {
			continue
		}
		attr := prefix + f.Key
		switch f.Kind {
		case KindText:
			if s, isStr := v.(string); isStr && s != "" && f.Key == ContactPhoneParam && !ValidContactPhone(s) {
				add(attr, msg("validation.regex", attr))
			}
		case KindURL:
			if s, isStr := v.(string); isStr && !IsLink(s) {
				add(attr, msg("validation.url", attr))
			}
		case KindEnumList:
			seen := map[string]bool{}
			keys, vals := phpval.Entries(v)
			for i, x := range vals {
				s, isStr := x.(string)
				if !isStr {
					continue
				}
				if seen[s] {
					add(attr+"."+keys[i], msg("validation.distinct", attr+"."+keys[i]))
				}
				seen[s] = true
			}
		case KindObject:
			if phpval.IsArray(v) {
				Check(attr+".", f.Fields, v, add, msg)
			}
		case KindObjectList:
			if !phpval.IsArray(v) {
				continue
			}
			seen := map[string]bool{}
			keys, vals := phpval.Entries(v)
			for i, item := range vals {
				if !phpval.IsArray(item) {
					continue
				}
				Check(attr+"."+keys[i]+".", f.Fields, item, add, msg)
				k, _ := phpval.Get(item, "key")
				if s, isStr := k.(string); isStr && s != "" {
					if seen[s] {
						field := attr + "." + keys[i] + ".key"
						add(field, msg("validation.distinct", field))
					}
					seen[s] = true
				}
			}
		}
	}
}

// IsLink accepts an absolute http(s) URL or an in-app path with a single leading slash.
func IsLink(s string) bool {
	if strings.HasPrefix(s, "/") {
		return !strings.HasPrefix(s, "//") && !strings.ContainsAny(s, " \t\r\n\\")
	}
	if !form.IsURL(s) {
		return false
	}
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")
}

// Build maps a validated value onto the schema: only known keys, in schema order, typed.
func Build(fields []Field, value any) *jsonx.OrderedMap {
	out := jsonx.NewObject()
	for _, f := range fields {
		v, _ := phpval.Get(value, f.Key)
		out.Set(f.Key, buildValue(f, v))
	}
	return out
}

func buildValue(f Field, v any) any {
	switch f.Kind {
	case KindText, KindURL, KindEnum:
		if v == nil {
			return nil
		}
		s := phpval.ToString(v)
		if s == "" && f.Nullable {
			return nil
		}
		return s
	case KindInt:
		if v == nil {
			return nil
		}
		return int64(phpval.ToFloat(v))
	case KindBool:
		switch x := v.(type) {
		case bool:
			return x
		case string:
			return x == "1" || strings.EqualFold(x, "true")
		case nil:
			return false
		}
		return phpval.Truthy(v)
	case KindTextList, KindEnumList:
		_, vals := phpval.Entries(v)
		list := make([]any, 0, len(vals))
		for _, x := range vals {
			if x != nil {
				list = append(list, phpval.ToString(x))
			}
		}
		return list
	case KindObject:
		if v == nil && f.Nullable {
			return nil
		}
		return Build(f.Fields, v)
	case KindObjectList:
		_, vals := phpval.Entries(v)
		list := make([]any, 0, len(vals))
		for _, x := range vals {
			list = append(list, Build(f.Fields, x))
		}
		return list
	}
	return v
}

// Schema is the fields as JSON for admin-web (so it can render a form per item).
func Schema(fields []Field) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(fields))
	for _, f := range fields {
		o := jsonx.Obj("key", f.Key, "kind", string(f.Kind), "nullable", f.Nullable)
		switch f.Kind {
		case KindText, KindURL, KindTextList:
			o.Set("max_length", f.maxLen())
		case KindInt:
			o.Set("min", f.Min)
			o.Set("max", f.Max)
		}
		if f.Kind == KindEnum || f.Kind == KindEnumList {
			o.Set("values", jsonx.List(f.Values))
		}
		if f.Kind == KindTextList || f.Kind == KindEnumList || f.Kind == KindObjectList {
			o.Set("min_items", f.MinItems)
			o.Set("max_items", f.maxItems())
		}
		if f.Kind == KindObject || f.Kind == KindObjectList {
			o.Set("fields", Schema(f.Fields))
		}
		out = append(out, o)
	}
	return out
}
