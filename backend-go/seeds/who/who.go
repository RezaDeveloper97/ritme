// Package who is the WHO Child Growth Standards seed data (bloom B-N5-02): the daily LMS parameters (Box-Cox power L,
// median M, coefficient of variation S) for age 0–1856 days (0–5 years), boys and girls, of three indicators:
//
//	weight  weight-for-age              (kg)   data/weight_{boy,girl}.csv
//	length  length/height-for-age       (cm)   data/length_{boy,girl}.csv  (recumbent length < 731 days, standing height after)
//	head    head-circumference-for-age  (cm)   data/head_{boy,girl}.csv
//
// Source: WHO Multicentre Growth Reference Study Group. WHO Child Growth Standards: Length/height-for-age,
// weight-for-age, weight-for-length, weight-for-height and body mass index-for-age: Methods and development. Geneva:
// World Health Organization, 2006; and Head circumference-for-age, arm circumference-for-age, triceps skinfold-for-age
// and subscapular skinfold-for-age: Methods and development, WHO 2007. Files: the official "expanded tables" (z-scores,
// by day) published at https://www.who.int/tools/child-growth-standards/standards — wfa-{boys,girls}-zscore-expanded-
// tables.xlsx, lhfa-{boys,girls}-zscore-expanded-tables.xlsx, hcfa-{boys,girls}-zscore-expanded-tables.xlsx — columns
// Day, L, M, S copied verbatim (no rounding), downloaded 2026-10-03. The standards are fixed reference data, so they
// are embedded (no table, no admin edit); B-N5-09's read-only viewer reads them through Table.
package who

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

//go:embed data/*.csv
var dataFS embed.FS

// Indicator is a growth measure with a WHO standard.
type Indicator string

// Indicators.
const (
	Weight Indicator = "weight"
	Length Indicator = "length"
	Head   Indicator = "head"
)

// Indicators in display order.
var Indicators = []Indicator{Weight, Length, Head}

// Sex of the standard.
type Sex string

// Sexes.
const (
	Boy  Sex = "boy"
	Girl Sex = "girl"
)

// Sexes in display order.
var Sexes = []Sex{Girl, Boy}

// MaxDay is the last age (in days) the standards cover (60 months).
const MaxDay = 1856

// LMS is one day's parameters.
type LMS struct {
	Day     int
	L, M, S float64
}

// Source is the citation shown with the data.
const Source = "WHO Child Growth Standards (WHO Multicentre Growth Reference Study Group, 2006/2007), expanded daily LMS tables, https://www.who.int/tools/child-growth-standards/standards"

var tables = sync.OnceValues(load)

func load() (map[string][]LMS, error) {
	out := map[string][]LMS{}
	for _, ind := range Indicators {
		for _, sex := range Sexes {
			name := fmt.Sprintf("data/%s_%s.csv", ind, sex)
			raw, err := dataFS.ReadFile(name)
			if err != nil {
				return nil, fmt.Errorf("who: %s: %w", name, err)
			}
			rows, err := parse(raw)
			if err != nil {
				return nil, fmt.Errorf("who: %s: %w", name, err)
			}
			out[string(ind)+"/"+string(sex)] = rows
		}
	}
	return out, nil
}

func parse(raw []byte) ([]LMS, error) {
	sc := bufio.NewScanner(bytes.NewReader(raw))
	rows := make([]LMS, 0, MaxDay+1)
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			first = false
			if line != "day,l,m,s" {
				return nil, fmt.Errorf("unexpected header %q", line)
			}
			continue
		}
		f := strings.Split(line, ",")
		if len(f) != 4 {
			return nil, fmt.Errorf("bad row %q", line)
		}
		day, err := strconv.Atoi(f[0])
		if err != nil || day != len(rows) {
			return nil, fmt.Errorf("bad day in %q", line)
		}
		var v [3]float64
		for i := range 3 {
			if v[i], err = strconv.ParseFloat(f[i+1], 64); err != nil {
				return nil, fmt.Errorf("bad number in %q: %w", line, err)
			}
		}
		rows = append(rows, LMS{Day: day, L: v[0], M: v[1], S: v[2]})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(rows) != MaxDay+1 {
		return nil, fmt.Errorf("%d rows, want %d", len(rows), MaxDay+1)
	}
	return rows, nil
}

// Table is the daily LMS table of an indicator and sex (index = age in days). Unknown pairs return nil.
func Table(ind Indicator, sex Sex) []LMS {
	t, err := tables()
	if err != nil {
		panic(err) // embedded data is checked by the package tests
	}
	return t[string(ind)+"/"+string(sex)]
}

// At is the LMS at age day (0–MaxDay); ok false outside the standard or for an unknown pair.
func At(ind Indicator, sex Sex, day int) (LMS, bool) {
	t := Table(ind, sex)
	if day < 0 || day >= len(t) {
		return LMS{}, false
	}
	return t[day], true
}
