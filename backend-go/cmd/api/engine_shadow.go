package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ritme/backend-go/internal/cycle/shadow"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
)

// engineShadowReport is `api engine-shadow-report [-n N]` (T-M9-32): the CYCLE_ENGINE=shadow class counts of the last
// 7 days and the newest N samples (default 50, 0 = all ≤ 1 000) from Redis. Samples carry a hashed user id, the
// target's offset from today and only the fields each class is about (internal/cycle/shadow). Exit 0/1 (2 = usage).
func engineShadowReport(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("engine-shadow-report", flag.ContinueOnError)
	n := fs.Int("n", 50, "samples to print (0 = all)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "engine-shadow-report:", err)
		return 1
	}
	rdb := cache.New(cfg.Redis)
	defer func() { _ = rdb.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := shadow.Report(ctx, rdb, out, *n); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "engine-shadow-report:", err)
		return 1
	}
	return 0
}
