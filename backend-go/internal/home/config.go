package home

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The admin-edited tables /home reads (T-M9-13b): each one is a process-wide configcache snapshot of its active rows,
// reloaded when an admin write publishes the table (httpadmin.Publishes, here or on ritme-go:config-changed) or after
// the 60 s TTL. The phase / mode filters run in Go over the snapshot. Snapshot rows are shared: never modify them.

// Tables are the tables the /home snapshots read.
var (
	TaskTemplatesTables = []string{"task_templates"}
	AffirmationsTables  = []string{"affirmations"}
	ChallengesTables    = []string{"challenges", "challenge_programs"} // v2 programs are not daily picks
	ArticlesTables      = []string{"articles"}
)

var (
	taskTemplatesSnap = configcache.New[[]store.ListActiveTaskTemplatesRow]("home:task_templates", TaskTemplatesTables, nil)
	affirmationsSnap  = configcache.New[[]store.ListActiveAffirmationsRow]("home:affirmations", AffirmationsTables, nil)
	challengesSnap    = configcache.New[[]store.ListActiveChallengesRow]("home:challenges", ChallengesTables, nil)
	articlesSnap      = configcache.New[[]store.ListPublishedHomeArticlesRow]("home:articles", ArticlesTables, nil)
)

// homeArticlesLimit is ArticlesSection's limit(6).
const homeArticlesLimit = 6

// forPhase is the scopeForPhase of TaskTemplate / Affirmation: cycle_phase IS NULL, or (when phase is truthy)
// cycle_phase = phase under the column's utf8mb4_unicode_ci collation (case-insensitive, trailing spaces ignored).
func forPhase(rowPhase string, valid bool, phase string, truthy bool) bool {
	if !valid {
		return true
	}
	return truthy && strings.EqualFold(strings.TrimRight(rowPhase, " "), strings.TrimRight(phase, " "))
}

// activeTaskTemplates is TaskTemplate::active()->forPhase($phase)->get() from the snapshot.
func activeTaskTemplates(ctx context.Context, q store.Querier, phase string) ([]store.ListActiveTaskTemplatesRow, error) {
	all, err := taskTemplatesSnap.GetWith(ctx, q.ListActiveTaskTemplates)
	if err != nil {
		return nil, fmt.Errorf("home: task templates: %w", err)
	}
	truthy := phpval.Truthy(phase)
	out := make([]store.ListActiveTaskTemplatesRow, 0, len(all))
	for _, r := range all {
		if forPhase(r.CyclePhase.String, r.CyclePhase.Valid, phase, truthy) {
			out = append(out, r)
		}
	}
	return out, nil
}

// activeAffirmations is Affirmation::active()->forPhase($phase)->orderBy('sort_order')->orderBy('id') from the snapshot.
func activeAffirmations(ctx context.Context, q store.Querier, phase string) ([]store.ListActiveAffirmationsRow, error) {
	all, err := affirmationsSnap.GetWith(ctx, q.ListActiveAffirmations)
	if err != nil {
		return nil, fmt.Errorf("home: affirmations: %w", err)
	}
	truthy := phpval.Truthy(phase)
	out := make([]store.ListActiveAffirmationsRow, 0, len(all))
	for _, r := range all {
		if forPhase(r.CyclePhase.String, r.CyclePhase.Valid, phase, truthy) {
			out = append(out, r)
		}
	}
	return out, nil
}

// activeChallenges is Challenge::active() without v2 programs, in sort_order, id, from the snapshot (shared rows).
func activeChallenges(ctx context.Context, q store.Querier) ([]store.ListActiveChallengesRow, error) {
	all, err := challengesSnap.GetWith(ctx, q.ListActiveChallenges)
	if err != nil {
		return nil, fmt.Errorf("home: challenges: %w", err)
	}
	return all, nil
}

// homeArticles is Article::published()->forPhase($candidates)->orderByRaw('cycle_phases is null')->orderBy('sort_order')
// ->limit(6) from the snapshot: untagged articles, or tagged with any candidate (JSON_OVERLAPS); tagged ones first.
func homeArticles(ctx context.Context, q store.Querier, candidates []string) ([]store.ListPublishedHomeArticlesRow, error) {
	all, err := articlesSnap.GetWith(ctx, q.ListPublishedHomeArticles)
	if err != nil {
		return nil, fmt.Errorf("home: articles: %w", err)
	}
	var tagged, untagged []store.ListPublishedHomeArticlesRow // the snapshot is in sort_order, id already
	for _, a := range all {
		switch {
		case !a.CyclePhases.Valid:
			untagged = append(untagged, a)
		case len(candidates) > 0 && jsonOverlaps(a.CyclePhases.V, candidates):
			tagged = append(tagged, a)
		}
		if len(tagged) >= homeArticlesLimit {
			break
		}
	}
	tagged = append(tagged, untagged...)
	return tagged[:min(len(tagged), homeArticlesLimit)], nil
}

// jsonOverlaps is MariaDB JSON_OVERLAPS(doc, candidates) for a candidate list of strings: an array overlaps when one
// of its elements is a candidate, a scalar string when it is one; objects and other scalars never overlap an array of
// strings.
func jsonOverlaps(doc json.RawMessage, candidates []string) bool {
	var v any
	if json.Unmarshal(doc, &v) != nil {
		return false
	}
	switch x := v.(type) {
	case string:
		return slices.Contains(candidates, x)
	case []any:
		for _, e := range x {
			if s, ok := e.(string); ok && slices.Contains(candidates, s) {
				return true
			}
		}
	}
	return false
}
