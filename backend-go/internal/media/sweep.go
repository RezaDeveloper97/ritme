package media

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ritme/backend-go/internal/media/store"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// StuckProcessingAfter is how long a row may stay in `processing` (a finalize that crashed mid-scan) before the sweep
// fails and removes it (B-N8-02b L4).
const StuckProcessingAfter = time.Hour

// strayMinAge keeps a file that was just created (its row is inserted right after) out of the stray sweep.
const strayMinAge = time.Hour

// Sweep removes expired unfinished uploads, rows stuck in processing, media of deleted lessons, directories of deleted instructors (account
// deletion cascades the rows) and files no row points at. It returns the number of rows / files removed.
func (s *Service) Sweep(ctx context.Context, now time.Time) (int, error) {
	removed := 0
	var errs []error
	expired, err := s.q.ListExpiredUploads(ctx, nt(now))
	if err != nil {
		return 0, fmt.Errorf("media: expired: %w", err)
	}
	for _, m := range expired {
		if err := s.remove(ctx, m); err != nil {
			errs = append(errs, err)
			continue
		}
		removed++
		errs = append(errs, s.restoreLesson(ctx, m, now))
	}
	stuck, err := s.q.ListStuckProcessing(ctx, nt(now.Add(-StuckProcessingAfter)))
	if err != nil {
		return removed, fmt.Errorf("media: stuck: %w", err)
	}
	for _, m := range stuck {
		unlock, ok := s.lock(m.ID)
		if !ok {
			continue // a finalize is running right now
		}
		_, err := s.q.SetMediaStatus(ctx, store.SetMediaStatusParams{Status: StatusFailed, Now: nt(now), ID: m.ID, FromStatus: StatusProcessing})
		if err == nil {
			err = s.remove(ctx, m)
		}
		unlock()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		removed++
		errs = append(errs, s.restoreLesson(ctx, m, now))
	}
	orphans, err := s.q.ListOrphanMedia(ctx)
	if err != nil {
		return removed, fmt.Errorf("media: orphans: %w", err)
	}
	for _, m := range orphans {
		if err := s.remove(ctx, m); err != nil {
			errs = append(errs, err)
			continue
		}
		removed++
	}
	ids, err := s.disk.instructorDirs()
	if err != nil || len(ids) == 0 {
		return removed, errors.Join(append(errs, err)...)
	}
	existing, err := s.q.ExistingInstructors(ctx, ids)
	if err != nil {
		return removed, fmt.Errorf("media: instructors: %w", err)
	}
	live := make(map[uint64]bool, len(existing))
	for _, id := range existing {
		live[id] = true
	}
	for _, id := range ids {
		if !live[id] {
			if err := s.disk.removeInstructor(id); err != nil {
				errs = append(errs, err)
			} else {
				removed++
			}
			continue
		}
		paths, err := s.q.ListInstructorMediaPaths(ctx, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("media: paths: %w", err))
			continue
		}
		keep := make(map[string]bool, len(paths))
		for _, p := range paths {
			keep[p] = true
		}
		stray, err := s.disk.strayFiles(ctx, id, keep, strayMinAge, now)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, rel := range stray {
			if err := s.disk.Remove(id, rel); err != nil {
				errs = append(errs, err)
			} else {
				removed++
			}
		}
	}
	return removed, errors.Join(errs...)
}

// Background job (T-M9-11, docs/engine-v6/jobs.md): the media sweep hourly on the low queue, also at start-up. It
// reads the local media root, so it must run where that volume is mounted.
const (
	JobSweep  = "media:sweep"
	SweepCron = "25 * * * *"
)

// RegisterJobs puts the sweep job on q.
func (s *Service) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobSweep, Priority: queue.Low, MaxTries: 1, Timeout: 30 * time.Minute,
	}, s.SweepJob(now), SweepCron, queue.Unique(55*time.Minute), queue.RunAtStart())
}

// SweepJob is the media:sweep handler. Idempotent: removed rows and files are not listed again. A failure is logged
// with the count removed so far; the next tick retries (no in-run retry, like the old loop).
func (s *Service) SweepJob(now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		if n, err := s.Sweep(ctx, now()); err != nil {
			s.logger.ErrorContext(ctx, "media: sweep failed", slog.Int("removed", n), slog.String("error", err.Error()))
		}
		return nil
	}
}
