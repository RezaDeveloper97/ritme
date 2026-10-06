package files

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/ritme/backend-go/internal/files/store"
)

// SweepEvery / SweepGrace: the orphan sweep runs every 6 h and leaves blobs younger than an hour alone (an upload
// writes its blob before its row commits).
const (
	SweepEvery = 6 * time.Hour
	SweepGrace = time.Hour
)

// Sweep removes generic blobs that have no `files` row (a failed removal after a delete, an account deleted by a path
// that did not call RemoveUser) and older than grace. Lab sheets have their own sweep. Returns how many it removed.
func (s *Service) Sweep(ctx context.Context, now time.Time, grace time.Duration) (int, error) {
	removed := 0
	for _, p := range s.purposes {
		if p.blobOnly || p.dir == "" {
			continue
		}
		owners, err := Owners(s.vault.root, p)
		if err != nil {
			return removed, err
		}
		if p.Public {
			owners = append(owners, 0)
		}
		for _, owner := range owners {
			var paths []string
			if owner > 0 {
				paths, err = s.q.ListOwnerPaths(ctx, store.ListOwnerPathsParams{UserID: ownerArg(owner), Purpose: p.Name})
			} else {
				paths, err = s.q.ListPlatformPaths(ctx, p.Name)
			}
			if err != nil {
				return removed, err
			}
			known := make(map[string]bool, len(paths))
			for _, rel := range paths {
				known[rel] = true
			}
			dir := ownerDir(p, owner)
			entries, err := os.ReadDir(filepath.Join(s.vault.root, filepath.FromSlash(dir)))
			if err != nil {
				continue // no directory (platform without files) or unreadable: nothing to sweep
			}
			for _, e := range entries {
				rel := dir + "/" + e.Name()
				info, err := e.Info()
				if err != nil || known[rel] || !valid(p, owner, rel) || now.Sub(info.ModTime()) < grace {
					continue
				}
				if s.vault.Remove(p, owner, rel) == nil {
					removed++
				}
			}
		}
	}
	return removed, nil
}

// SweepLoop runs Sweep every SweepEvery until ctx ends (errors are skipped; nothing about a file is logged).
func (s *Service) SweepLoop(ctx context.Context) {
	t := time.NewTicker(SweepEvery)
	defer t.Stop()
	for {
		_, _ = s.Sweep(ctx, time.Now(), SweepGrace)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
