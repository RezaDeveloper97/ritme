package catalog

import (
	"encoding/json"
	"regexp"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// langKeyRe is what a language code looks like ("fa", "en", "pt-br"); the list itself is data.
var langKeyRe = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})?$`)

// Localizer picks translations for one request: locale, then the default language, then the
// first non-empty translation (i18n.Pick semantics).
type Localizer struct {
	Locale  string
	Default string
	Langs   i18n.Languages
}

// Text is a translatable column picked as a string (nil when every translation is empty).
func (l Localizer) Text(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(i18n.Pick(raw, l.Locale, l.Default), &s); err != nil || s == "" {
		return nil
	}
	return s
}

// Meta decodes meta and replaces every translation object inside it — an object whose keys all look
// like language codes and include at least one active language — by its picked value, recursively.
// nil when meta is empty or not valid JSON.
func (l Localizer) Meta(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	v, err := phpval.Decode(raw)
	if err != nil {
		return nil
	}
	return l.walk(v)
}

func (l Localizer) walk(v any) any {
	switch a := v.(type) {
	case []any:
		out := make([]any, len(a))
		for i, x := range a {
			out[i] = l.walk(x)
		}
		return out
	case *jsonx.OrderedMap:
		if l.isTranslation(a) {
			return l.walk(l.pick(a))
		}
		out := jsonx.NewObject()
		for _, k := range a.Keys() {
			x, _ := a.Get(k)
			out.Set(k, l.walk(x))
		}
		return out
	}
	return v
}

func (l Localizer) isTranslation(m *jsonx.OrderedMap) bool {
	keys := m.Keys()
	if len(keys) == 0 {
		return false
	}
	active := false
	for _, k := range keys {
		if !langKeyRe.MatchString(k) {
			return false
		}
		if l.Langs.IsSupported(k) {
			active = true
		}
	}
	return active
}

func empty(v any) bool {
	s, isStr := v.(string)
	return v == nil || (isStr && s == "")
}

func (l Localizer) pick(m *jsonx.OrderedMap) any {
	for _, code := range []string{l.Locale, l.Default} {
		if v, ok := m.Get(code); ok && !empty(v) {
			return v
		}
	}
	for _, k := range m.Keys() {
		if v, _ := m.Get(k); !empty(v) {
			return v
		}
	}
	return nil
}

// Public is an item as GET /catalog/{group} returns it.
func (l Localizer) Public(it Item) *jsonx.OrderedMap {
	var auds any
	if len(it.Audiences) > 0 {
		auds = jsonx.List(it.Audiences)
	}
	return jsonx.Obj(
		"code", it.Code,
		"title", l.Text(it.Title),
		"body", l.Text(it.Body),
		"meta", l.Meta(it.Meta),
		"audiences", auds,
		"needs_review", it.NeedsReview,
	)
}
