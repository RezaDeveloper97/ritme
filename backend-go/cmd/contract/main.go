// Command contract is the Laravel → Go contract harness (T-M2-05).
//
//	contract record [--routes <groups|all>] [--case <glob>]   record Laravel goldens
//	contract diff   [--routes <groups|all>] [--case <glob>]   diff a fresh Go server against them
//
// Cases: contract/cases/<group>.yaml. Goldens: contract/golden/<group>/<case>.json.
// Fixtures: contract/fixtures/dump.sql (seeded DB) and contract/fixtures/keys/ (the
// contract-only Passport pair). Allow-lists: contract/allowlist/<group>.yaml.
// Case file format: see the doc comments in cases.go.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

func main() { os.Exit(run()) }

func run() int {
	if len(os.Args) < 2 {
		usage()
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	routes := fs.String("routes", "all", "case groups: all, or comma-separated names/globs (cycle*)")
	caseGlob := fs.String("case", "", "only cases whose id matches this glob")
	verbose := fs.Bool("v", false, "print passing cases too")
	reseed := fs.Bool("reseed", false, "record: re-run contract-reset and re-export dump.sql (implied by --routes all)")
	_ = fs.Parse(os.Args[2:])

	paths, err := findPaths()
	if err != nil {
		return fail(err)
	}
	opts := options{routes: *routes, caseGlob: *caseGlob, verbose: *verbose, reseed: *reseed}
	switch os.Args[1] {
	case "record":
		err = record(ctx, paths, opts)
	case "diff":
		err = diff(ctx, paths, opts)
	default:
		usage()
		return 2
	}
	if err != nil {
		return fail(err)
	}
	return 0
}

type options struct {
	routes   string
	caseGlob string
	verbose  bool
	reseed   bool
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: contract record|diff [--routes <groups|all>] [--case <glob>] [-v]")
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "contract:", err)
	return 1
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "contract: "+format+"\n", args...)
}

// Paths are the harness's file locations, resolved from the backend-go module root.
type Paths struct {
	GoRoot, RepoRoot, Contract, Keys, Dump, Work string
	ContractCompose, TestCompose, Deviations     string
}

func findPaths() (Paths, error) {
	dir, err := os.Getwd()
	if err != nil {
		return Paths{}, err
	}
	for {
		mod, err := os.ReadFile(filepath.Join(dir, "go.mod")) //nolint:gosec // G304: module discovery
		if err == nil && bytes.Contains(mod, []byte("module github.com/ritme/backend-go")) {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Paths{}, errors.New("run from inside backend-go/ (go.mod not found)")
		}
		dir = parent
	}
	repo := filepath.Dir(dir)
	c := filepath.Join(dir, "contract")
	return Paths{
		GoRoot: dir, RepoRoot: repo, Contract: c,
		Keys:            filepath.Join(c, "fixtures", "keys"),
		Dump:            filepath.Join(c, "fixtures", "dump.sql"),
		Work:            filepath.Join(c, ".work"),
		ContractCompose: filepath.Join(repo, "docker-compose.contract.yml"),
		TestCompose:     filepath.Join(dir, "docker-compose.test.yml"),
		Deviations:      filepath.Join(repo, "docs", "go-migration", "deviations.md"),
	}, nil
}

type selection struct {
	group string
	cases []*Case
}

func selectCases(paths Paths, opts options) ([]selection, error) {
	all, err := LoadGroups(filepath.Join(paths.Contract, "cases"))
	if err != nil {
		return nil, err
	}
	names, err := SelectGroups(all, opts.routes)
	if err != nil {
		return nil, err
	}
	var out []selection
	for _, n := range names {
		cases, err := all[n].Expand()
		if err != nil {
			return nil, err
		}
		if opts.caseGlob != "" {
			var kept []*Case
			for _, c := range cases {
				if ok, _ := filepath.Match(opts.caseGlob, c.ID); ok {
					kept = append(kept, c)
				}
			}
			cases = kept
		}
		out = append(out, selection{group: n, cases: cases})
	}
	return out, nil
}

