package lang

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// checkInterval bounds how often a locale's storage directory is stat'ed: a request can
// ask for dozens of lines, and an admin edit showing up a second later is fine.
var checkInterval = time.Second

// localePattern keeps request locales from escaping the lang directory (codes are
// normalised lower-case by the admin API; the loader is lenient about case and "_").
var localePattern = regexp.MustCompile(`^[A-Za-z]{2,3}([-_][A-Za-z0-9]{2,8})?$`)

// WithStorage returns a copy of t that overlays storagePath/app/lang/<locale>/<group>.json
// on the embedded groups (storage wins, per group). "" returns t unchanged.
func (t *Translator) WithStorage(storagePath string) *Translator {
	if storagePath == "" {
		return t
	}
	c := *t
	c.storage = &storageOverlay{dir: filepath.Join(storagePath, "app", "lang"), locales: map[string]*localeFiles{}}
	return &c
}

// storageOverlay caches the decoded storage groups per locale, reloading a locale when
// the set, size or mtime of its *.json files changes (the admin writes temp + rename).
type storageOverlay struct {
	dir     string
	mu      sync.Mutex
	locales map[string]*localeFiles
}

type localeFiles struct {
	checked time.Time
	sig     string
	groups  map[string]any
}

func (s *storageOverlay) groups(locale string) map[string]any {
	if !localePattern.MatchString(locale) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lf := s.locales[locale]
	now := time.Now()
	if lf != nil && now.Sub(lf.checked) < checkInterval {
		return lf.groups
	}
	dir := filepath.Join(s.dir, locale)
	sig, files := scan(dir)
	if lf != nil && lf.sig == sig {
		lf.checked = now
		return lf.groups
	}
	lf = &localeFiles{checked: now, sig: sig, groups: load(dir, files)}
	s.locales[locale] = lf
	return lf.groups
}

// scan lists dir's group files with a signature of their names, sizes and mtimes.
func scan(dir string) (string, []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil // no directory = no overlay
	}
	var b strings.Builder
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || filepath.Ext(name) != ".json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, name)
		fmt.Fprintf(&b, "%s|%d|%d\n", name, info.Size(), info.ModTime().UnixNano())
	}
	return b.String(), files
}

// load decodes the group files; an unreadable or malformed file is skipped (the embedded
// group, if any, stays in effect) and logged once per change.
func load(dir string, files []string) map[string]any {
	groups := make(map[string]any, len(files))
	for _, name := range files {
		p := filepath.Join(dir, name)
		data, err := os.ReadFile(p) //nolint:gosec // G304: locale matched localePattern, name from ReadDir
		if err == nil {
			var v any
			if v, err = phpval.Decode(data); err == nil {
				groups[strings.TrimSuffix(name, ".json")] = v
				continue
			}
		}
		slog.Warn("lang: ignoring storage lang file", "file", p, "err", err)
	}
	return groups
}
