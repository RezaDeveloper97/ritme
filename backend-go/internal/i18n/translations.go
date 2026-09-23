package i18n

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// TranslationStore is App\Services\Language\TranslationStore (read side): the UI-string
// bundles the frontend renders from, one folder per language and one JSON file per
// namespace. Two layers, the second winning key by key:
//
//  1. the seed shipped with the binary (backend-go/resources/translations, a copy of
//     backend/resources/translations);
//  2. the live, admin-edited files under STORAGE_PATH/app/translations/<code>/<ns>.json
//     (the backend-storage volume Laravel writes to).
//
// Every read is backfilled from the default language, so an untranslated key shows the
// default text instead of nothing.
type TranslationStore struct {
	seed fs.FS
	live string // directory holding <code>/<ns>.json ("" = no live layer)

	mu    sync.Mutex
	cache map[string]any // decoded seed files by path (the seed is immutable)
}

// NewTranslationStore reads seed (usually translations.FS) and the live files under
// storagePath/app/translations.
func NewTranslationStore(seed fs.FS, storagePath string) *TranslationStore {
	live := ""
	if storagePath != "" {
		live = filepath.Join(storagePath, "app", "translations")
	}
	return &TranslationStore{seed: seed, live: live, cache: map[string]any{}}
}

// Namespaces is every namespace of the default language (seed ∪ live), sorted.
func (s *TranslationStore) Namespaces(defaultCode string) []string {
	seen := map[string]bool{}
	var names []string
	add := func(entries []fs.DirEntry) {
		for _, e := range entries {
			if e.IsDir() || path.Ext(e.Name()) != ".json" {
				continue
			}
			n := strings.TrimSuffix(e.Name(), ".json")
			if !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}
	if entries, err := fs.ReadDir(s.seed, defaultCode); err == nil {
		add(entries)
	}
	if s.live != "" {
		if entries, err := os.ReadDir(filepath.Join(s.live, defaultCode)); err == nil {
			add(entries)
		}
	}
	sort.Strings(names)
	return names
}

// Bundle is the complete message bundle for code (namespace → nested messages), what
// GET /languages/{code}/messages returns under data.messages. code must already be a
// resolved, active language.
func (s *TranslationStore) Bundle(code, defaultCode string) phpval.Map {
	bundle := phpval.NewMap()
	for _, ns := range s.Namespaces(defaultCode) {
		bundle.Set(ns, s.NamespaceMessages(code, ns, defaultCode))
	}
	return bundle
}

// NamespaceMessages is one namespace for code with the default language underneath.
func (s *TranslationStore) NamespaceMessages(code, ns, defaultCode string) any {
	var base any = phpval.NewMap()
	if code != defaultCode {
		base = s.RawNamespace(defaultCode, ns)
	}
	return phpval.Packed(deepMerge(base, s.RawNamespace(code, ns)))
}

// RawNamespace is code's own strings for ns (seed with live on top, no default backfill).
func (s *TranslationStore) RawNamespace(code, ns string) any {
	seed := phpval.Clone(s.seedFile(code + "/" + ns + ".json"))
	return deepMerge(seed, s.liveFile(code, ns))
}

func (s *TranslationStore) seedFile(name string) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.cache[name]; ok {
		return v
	}
	var v any = phpval.NewMap()
	if b, err := fs.ReadFile(s.seed, name); err == nil {
		v = decodeArray(b)
	}
	s.cache[name] = v
	return v
}

func (s *TranslationStore) liveFile(code, ns string) any {
	if s.live == "" || strings.ContainsAny(code+ns, `/\`) || code == ".." || ns == ".." {
		return phpval.NewMap()
	}
	b, err := os.ReadFile(filepath.Join(s.live, code, ns+".json")) //nolint:gosec // code/ns contain no path separators (checked above)
	if err != nil {
		return phpval.NewMap()
	}
	return decodeArray(b)
}

// decodeArray is readJson(): json_decode(…, true), anything but an array becomes [].
// Lists are kept as maps keyed "0", "1", … so deepMerge can treat both alike.
func decodeArray(b []byte) any {
	v, err := phpval.Decode(b)
	if err != nil || !phpval.IsArray(v) {
		return phpval.NewMap()
	}
	return asMap(v)
}

func asMap(v any) any {
	switch a := v.(type) {
	case []any, phpval.Map:
		m := phpval.NewMap()
		keys, vals := phpval.Entries(a)
		for i, k := range keys {
			m.Set(k, asMap(vals[i]))
		}
		return m
	}
	return v
}

// deepMerge is TranslationStore::deepMerge: override wins key by key, arrays merge
// recursively, and null/"" overrides are ignored (so they inherit the base).
func deepMerge(base, override any) any {
	b, ok := base.(phpval.Map)
	if !ok {
		b = phpval.NewMap()
	}
	o, ok := override.(phpval.Map)
	if !ok {
		return b
	}
	for _, k := range o.Keys() {
		val, _ := o.Get(k)
		if vm, isMap := val.(phpval.Map); isMap {
			if cur, exists := b.Get(k); exists {
				if cm, curIsMap := cur.(phpval.Map); curIsMap {
					b.Set(k, deepMerge(cm, vm))
					continue
				}
			}
		}
		if val == nil || val == "" {
			continue
		}
		b.Set(k, val)
	}
	return b
}
