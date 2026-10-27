package todo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/todo"
)

// engineStates is the production state source (state.Source over a fresh provider) on the test database.
func engineStates(t *testing.T, e *env) state.Source {
	t.Helper()
	q := cyclestore.New(e.db)
	p, err := state.New(q, nil, clock.Real{}, state.DefaultOptions())
	require.NoError(t, err)
	return state.Source{P: p, Revs: q}
}

// T-M9-37 (D-136): under the v6 engine the offer gates on the resolved mode and the engine limits, not on the raw
// stored life mode — a pregnant user without a stored mode is offered nothing.
func TestTodo_EngineSuggestion_PregnantWithoutStoredMode(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	svc := todo.NewService(e.db).WithEngine(engineStates(t, e))
	today := civildate.InTehran(fixed)

	cycleUser, _ := e.user(t, "09120000901")
	pregnant, _ := e.user(t, "09120000902")
	for _, id := range []uint64{cycleUser, pregnant} { // next predicted 2026-10-09, 3 days away
		e.period(t, id, "2026-08-12")
		e.period(t, id, "2026-09-10")
	}
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, pregnant)
	require.NoError(t, err)
	require.NoError(t, revision.Bump(ctx, e.db, pregnant))

	sg, ok, err := svc.Suggestion(ctx, cycleUser, today)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "2026-10-09", sg.PeriodStart.String())
	assert.Equal(t, 3, sg.Days)

	_, ok, err = svc.Suggestion(ctx, pregnant, today)
	require.NoError(t, err)
	assert.False(t, ok, "pregnancy (resolved mode, engine suspended) → no period offer")

	// The v1 path keeps its raw-stored-mode gate ("" allowed): the same user is still offered — unchanged under v1.
	_, ok, err = todo.NewService(e.db).Suggestion(ctx, pregnant, today)
	require.NoError(t, err)
	assert.True(t, ok)
}

// A late period (predicted start passed) has no offer under v6.
func TestTodo_EngineSuggestion_LateHasNone(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	svc := todo.NewService(e.db).WithEngine(engineStates(t, e))
	id, _ := e.user(t, "09120000903")
	e.period(t, id, "2026-07-15")
	e.period(t, id, "2026-08-12") // next 2026-09-09: 27 days late on 2026-10-06
	_, ok, err := svc.Suggestion(ctx, id, civildate.InTehran(fixed))
	require.NoError(t, err)
	assert.False(t, ok)
}
