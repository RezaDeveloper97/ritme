// Command metricsdiff summarises the server side of a load run from two /metrics scrapes (T-M9-14):
// requests, error share, server-side p50/p95/p99 and queries per request per route.
//
//	metricsdiff scrape -url http://127.0.0.1:9090/metrics -out before.prom
//	metricsdiff diff   -before before.prom -after after.prom
//
// /metrics is the loopback-only listener of the API (README §6); the tool refuses non-loopback URLs.
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: metricsdiff scrape|diff ...")
	}
	args := parseFlags(os.Args[2:])
	switch os.Args[1] {
	case "scrape":
		u, out := args["url"], args["out"]
		if u == "" || out == "" {
			fail("scrape needs -url and -out")
		}
		if err := scrape(u, out); err != nil {
			fail(err.Error())
		}
	case "diff":
		b, a := args["before"], args["after"]
		if b == "" || a == "" {
			fail("diff needs -before and -after")
		}
		if err := diffFiles(b, a, os.Stdout); err != nil {
			fail(err.Error())
		}
	default:
		fail("unknown command " + os.Args[1])
	}
}

func parseFlags(a []string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(a); i += 2 {
		if len(a[i]) > 1 && a[i][0] == '-' {
			m[a[i][1:]] = a[i+1]
		}
	}
	return m
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "metricsdiff:", msg)
	os.Exit(1)
}

func scrape(raw, out string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if h := u.Hostname(); h != "127.0.0.1" && h != "localhost" && h != "::1" {
		return fmt.Errorf("refusing to scrape %q: loopback only", h)
	}
	c := http.Client{Timeout: 10 * time.Second}
	resp, err := c.Get(u.String()) //nolint:gosec,noctx // G107: loopback checked above; a one-shot CLI
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scrape: HTTP %d", resp.StatusCode)
	}
	return os.WriteFile(out, b, 0o600)
}
