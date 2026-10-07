package media

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// SweepInterval is how often SweepLoop runs.
const SweepInterval = time.Hour

// strayMinAge keeps a file that was just created (its row is inserted right after) out of the stray sweep.
const strayMinAge = time.Hour

// Sweep removes expired unfinished uploads, media of deleted lessons, directories of deleted instructors (account
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

// SweepLoop runs Sweep every SweepInterval until ctx ends.
func (s *Service) SweepLoop(ctx context.Context, now func() time.Time) {
	t := time.NewTicker(SweepInterval)
	defer t.Stop()
	for {
		if n, err := s.Sweep(ctx, now()); err != nil {
			s.logger.ErrorContext(ctx, "media: sweep failed", slog.Int("removed", n), slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
