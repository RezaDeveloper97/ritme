package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var sampleRe = regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{(.*)\})?\s+(\S+)`)
var labelRe = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)="((?:[^"\\]|\\.)*)"`)

// routeStats is the per-route data of one scrape.
type routeStats struct {
	dur    map[float64]float64 // http_request_duration_seconds bucket le → cumulative count (all methods/statuses)
	durSum float64
	count  float64
	errors float64 // status 5xx
	qSum   float64
	qCount float64
}

type scrapeData map[string]*routeStats

func (s scrapeData) route(r string) *routeStats {
	if s[r] == nil {
		s[r] = &routeStats{dur: map[float64]float64{}}
	}
	return s[r]
}

func parseScrape(r io.Reader) (scrapeData, error) {
	out := scrapeData{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		m := sampleRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		if !strings.HasPrefix(name, "http_request_duration_seconds") && !strings.HasPrefix(name, "db_queries_per_request") {
			continue
		}
		v, err := strconv.ParseFloat(m[3], 64)
		if err != nil {
			continue
		}
		labels := map[string]string{}
		for _, l := range labelRe.FindAllStringSubmatch(m[2], -1) {
			labels[l[1]] = l[2]
		}
		route := labels["route"]
		if route == "" {
			continue
		}
		rs := out.route(route)
		switch name {
		case "http_request_duration_seconds_bucket":
			le := math.Inf(1)
			if labels["le"] != "+Inf" {
				le, _ = strconv.ParseFloat(labels["le"], 64)
			}
			rs.dur[le] += v
		case "http_request_duration_seconds_sum":
			rs.durSum += v
		case "http_request_duration_seconds_count":
			rs.count += v
			if strings.HasPrefix(labels["status"], "5") {
				rs.errors += v
			}
		case "db_queries_per_request_sum":
			rs.qSum += v
		case "db_queries_per_request_count":
			rs.qCount += v
		}
	}
	return out, sc.Err()
}

// quantile interpolates q over the delta histogram (Prometheus histogram_quantile semantics).
func quantile(q float64, delta map[float64]float64) float64 {
	les := make([]float64, 0, len(delta))
	for le := range delta {
		les = append(les, le)
	}
	sort.Float64s(les)
	if len(les) == 0 {
		return math.NaN()
	}
	total := delta[les[len(les)-1]]
	if total <= 0 {
		return math.NaN()
	}
	rank := q * total
	prevLe, prevC := 0.0, 0.0
	for _, le := range les {
		c := delta[le]
		if c >= rank {
			if math.IsInf(le, 1) {
				return prevLe
			}
			if c == prevC {
				return le
			}
			return prevLe + (le-prevLe)*(rank-prevC)/(c-prevC)
		}
		prevLe, prevC = le, c
	}
	return prevLe
}

func diffFiles(beforePath, afterPath string, w io.Writer) error {
	read := func(p string) (scrapeData, error) {
		f, err := os.Open(p) //nolint:gosec // G304: CLI argument
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		return parseScrape(f)
	}
	b, err := read(beforePath)
	if err != nil {
		return err
	}
	a, err := read(afterPath)
	if err != nil {
		return err
	}
	return render(b, a, w)
}

func render(before, after scrapeData, w io.Writer) error {
	routes := make([]string, 0, len(after))
	for r := range after {
		routes = append(routes, r)
	}
	sort.Strings(routes)
	_, _ = fmt.Fprintln(w, "| route | requests | 5xx | server p50 ms | p95 ms | p99 ms | queries/request (avg) |")
	_, _ = fmt.Fprintln(w, "|---|---:|---:|---:|---:|---:|---:|")
	for _, r := range routes {
		a, b := after[r], before[r]
		if b == nil {
			b = &routeStats{dur: map[float64]float64{}}
		}
		n := a.count - b.count
		if n <= 0 {
			continue
		}
		delta := map[float64]float64{}
		for le, c := range a.dur {
			delta[le] = c - b.dur[le]
		}
		qpr := math.NaN()
		if dq := a.qCount - b.qCount; dq > 0 {
			qpr = (a.qSum - b.qSum) / dq
		}
		_, _ = fmt.Fprintf(w, "| `%s` | %.0f | %.0f | %s | %s | %s | %s |\n", r, n, a.errors-b.errors,
			ms(quantile(0.50, delta)), ms(quantile(0.95, delta)), ms(quantile(0.99, delta)), f2(qpr))
	}
	return nil
}

func ms(sec float64) string {
	if math.IsNaN(sec) {
		return "-"
	}
	return strconv.FormatFloat(sec*1000, 'f', 1, 64)
}

func f2(v float64) string {
	if math.IsNaN(v) {
		return "-"
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
