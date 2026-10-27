package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Go-native engine v6 goldens (T-M9-32, DECISIONS.md X-04, D-120).
//
//	contract v6 [--routes cycle] [--case <glob>] [--update] [-v]
//
// The Laravel goldens (contract/golden) characterise v1.1 and keep running with CYCLE_ENGINE=v1 (`contract diff`).
// This mode starts the Go server with CYCLE_ENGINE=v6 on the same fixture and replays the cases that only read
// /cycle/today or /cycle/date (every step a GET on those paths); --update (re)writes contract/golden-v6/<group>/
// <case>.json from Go, without it the run diffs against them. Goldens are reviewed by hand (the §2.35 example persona
// against the spec). --routes all means the cycle group here (the sweep groups would add thousands of files).

// v6Env is the server environment of the v6 run.
var v6Env = map[string]string{"CYCLE_ENGINE": "v6"}

// v6Paths are the request paths whose answer the v6 engine builds.
var v6Paths = []string{"/api/v1/cycle/today", "/api/v1/cycle/date/"}

// isV6Case reports whether every step of c is a GET the v6 engine answers.
func isV6Case(c *Case) bool {
	if len(c.Steps) == 0 {
		return false
	}
	for _, s := range c.Steps {
		if s.Method != http.MethodGet {
			return false
		}
		ok := false
		for _, p := range v6Paths {
			if s.Path == strings.TrimSuffix(p, "/") || strings.HasPrefix(s.Path, p) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func goldenV6Path(root string, c *Case) string {
	return filepath.Join(root, "golden-v6", c.Group, c.ID+".json")
}

// runV6 records (update) or diffs the v6 goldens.
func runV6(ctx context.Context, paths Paths, opts options, update bool) error {
	if opts.routes == "all" {
		opts.routes = "cycle"
	}
	sel, err := selectCases(paths, opts)
	if err != nil {
		return err
	}
	for i := range sel {
		var kept []*Case
		for _, c := range sel[i].cases {
			if isV6Case(c) {
				kept = append(kept, c)
			}
		}
		sel[i].cases = kept
	}
	dump, err := os.ReadFile(paths.Dump)
	if err != nil {
		return fmt.Errorf("%s missing (run make contract-record): %w", paths.Dump, err)
	}
	if !update {
		for _, s := range sel {
			for _, c := range s.cases {
				if !fileExists(goldenV6Path(paths.Contract, c)) {
					return fmt.Errorf("no v6 golden for %s/%s (run make contract-v6-update ROUTES=%s)", c.Group, c.ID, c.Group)
				}
			}
		}
	}

	g, err := NewGoServer(paths)
	if err != nil {
		return err
	}
	g.ExtraEnv = v6Env
	defer g.Stop()
	if err := g.Start(ctx, dump); err != nil {
		return err
	}
	runner := NewRunner(g)

	var failed, passed, written int
	for _, s := range sel {
		start := time.Now()
		keep := map[string]bool{}
		for _, c := range s.cases {
			results, runErr := runner.Run(ctx, c)
			if runErr != nil {
				failed++
				fmt.Printf("FAIL %s/%s\n    error: %v\n", c.Group, c.ID, runErr)
				continue
			}
			path := goldenV6Path(paths.Contract, c)
			if update {
				doc, probs := BuildGolden(c, results)
				for _, p := range probs {
					fmt.Printf("PROBLEM %s/%s: %s\n", c.Group, c.ID, strings.Replace(p, "Laravel answered", "Go (v6) answered", 1))
				}
				if len(probs) > 0 {
					failed++
				}
				if err := writeIfChanged(path, doc); err != nil {
					return err
				}
				keep[filepath.Base(path)] = true
				written++
				continue
			}
			golden, err := LoadGolden(path)
			if err != nil {
				return err
			}
			var lines []string
			if len(golden) != len(c.Steps) {
				lines = append(lines, fmt.Sprintf("golden has %d steps, case has %d (re-run with --update)", len(golden), len(c.Steps)))
			}
			for i := range results {
				if i >= len(golden) {
					break
				}
				for _, m := range CompareStep(i+1, &c.Steps[i], &golden[i], &results[i]) {
					lines = append(lines, fmt.Sprintf("%s  [%s %s]", m, results[i].Method, results[i].URL))
				}
			}
			if len(lines) > 0 {
				failed++
				fmt.Printf("FAIL %s/%s\n", c.Group, c.ID)
				for _, l := range lines {
					fmt.Println("    " + l)
				}
				continue
			}
			passed++
			if opts.verbose {
				fmt.Printf("ok   %s/%s\n", c.Group, c.ID)
			}
		}
		if update && opts.caseGlob == "" {
			if err := removeStale(filepath.Join(paths.Contract, "golden-v6", s.group), keep); err != nil {
				return err
			}
		}
		logf("v6 %-14s %4d cases in %s", s.group, len(s.cases), time.Since(start).Round(time.Millisecond))
	}
	if update {
		fmt.Printf("contract v6 --update: %d goldens written, %d problem(s)\n", written, failed)
	} else {
		fmt.Printf("contract v6: %d passed, %d failed\n", passed, failed)
	}
	if failed > 0 {
		if os.Getenv("CONTRACT_SHOW_LOG") != "" {
			fmt.Fprintln(os.Stderr, g.LogTail())
		}
		return fmt.Errorf("%d v6 case(s) failed", failed)
	}
	return nil
}
