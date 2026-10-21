// Command gate is the benchmark CI gate (T-M9-08, DECISIONS.md X-09): it compares two `go test -bench` outputs
// by the median of each benchmark's samples and
//
//   - fails (exit 1) when any allocs/op median goes above the baseline's (a deterministic count: an allocation
//     regression is a code change, never noise). -alloc-slack (0.1 % of the baseline) absorbs the few allocations
//     amortised map/slice growth adds or drops between runs of a large benchmark (MonthJSON/full: 26 949–26 956);
//     below 1 000 allocs/op it is under one allocation, so +1 still fails;
//   - warns (a GitHub `::warning::` annotation, exit 0) when a sec/op median is more than -time-threshold (20 %)
//     slower — shared CI runners are too noisy to fail on time.
//
// Benchmarks are matched by package + name without the -GOMAXPROCS suffix, so a 10-core baseline still matches a
// 4-core runner. Allocations are compared with -baseline (the committed bench/baseline.txt); time with -time-base
// when given (a run of the base commit on the same runner), else with -baseline; -skip-time compares allocations
// only (CI without a base run: the baseline is from another machine).
//
//	go run ./bench/gate -baseline bench/baseline.txt -current bench/current.txt [-time-base bench/base-ref.txt | -skip-time]
//
// Exit codes: 0 ok (warnings allowed), 1 allocs regression, 2 bad input.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// Units the gate reads.
const (
	unitTime   = "sec/op"
	unitAllocs = "allocs/op"
)

// result is one benchmark's samples per unit (time normalised to sec/op).
type result map[string][]float64

// file is a parsed benchmark output: results by key (pkg + "." + name) and the header's Go version.
type file struct {
	results map[string]result
	order   []string
	goVer   string
}

var (
	resultLine = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?\s+\d+\s+(.+)$`)
	goHeader   = regexp.MustCompile(`^#\s*go:\s*(\S+)`)
)

func parse(r io.Reader) (*file, error) {
	f := &file{results: map[string]result{}}
	pkg := ""
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if m := goHeader.FindStringSubmatch(line); m != nil {
			f.goVer = m[1]
			continue
		}
		if p, ok := strings.CutPrefix(line, "pkg: "); ok {
			pkg = strings.TrimSpace(p)
			continue
		}
		m := resultLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fields := strings.Fields(m[2])
		if len(fields)%2 != 0 {
			return nil, fmt.Errorf("malformed result line: %q", line)
		}
		key := pkg + "." + m[1]
		res, ok := f.results[key]
		if !ok {
			res = result{}
			f.results[key] = res
			f.order = append(f.order, key)
		}
		for i := 0; i < len(fields); i += 2 {
			v, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				return nil, fmt.Errorf("value %q in %q: %w", fields[i], line, err)
			}
			unit := fields[i+1]
			if unit == "ns/op" {
				unit, v = unitTime, v/1e9
			}
			res[unit] = append(res[unit], v)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return f, nil
}

func parseFile(path string) (*file, error) {
	fh, err := os.Open(path) //nolint:gosec // a path the caller names
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	f, err := parse(fh)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(f.results) == 0 {
		return nil, fmt.Errorf("%s: no benchmark results", path)
	}
	return f, nil
}

func median(xs []float64) (float64, bool) {
	if len(xs) == 0 {
		return 0, false
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2], true
	}
	return (s[n/2-1] + s[n/2]) / 2, true
}

func (f *file) median(key, unit string) (float64, bool) {
	r, ok := f.results[key]
	if !ok {
		return 0, false
	}
	return median(r[unit])
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	baselinePath := fs.String("baseline", "bench/baseline.txt", "committed baseline (allocs reference)")
	currentPath := fs.String("current", "bench/current.txt", "this run")
	timeBasePath := fs.String("time-base", "", "time reference from the same machine (default: -baseline)")
	threshold := fs.Float64("time-threshold", 0.20, "warn when sec/op is slower than this fraction")
	allocSlack := fs.Float64("alloc-slack", 0.001, "allocs/op tolerance as a fraction of the baseline (amortised map/slice growth)")
	skipTime := fs.Bool("skip-time", false, "compare allocs only (no same-machine time reference)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	base, err := parseFile(*baselinePath)
	if err == nil && *baselinePath == *currentPath {
		err = errors.New("-baseline and -current are the same file")
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "gate:", err)
		return 2
	}
	cur, err := parseFile(*currentPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "gate:", err)
		return 2
	}
	timeBase := base
	if *timeBasePath != "" {
		if timeBase, err = parseFile(*timeBasePath); err != nil {
			_, _ = fmt.Fprintln(stderr, "gate:", err)
			return 2
		}
	}
	if *skipTime {
		timeBase = nil
	}
	return compare(base, timeBase, cur, *threshold, *allocSlack, stdout)
}

func compare(base, timeBase, cur *file, threshold, allocSlack float64, out io.Writer) int {
	p := func(format string, a ...any) { _, _ = fmt.Fprintf(out, format+"\n", a...) }
	if base.goVer != "" && cur.goVer != "" && base.goVer != cur.goVer {
		p("::warning title=bench gate::baseline was recorded with %s, this run is %s — allocs can differ between Go releases; re-record bench/baseline.txt with the new toolchain", base.goVer, cur.goVer)
	}
	failed, warned := 0, 0
	for _, key := range cur.order {
		ca, okA := cur.median(key, unitAllocs)
		ba, okB := base.median(key, unitAllocs)
		switch {
		case !okB && !okA:
		case !okB:
			if _, known := base.results[key]; !known {
				p("::notice title=new benchmark::%s is not in the baseline — re-record bench/baseline.txt (make bench-baseline)", key)
			}
		case okA && ca > ba*(1+allocSlack):
			failed++
			p("::error title=allocs/op regression::%s: %s → %s allocs/op", key, num(ba), num(ca))
		}
		if timeBase == nil {
			continue
		}
		ct, okT := cur.median(key, unitTime)
		bt, okBT := timeBase.median(key, unitTime)
		if okT && okBT && bt > 0 && ct > bt*(1+threshold) {
			warned++
			p("::warning title=sec/op regression::%s: %s → %s (+%.0f%%, threshold %.0f%%)", key, dur(bt), dur(ct), (ct/bt-1)*100, threshold*100)
		}
	}
	for _, key := range base.order {
		if _, ok := cur.results[key]; !ok {
			p("::warning title=missing benchmark::%s is in the baseline but did not run", key)
		}
	}
	p("bench gate: %d benchmarks, %d allocs/op regressions, %d sec/op warnings (> %.0f%%)", len(cur.order), failed, warned, threshold*100)
	if failed > 0 {
		return 1
	}
	return 0
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func dur(sec float64) string {
	switch {
	case sec >= 1e-3:
		return fmt.Sprintf("%.2fms", sec*1e3)
	case sec >= 1e-6:
		return fmt.Sprintf("%.2fµs", sec*1e6)
	default:
		return fmt.Sprintf("%.0fns", sec*1e9)
	}
}
