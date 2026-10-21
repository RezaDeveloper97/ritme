package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func gate(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut)
	return code, out.String() + errOut.String()
}

// An allocs/op regression fails the gate, whatever the time did (and across GOMAXPROCS suffixes).
func TestGate_AllocsRegressionFails(t *testing.T) {
	code, out := gate(t, "-baseline", "testdata/baseline.txt", "-current", "testdata/allocs_regression.txt")
	assert.Equal(t, 1, code, out)
	assert.Contains(t, out, "::error title=allocs/op regression::github.com/ritme/backend-go/internal/cycle/resolver.BenchmarkResolve/H=24: 7 → 8 allocs/op")
	assert.NotContains(t, out, "BenchmarkCalculate/H=24: 18", "unchanged allocs are not reported")
	assert.NotContains(t, out, "sec/op regression")
}

// A time-only regression (> 20 %) passes with a warning annotation.
func TestGate_TimeOnlyRegressionWarns(t *testing.T) {
	code, out := gate(t, "-baseline", "testdata/baseline.txt", "-current", "testdata/time_regression.txt")
	assert.Equal(t, 0, code, out)
	assert.Contains(t, out, "::warning title=sec/op regression::github.com/ritme/backend-go/internal/cycle/resolver.BenchmarkCalculate/H=24: 17.00µs → 25.00µs (+47%, threshold 20%)")
	assert.NotContains(t, out, "BenchmarkResolve/H=24: ", "within the threshold")
	assert.NotContains(t, out, "::error")
	assert.Contains(t, out, "0 allocs/op regressions, 1 sec/op warnings")
}

// The same file against itself is clean; a raised threshold silences the time warning.
func TestGate_NoChangeAndThreshold(t *testing.T) {
	code, out := gate(t, "-baseline", "testdata/baseline.txt", "-current", "testdata/time_regression.txt", "-time-threshold", "0.6")
	assert.Equal(t, 0, code, out)
	assert.NotContains(t, out, "::warning")

	code, out = gate(t, "-baseline", "testdata/time_regression.txt", "-current", "testdata/baseline.txt")
	assert.Equal(t, 0, code, out)
	assert.NotContains(t, out, "::")
}

// -time-base takes the time reference from a same-machine run; allocs still come from -baseline.
func TestGate_TimeBaseOverridesTheTimeReference(t *testing.T) {
	code, out := gate(t, "-baseline", "testdata/baseline.txt", "-current", "testdata/time_regression.txt", "-time-base", "testdata/time_regression.txt")
	assert.Equal(t, 0, code, out)
	assert.NotContains(t, out, "sec/op regression")
}

func TestGate_SkipTime(t *testing.T) {
	code, out := gate(t, "-baseline", "testdata/baseline.txt", "-current", "testdata/time_regression.txt", "-skip-time")
	assert.Equal(t, 0, code, out)
	assert.NotContains(t, out, "sec/op regression")
	code, _ = gate(t, "-baseline", "testdata/baseline.txt", "-current", "testdata/allocs_regression.txt", "-skip-time")
	assert.Equal(t, 1, code, "allocs still gate")
}

func TestGate_NewAndMissingBenchmarks(t *testing.T) {
	f, err := parse(strings.NewReader("pkg: p\nBenchmarkNew-8  10  100 ns/op  1 allocs/op\n"))
	assert.NoError(t, err)
	base, err := parseFile("testdata/baseline.txt")
	assert.NoError(t, err)
	var out bytes.Buffer
	assert.Equal(t, 0, compare(base, base, f, 0.2, 0.001, &out))
	assert.Contains(t, out.String(), "::notice title=new benchmark::p.BenchmarkNew")
	assert.Contains(t, out.String(), "::warning title=missing benchmark::github.com/ritme/backend-go/internal/cycle/resolver.BenchmarkResolve/H=24")
}

func TestGate_BadInput(t *testing.T) {
	code, _ := gate(t, "-baseline", "testdata/nope.txt", "-current", "testdata/baseline.txt")
	assert.Equal(t, 2, code)
	code, _ = gate(t, "-baseline", "testdata/baseline.txt", "-current", "testdata/baseline.txt")
	assert.Equal(t, 2, code, "comparing a file with itself is a mistake")
	_, err := parse(strings.NewReader("BenchmarkX-8 10 100 ns/op 3\n"))
	assert.Error(t, err)
}

// Large counts may wobble by a few allocations between runs (amortised growth); 0.1 % absorbs that.
func TestGate_AllocSlackOnLargeCounts(t *testing.T) {
	base, err := parse(strings.NewReader("pkg: p\nBenchmarkBig-8  10  100 ns/op  26950 allocs/op\n"))
	assert.NoError(t, err)
	wobble, err := parse(strings.NewReader("pkg: p\nBenchmarkBig-8  10  100 ns/op  26956 allocs/op\n"))
	assert.NoError(t, err)
	grown, err := parse(strings.NewReader("pkg: p\nBenchmarkBig-8  10  100 ns/op  27100 allocs/op\n"))
	assert.NoError(t, err)
	var out bytes.Buffer
	assert.Equal(t, 0, compare(base, nil, wobble, 0.2, 0.001, &out), out.String())
	assert.Equal(t, 1, compare(base, nil, grown, 0.2, 0.001, &out), out.String())
}

func TestMedian(t *testing.T) {
	m, ok := median([]float64{3, 1, 2})
	assert.True(t, ok)
	assert.InDelta(t, 2, m, 0)
	m, _ = median([]float64{4, 1, 3, 2})
	assert.InDelta(t, 2.5, m, 0)
	_, ok = median(nil)
	assert.False(t, ok)
}
