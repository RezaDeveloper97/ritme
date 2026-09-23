// Command enumgen turns the Laravel string-backed enums (backend/app/Enums/*.php and
// backend/app/Services/MessageSystem/Enums/*.php) into Go: one internal/enums/zz_generated_<name>.go
// per enum with the typed string, its constants (PHP case order, exact backed values), Values(),
// IsValid(), From(), Label/Description(locale), Icon() and Options(locale).
//
// Only value/label tables are generated; behaviour (fertility mapping, regularity, …) is hand-ported
// in internal/enums/*_logic.go. Run through `go generate ./internal/enums/...` from backend-go/.
//
// After the Laravel backend is retired (T-M2-27) the generated files are kept and this tool deleted.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// enumDirs are scanned relative to the Laravel backend root.
var enumDirs = []string{"app/Enums", "app/Services/MessageSystem/Enums"}

func main() {
	backend := flag.String("backend", "../../../backend", "path to the Laravel backend root")
	out := flag.String("out", ".", "output directory (internal/enums)")
	flag.Parse()

	if err := run(*backend, *out); err != nil {
		fmt.Fprintln(os.Stderr, "enumgen:", err)
		os.Exit(1)
	}
}

func run(backend, out string) error {
	var enums []*phpEnum
	seen := map[string]string{}
	for _, dir := range enumDirs {
		files, err := filepath.Glob(filepath.Join(backend, dir, "*.php"))
		if err != nil {
			return err
		}
		sort.Strings(files)
		for _, f := range files {
			src, err := os.ReadFile(f) //nolint:gosec // dev tool reading the repo's own PHP sources
			if err != nil {
				return err
			}
			rel := "backend/" + filepath.ToSlash(filepath.Join(dir, filepath.Base(f)))
			e, err := parseEnum(string(src), rel)
			if err != nil {
				return err
			}
			if prev, dup := seen[e.Name]; dup {
				return fmt.Errorf("enum %s defined twice (%s, %s)", e.Name, prev, rel)
			}
			seen[e.Name] = rel
			enums = append(enums, e)
		}
	}
	if len(enums) == 0 {
		return fmt.Errorf("no enums found under %s", backend)
	}

	// Drop stale output so a deleted PHP enum disappears from Go too.
	old, err := filepath.Glob(filepath.Join(out, "zz_generated_*.go"))
	if err != nil {
		return err
	}
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	for _, e := range enums {
		code, err := render(e)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name, err)
		}
		name := filepath.Join(out, "zz_generated_"+snake(e.Name)+".go")
		if err := os.WriteFile(name, code, 0o644); err != nil { //nolint:gosec // generated source file
			return err
		}
	}
	reg, err := renderRegistry(enums)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "zz_generated_registry_test.go"), reg, 0o644) //nolint:gosec // generated source file
}

// snake turns CycleSubphase into cycle_subphase.
func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// pascal turns UPPER_SNAKE (or HOURS_0_3) into UpperSnake (Hours03).
func pascal(s string) string {
	var b strings.Builder
	for _, part := range strings.Split(s, "_") {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(strings.ToLower(part[1:]))
	}
	return b.String()
}
