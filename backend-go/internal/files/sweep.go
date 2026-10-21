package files

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/ritme/backend-go/internal/files/store"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// SweepGrace: the orphan sweep (every 6 h, SweepCron) leaves blobs younger than an hour alone (an upload writes its
// blob before its row commits).
const SweepGrace = time.Hour

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

// Background job (T-M9-11, docs/engine-v6/jobs.md): the orphan-blob sweep every 6 h on the low queue, also at
// start-up. It reads the local STORAGE_PATH, so it must run where that volume is mounted.
const (
	JobSweep  = "files:sweep"
	SweepCron = "20 */6 * * *"
)

// RegisterJobs puts the sweep job on q.
func (s *Service) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobSweep, Priority: queue.Low, MaxTries: 1, Timeout: 30 * time.Minute,
	}, s.SweepJob(now), SweepCron, queue.Unique(5*time.Hour), queue.RunAtStart())
}

// SweepJob is the files:sweep handler. Idempotent: a removed blob is not listed again. Errors are skipped (nothing
// about a file is logged), the next tick retries.
func (s *Service) SweepJob(now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		_, _ = s.Sweep(ctx, now(), SweepGrace)
		return nil
	}
}
