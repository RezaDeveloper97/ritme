package languages

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// codePattern is the stored (normalised) shape of a language code. Codes become directory
// names on the storage volume, so anything else is refused before touching the disk.
var codePattern = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})?$`)

// namePattern is a safe file-name stem for namespaces and lang groups.
var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

var errUnsafe = errors.New("languages: unsafe code or namespace")

// Bundles is the write side of App\Services\Language\TranslationStore plus the D-04 lang
// files: UI-string bundles under STORAGE_PATH/app/translations/<code>/<ns>.json (the same
// files Laravel's editor writes and both APIs read) and the translator's lang groups as
// JSON under STORAGE_PATH/app/lang/<code>/<group>.json (Laravel wrote lang/<code>/*.php
// into its image, which a deploy wiped — D-04).
type Bundles struct {
	store   *i18n.TranslationStore // read side (seed + live)
	live    string                 // STORAGE_PATH/app/translations
	langDir string                 // STORAGE_PATH/app/lang
	langFS  fs.FS                  // embedded resources/lang (<locale>/<group>.json)
}

// NewBundles wires the store over storagePath ("" = writes fail).
func NewBundles(store *i18n.TranslationStore, storagePath string, langFS fs.FS) *Bundles {
	b := &Bundles{store: store, langFS: langFS}
	if storagePath != "" {
		b.live = filepath.Join(storagePath, "app", "translations")
		b.langDir = filepath.Join(storagePath, "app", "lang")
	}
	return b
}

// Namespaces lists the namespaces (default language's seed ∪ live).
func (b *Bundles) Namespaces(defaultCode string) []string { return b.store.Namespaces(defaultCode) }

// Raw is the language's own strings for a namespace (no default backfill).
func (b *Bundles) Raw(code, ns string) any { return b.store.RawNamespace(code, ns) }

func (b *Bundles) dir(root, code string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("languages: STORAGE_PATH not configured: %w", errUnsafe)
	}
	if !codePattern.MatchString(code) {
		return "", errUnsafe
	}
	return filepath.Join(root, code), nil
}

// WriteNamespace is TranslationStore::writeNamespace: empty leaves are dropped (they fall
// back to the default language), pretty JSON with raw UTF-8 and slashes, plus "\n".
func (b *Bundles) WriteNamespace(code, ns string, messages any) error {
	if !namePattern.MatchString(ns) {
		return errUnsafe
	}
	dir, err := b.dir(b.live, code)
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, ns+".json"), prune(messages))
}

// GenerateFor is TranslationStore::generateFor: every namespace of source (backfilled from
// the default language) is written for code. It returns the number of files written.
func (b *Bundles) GenerateFor(code, source, defaultCode string) (int, error) {
	n := 0
	for _, ns := range b.store.Namespaces(defaultCode) {
		msgs := b.store.NamespaceMessages(source, ns, defaultCode)
		if phpval.Count(msgs) == 0 {
			continue
		}
		if err := b.WriteNamespace(code, ns, msgs); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// CopyLangFiles is LanguageProvisioner::copyLangFiles on the volume (D-04): source's lang
// groups (embedded seed, or a volume copy made for an earlier language) are copied to
// app/lang/<code>/ unless the target already exists. It returns the files copied.
func (b *Bundles) CopyLangFiles(code, source string) (int, error) {
	if code == source {
		return 0, nil
	}
	dst, err := b.dir(b.langDir, code)
	if err != nil {
		return 0, err
	}
	files := map[string][]byte{}
	if entries, err := fs.ReadDir(b.langFS, source); err == nil {
		for _, e := range entries {
			if data, err := fs.ReadFile(b.langFS, source+"/"+e.Name()); err == nil && !e.IsDir() {
				files[e.Name()] = data
			}
		}
	}
	if srcDir, err := b.dir(b.langDir, source); err == nil {
		if entries, err := os.ReadDir(srcDir); err == nil {
			for _, e := range entries {
				if _, seen := files[e.Name()]; seen || e.IsDir() {
					continue
				}
				if data, err := os.ReadFile(filepath.Join(srcDir, e.Name())); err == nil { //nolint:gosec // G304: code-checked dir
					files[e.Name()] = data
				}
			}
		}
	}
	if len(files) == 0 {
		return 0, nil
	}
	copied := 0
	for name, data := range files {
		if filepath.Ext(name) != ".json" || !namePattern.MatchString(strings.TrimSuffix(name, ".json")) {
			continue
		}
		target := filepath.Join(dst, name)
		if _, err := os.Stat(target); err == nil {
			continue // never clobber a translation someone already wrote
		}
		if err := writeFile(target, data); err != nil {
			return copied, err
		}
		copied++
	}
	return copied, nil
}

// DeleteFor removes a deleted language's live bundles and lang files.
func (b *Bundles) DeleteFor(code string) error {
	for _, root := range []string{b.live, b.langDir} {
		if root == "" {
			continue
		}
		dir, err := b.dir(root, code)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("languages: remove %s: %w", dir, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Files

func writeJSON(path string, v any) error {
	b, err := jsonx.Marshal(v, jsonx.PrettyPrint|jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	if err != nil {
		return fmt.Errorf("languages: encode %s: %w", filepath.Base(path), err)
	}
	return writeFile(path, append(b, '\n'))
}

// writeFile writes atomically (temp file + rename), 0644 in a 0755 directory.
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: read by both API containers
		return fmt.Errorf("languages: mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("languages: create: %w", err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("languages: write: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("languages: chmod: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("languages: close: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("languages: rename: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Array helpers (Arr::dot / Arr::undot / prune)

// prune drops null and "" leaves and arrays left empty (TranslationStore::prune).
func prune(v any) any {
	out := phpval.NewMap()
	keys, vals := phpval.Entries(v)
	for i, k := range keys {
		x := vals[i]
		if phpval.IsArray(x) {
			if nested := prune(x); phpval.Count(nested) > 0 {
				out.Set(k, nested)
			}
			continue
		}
		if x == nil || x == "" {
			continue
		}
		out.Set(k, x)
	}
	return phpval.Packed(out)
}

// Flatten is TranslationStore::flatten: Arr::dot with every leaf as a string (non-scalar
// leaves, e.g. empty arrays, become "").
func Flatten(v any) map[string]string {
	out := map[string]string{}
	var walk func(prefix string, x any)
	walk = func(prefix string, x any) {
		keys, vals := phpval.Entries(x)
		for i, k := range keys {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if phpval.IsArray(vals[i]) && phpval.Count(vals[i]) > 0 {
				walk(key, vals[i])
				continue
			}
			switch leaf := vals[i].(type) {
			case string, bool, int64, float64:
				out[key] = phpval.ToString(leaf)
			default:
				out[key] = ""
			}
		}
	}
	walk("", v)
	return out
}

// Unflatten is Arr::undot: "a.b.c" → {"a":{"b":{"c":…}}}; lists come back as JSON arrays.
func Unflatten(flat map[string]string, order []string) any {
	root := phpval.NewMap()
	for _, key := range order {
		val, ok := flat[key]
		if !ok {
			continue
		}
		segs := strings.Split(key, ".")
		cur := root
		for i, seg := range segs {
			if i == len(segs)-1 {
				cur.Set(seg, val)
				break
			}
			next, exists := cur.Get(seg)
			m, isMap := next.(phpval.Map)
			if !exists || !isMap {
				m = phpval.NewMap()
				cur.Set(seg, m)
			}
			cur = m
		}
	}
	return packAll(root)
}

func packAll(v any) any {
	m, ok := v.(phpval.Map)
	if !ok {
		return v
	}
	out := phpval.NewMap()
	for _, k := range m.Keys() {
		x, _ := m.Get(k)
		out.Set(k, packAll(x))
	}
	return phpval.Packed(out)
}

// SortKeys is PHP sort() on string keys: numeric strings compare as numbers, others
// byte-wise.
func SortKeys(keys []string) {
	sort.SliceStable(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		fa, errA := strconv.ParseFloat(a, 64)
		fb, errB := strconv.ParseFloat(b, 64)
		if errA == nil && errB == nil && phpval.IsNumericString(a) && phpval.IsNumericString(b) {
			return fa < fb
		}
		return a < b
	})
}
