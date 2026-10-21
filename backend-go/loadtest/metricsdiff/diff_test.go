package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const before = `http_request_duration_seconds_bucket{route="/api/v1/cycle/today",method="GET",status="200",le="0.01"} 0
http_request_duration_seconds_bucket{route="/api/v1/cycle/today",method="GET",status="200",le="+Inf"} 0
http_request_duration_seconds_count{route="/api/v1/cycle/today",method="GET",status="200"} 0
db_queries_per_request_sum{route="/api/v1/cycle/today"} 0
db_queries_per_request_count{route="/api/v1/cycle/today"} 0
`

const after = `http_request_duration_seconds_bucket{route="/api/v1/cycle/today",method="GET",status="200",le="0.01"} 90
http_request_duration_seconds_bucket{route="/api/v1/cycle/today",method="GET",status="200",le="0.025"} 100
http_request_duration_seconds_bucket{route="/api/v1/cycle/today",method="GET",status="200",le="+Inf"} 100
http_request_duration_seconds_count{route="/api/v1/cycle/today",method="GET",status="200"} 98
http_request_duration_seconds_count{route="/api/v1/cycle/today",method="GET",status="500"} 2
db_queries_per_request_sum{route="/api/v1/cycle/today"} 120
db_queries_per_request_count{route="/api/v1/cycle/today"} 100
`

func TestRender(t *testing.T) {
	b, err := parseScrape(strings.NewReader(before))
	require.NoError(t, err)
	a, err := parseScrape(strings.NewReader(after))
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, render(b, a, &out))
	row := out.String()
	assert.Contains(t, row, "`/api/v1/cycle/today` | 100 | 2 |")
	assert.Contains(t, row, "| 1.20 |") // 120 queries / 100 requests
	// p50 = 50th of 100 inside the first bucket (0, 10 ms] → 5.6 ms
	assert.Contains(t, row, "| 5.6 |")
}

func TestScrapeRefusesRemote(t *testing.T) {
	assert.Error(t, scrape("http://89.251.8.115:9090/metrics", t.TempDir()+"/x"))
}