// record replays the selected cases against Laravel and rewrites their goldens.
// Recording runs are serialised with a lock file (they share one Laravel stack).
func record(ctx context.Context, paths Paths, opts options) error {
	sel, err := selectCases(paths, opts)
	if err != nil {
		return err
	}
	unlock, err := lockFile(filepath.Join(paths.Work, "record.lock"))
	if err != nil {
		return err
	}
	defer unlock()

	lv := NewLaravel(paths)
	if err := lv.EnsureUp(ctx); err != nil {
		return err
	}
	if err := lv.SyncKeys(ctx); err != nil {
		return fmt.Errorf("keys: %w", err)
	}
	// A full recording (or --reseed) rebuilds the fixture DB and re-exports dump.sql;
	// a partial one restores the committed dump (fast, and dump.sql stays the one spec).
	if opts.reseed || opts.routes == "all" || !fileExists(paths.Dump) {
		if err := lv.Reseed(ctx); err != nil {
			return err
		}
		if err := writeIfChanged(paths.Dump, lv.Dump()); err != nil {
			return err
		}
	} else if err := lv.LoadDump(paths.Dump); err != nil {
		return err
	}
	minter, err := NewMinter(filepath.Join(paths.Keys, "oauth-private.key"), ContractClientID)
	if err != nil {
		return err
	}
	if err := lv.Login(ctx, minter); err != nil {
		return err
	}

	runner := NewRunner(lv)
	var problems []string
	total := 0
	for _, s := range sel {
		start := time.Now()
		written := map[string]bool{}
		for _, c := range s.cases {
			results, err := runner.Run(ctx, c)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s/%s: %v", c.Group, c.ID, err))
				continue
			}
			doc, probs := BuildGolden(c, results)
			for _, p := range probs {
				problems = append(problems, fmt.Sprintf("%s/%s: %s", c.Group, c.ID, p))
			}
			path := goldenPath(paths.Contract, c)
			if err := writeIfChanged(path, doc); err != nil {
				return err
			}
			written[filepath.Base(path)] = true
			total++
		}
		if opts.caseGlob == "" {
			if err := removeStale(filepath.Join(paths.Contract, "golden", s.group), written); err != nil {
				return err
			}
		}
		logf("recorded %-14s %4d cases in %s", s.group, len(s.cases), time.Since(start).Round(time.Millisecond))
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "  PROBLEM", p)
		}
		return fmt.Errorf("%d recording problem(s) in %d cases (goldens were written; fix the cases)", len(problems), total)
	}
	logf("recorded %d cases", total)
	return nil
}

// diff runs the selected cases against a freshly started Go server.
func diff(ctx context.Context, paths Paths, opts options) error {
	sel, err := selectCases(paths, opts)
	if err != nil {
		return err
	}
	dump, err := os.ReadFile(paths.Dump)
	if err != nil {
		return fmt.Errorf("%s missing (run make contract-record): %w", paths.Dump, err)
	}
	allow := map[string]*Allowlist{}
	for _, s := range sel {
		al, err := LoadAllowlist(paths.Contract, s.group, paths.Deviations)
		if err != nil {
			return err
		}
		allow[s.group] = al
		for _, c := range s.cases {
			if !fileExists(goldenPath(paths.Contract, c)) {
				return fmt.Errorf("no golden for %s/%s (run make contract-record ROUTES=%s)", c.Group, c.ID, c.Group)
			}
		}
	}

	g, err := NewGoServer(paths)
	if err != nil {
		return err
	}
	defer g.Stop()
	if err := g.Start(ctx, dump); err != nil {
		return err
	}
	runner := NewRunner(g)

	var failed, passed, allowed int
	for _, s := range sel {
		for _, c := range s.cases {
			golden, err := LoadGolden(goldenPath(paths.Contract, c))
			if err != nil {
				return err
			}
			results, runErr := runner.Run(ctx, c)
			var lines []string
			if runErr != nil {
				lines = append(lines, "error: "+runErr.Error())
			}
			if len(golden) != len(c.Steps) {
				lines = append(lines, fmt.Sprintf("golden has %d steps, case has %d (re-record)", len(golden), len(c.Steps)))
			}
			for i := range results {
				if i >= len(golden) {
					break
				}
				for _, m := range CompareStep(i+1, &c.Steps[i], &golden[i], &results[i]) {
					if e := allow[s.group].Allowed(c.ID, m); e != nil {
						allowed++
						if opts.verbose {
							lines = append(lines, fmt.Sprintf("allowed (%s) %s", e.Deviation, m))
						}
						continue
					}
					lines = append(lines, fmt.Sprintf("%s  [%s %s]", m, results[i].Method, results[i].URL))
				}
			}
			failing := runErr != nil || len(golden) != len(c.Steps)
			for _, l := range lines {
				if !strings.HasPrefix(l, "allowed") {
					failing = true
				}
			}
			switch {
			case failing:
				failed++
				fmt.Printf("FAIL %s/%s\n", c.Group, c.ID)
				for _, l := range lines {
					fmt.Println("    " + l)
				}
			default:
				passed++
				if opts.verbose {
					fmt.Printf("ok   %s/%s\n", c.Group, c.ID)
					for _, l := range lines {
						fmt.Println("    " + l)
					}
				}
			}
		}
	}
	names := make([]string, len(sel))
	for i, s := range sel {
		names[i] = s.group
	}
	fmt.Printf("contract diff [%s]: %d passed, %d failed, %d allow-listed differences\n",
		strings.Join(names, ","), passed, failed, allowed)
	if failed > 0 {
		if os.Getenv("CONTRACT_SHOW_LOG") != "" {
			fmt.Fprintln(os.Stderr, g.LogTail())
		}
		return fmt.Errorf("%d case(s) differ", failed)
	}
	return nil
}

func writeIfChanged(path string, data []byte) error {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) { //nolint:gosec // G304: harness output
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644) //nolint:gosec // G306: committed goldens
}

func removeStale(dir string, keep map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") && !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// lockFile takes an exclusive flock (blocking), creating the file if needed.
func lockFile(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // G304: harness lock
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		logf("waiting for another recording run (%s)", path)
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
