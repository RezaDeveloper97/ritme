package safety

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/safety/store"
)

// RunsShown is how many persisted runs the admin screen lists.
const RunsShown = 30

// RunSummary is one persisted run.
type RunSummary struct {
	RunID     string
	StartedAt time.Time
	Scenarios int
	Failed    int
}

func asTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case []byte:
		p, _ := time.ParseInLocation(time.DateTime, string(t), civildate.Tehran)
		return p
	case string:
		p, _ := time.ParseInLocation(time.DateTime, t, civildate.Tehran)
		return p
	}
	return time.Time{}
}

// Runs are the latest limit persisted runs, newest first.
func (s *Service) Runs(ctx context.Context, limit int) ([]RunSummary, error) {
	rows, err := s.q.ListRecentSafetyRunIDs(ctx, int32(min(max(limit, 1), 1000))) //nolint:gosec // bounded
	if err != nil {
		return nil, fmt.Errorf("safety: runs: %w", err)
	}
	out := make([]RunSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, RunSummary{RunID: r.RunID, StartedAt: asTime(r.StartedAt), Scenarios: int(r.Scenarios), Failed: int(r.Failed)})
	}
	return out, nil
}

// RunResults are the scenario rows of one run (empty for an unknown id).
func (s *Service) RunResults(ctx context.Context, runID string) ([]store.SafetyScenarioRun, error) {
	rows, err := s.q.ListSafetyScenarioRunsByRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("safety: run: %w", err)
	}
	return rows, nil
}

// LatestRun is the newest persisted run in the dashboard's Run shape (nil when none ran yet).
func (s *Service) LatestRun(ctx context.Context) (*Run, error) {
	runs, err := s.Runs(ctx, 1)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	rows, err := s.RunResults(ctx, runs[0].RunID)
	if err != nil {
		return nil, err
	}
	r := &Run{ID: runs[0].RunID, RanAt: runs[0].StartedAt, Total: len(rows)}
	for _, row := range rows {
		if row.Passed {
			r.Passed++
		} else {
			r.Failed = append(r.Failed, row.Scenario)
		}
	}
	return r, nil
}

// KindOf is the kind of a persisted scenario name.
func KindOf(scenario string) string {
	if strings.HasPrefix(scenario, "config.") {
		return KindConfig
	}
	return KindPath
}

var runIDPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// RunsHandlers serve /safety/runs (capability «clinical» by path).
type RunsHandlers struct{ svc *Service }

// NewRunsHandlers wires the handlers.
func NewRunsHandlers(sqlDB *sql.DB) *RunsHandlers { return &RunsHandlers{svc: New(sqlDB)} }

// Routes mounts the module.
func (h *RunsHandlers) Routes(route func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/safety/runs", kit.Admin(h.List))
	route(fiber.MethodGet, "/safety/runs/:run_id", kit.Admin(h.Show))
}

func at(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return httpadmin.Time(sql.NullTime{Time: t, Valid: true})
}

// List is GET /safety/runs: {runs[{run_id, started_at, scenarios, passed, failed}], limit}.
func (h *RunsHandlers) List(c fiber.Ctx) error {
	runs, err := h.svc.Runs(c.Context(), RunsShown)
	if err != nil {
		return err
	}
	list := make([]any, 0, len(runs))
	for _, r := range runs {
		list = append(list, jsonx.Obj("run_id", r.RunID, "started_at", at(r.StartedAt), "scenarios", r.Scenarios,
			"passed", r.Scenarios-r.Failed, "failed", r.Failed))
	}
	return httpadmin.OK(c, jsonx.Obj("runs", list, "limit", RunsShown))
}

// Show is GET /safety/runs/:run_id: {run_id, started_at, results[{scenario, kind, passed, duration_ms, detail}]}
// (failures first); 404 for an unknown run.
func (h *RunsHandlers) Show(c fiber.Ctx) error {
	id := c.Params("run_id")
	if !runIDPattern.MatchString(id) {
		return httpadmin.NotFound("safety run")
	}
	rows, err := h.svc.RunResults(c.Context(), id)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return httpadmin.NotFound("safety run")
	}
	results := make([]any, 0, len(rows))
	var started time.Time
	for pass := range 2 { // failures first, then the passes, each in scenario order
		for _, r := range rows {
			if r.Passed != (pass == 1) {
				continue
			}
			if r.CreatedAt.Valid && (started.IsZero() || r.CreatedAt.Time.Before(started)) {
				started = r.CreatedAt.Time
			}
			var detail any
			if r.Detail.Valid {
				_ = json.Unmarshal(r.Detail.V, &detail)
			}
			results = append(results, jsonx.Obj("scenario", r.Scenario, "kind", KindOf(r.Scenario), "passed", r.Passed,
				"duration_ms", r.DurationMs, "detail", detail))
		}
	}
	return httpadmin.OK(c, jsonx.Obj("run_id", id, "started_at", at(started), "results", results))
}
