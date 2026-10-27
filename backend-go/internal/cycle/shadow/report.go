package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/ritme/backend-go/internal/platform/cache"
)

// Report prints the 7-day class counts and the newest limit samples (limit ≤ 0 = all stored, at most MaxSamples).
func Report(ctx context.Context, rdb *cache.Client, w io.Writer, limit int) error {
	counts, err := rdb.Redis().HGetAll(ctx, rdb.Key(CountsKey)).Result()
	if err != nil {
		return fmt.Errorf("shadow: read counts: %w", err)
	}
	compared, _ := strconv.Atoi(counts["compared"])
	_, _ = fmt.Fprintf(w, "engine shadow (CYCLE_ENGINE=shadow), last 7 days: %d comparisons\n", compared)
	for _, c := range Classes() {
		n, _ := strconv.Atoi(counts[string(c)])
		pct := 0.0
		if compared > 0 {
			pct = 100 * float64(n) / float64(compared)
		}
		_, _ = fmt.Fprintf(w, "  %-18s %7d  %5.1f%%\n", c, n, pct)
	}

	stop := int64(MaxSamples - 1)
	if limit > 0 && limit < MaxSamples {
		stop = int64(limit - 1)
	}
	raw, err := rdb.Redis().LRange(ctx, rdb.Key(SamplesKey), 0, stop).Result()
	if err != nil {
		return fmt.Errorf("shadow: read samples: %w", err)
	}
	_, _ = fmt.Fprintf(w, "samples (newest first, %d shown):\n", len(raw))
	for _, r := range raw {
		var s Sample
		if json.Unmarshal([]byte(r), &s) != nil {
			continue // a corrupt entry is skipped, never printed raw
		}
		line, _ := json.Marshal(s)
		_, _ = fmt.Fprintf(w, "  %s\n", line)
	}
	return nil
}
