package challenges

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chs "github.com/ritme/backend-go/internal/challenges"
	chstore "github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func input(t *testing.T, js string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(js))
	require.NoError(t, err)
	return v.(phpval.Map)
}

func check(t *testing.T, js string) map[string]string {
	t.Helper()
	errs := map[string]string{}
	require.NoError(t, Check(nil)(input(t, js), func(f, m string) {
		if _, ok := errs[f]; !ok {
			errs[f] = m
		}
	}))
	return errs
}

func TestCheck(t *testing.T) {
	assert.Empty(t, check(t, `{"eligibility":{"all_of":[{"fact":"life_mode","op":"in","value":["cycle","ttc"]}],"any_of":[],"exclude":[]},
		"contraindications":{"any_of":[[{"fact":"mode","op":"eq","value":"pregnancy"},{"fact":"postpartum_week","op":"lte","value":6}]]}}`))
	assert.Empty(t, check(t, `{"eligibility":[],"contraindications":null}`), "empty = no rule")

	errs := check(t, `{"eligibility":{"all_of":[{"fact":"no_such_fact","op":"eq","value":1}]},
		"contraindications":{"any_of":[[{"fact":"postpartum_week","op":"lte","value":"six"}]],"exclude":[{"fact":"mode","op":"zz","value":1}]}}`)
	assert.Contains(t, errs["eligibility.all_of.0"], "Unknown fact")
	assert.Contains(t, errs["contraindications.any_of.0.0"], "number")
	assert.Contains(t, errs["contraindications.exclude.0"], "valid operator")

	assert.Contains(t, check(t, `{"eligibility":{"if":[]}}`)["eligibility"], "unknown key")
	assert.Contains(t, check(t, `{"eligibility":{"all_of":"x"}}`)["eligibility"], "list of conditions")
	assert.Contains(t, check(t, `{"start_rule":"on_phase_start"}`)["start_phase"], "phase")
	assert.Empty(t, check(t, `{"start_rule":"on_phase_start","start_phase":"luteal"}`))
}

func TestStartRuleAndDropOffJSON(t *testing.T) {
	assert.Equal(t, chs.StartOnJoin, StartRule(input(t, `{"start_rule":"anytime"}`)))
	assert.Equal(t, chs.StartOnJoin, StartRule(input(t, `{}`)))
	assert.Equal(t, chs.StartMenstrual, StartRule(input(t, `{"start_rule":"menstrual"}`)), "legacy menstrual kept")
	out := DropOffJSON([]chs.DayRetention{{Day: 1, Reached: 2, Retained: 1, Dropped: 1, RetainedPct: 50, DroppedPct: 50}, {Day: 2, RetainedPct: -1, DroppedPct: -1}})
	b, err := out[0].MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"day":1,"reached":2,"retained":1,"dropped":1,"retained_rate":50,"drop_off_rate":50}`, string(b))
	b, _ = out[1].MarshalJSON()
	assert.JSONEq(t, `{"day":2,"reached":0,"retained":0,"dropped":0,"retained_rate":null,"drop_off_rate":null}`, string(b))
}

// Integration: Save writes the v6 shape (legacy lists converted), keeps stored contraindications on a legacy save.
func TestSave(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := sql.NullTime{Time: time.Date(2026, 10, 6, 9, 0, 0, 0, time.Local), Valid: true}
	res, err := db.Exec(`INSERT INTO challenges (slug, title, is_active, sort_order, created_at, updated_at) VALUES ('s', '{"fa":"x"}', 1, 1, NOW(), NOW())`)
	require.NoError(t, err)
	id64, _ := res.LastInsertId()
	id := uint64(id64)
	q := chstore.New(db)
	require.NoError(t, q.UpsertProgram(ctx, chstore.UpsertProgramParams{ChallengeID: id, DurationDays: 7, StartRule: "anytime", Now: now}))

	// legacy request: lists → v6 shape; new contraindications stored; anytime → on_join
	require.NoError(t, Save(ctx, q, id, input(t, `{"start_rule":"anytime","modes_include":["cycle"],"phases_exclude":["fertile"],
		"contraindications":{"all_of":[{"fact":"mode","op":"eq","value":"pregnancy"}]}}`), now))
	p, err := q.GetProgram(ctx, id)
	require.NoError(t, err)
	assert.JSONEq(t, `{"all_of":[{"fact":"life_mode","op":"in","value":["cycle"]}],"any_of":[],"exclude":[{"fact":"main_phase","op":"in","value":["fertile"]}]}`, string(p.Eligibility.V))
	assert.JSONEq(t, `{"all_of":[{"fact":"mode","op":"eq","value":"pregnancy"}],"any_of":[],"exclude":[]}`, string(p.Contraindications.V))
	assert.Equal(t, chs.StartOnJoin, p.StartRule)
	assert.False(t, p.StartPhase.Valid)

	// an older editor's save (no contraindications key) keeps them; on_phase_start stores its phase
	require.NoError(t, Save(ctx, q, id, input(t, `{"start_rule":"on_phase_start","start_phase":"luteal","eligibility":{"all_of":[]}}`), now))
	p, err = q.GetProgram(ctx, id)
	require.NoError(t, err)
	assert.False(t, p.Eligibility.Valid, "an empty rule is NULL")
	assert.True(t, p.Contraindications.Valid, "kept")
	assert.Equal(t, "luteal", p.StartPhase.String)

	// the stored rules evaluate as written
	got := chs.ParseRule(p.Contraindications)
	assert.Equal(t, chs.ReasonContraindicated, chs.Decide(chs.ParseRule(p.Eligibility), got, map[string]any{"mode": "pregnancy", "life_mode": "pregnancy"}))

	// explicit null clears the contraindications
	require.NoError(t, Save(ctx, q, id, input(t, `{"contraindications":null}`), now))
	p, _ = q.GetProgram(ctx, id)
	assert.False(t, p.Contraindications.Valid)
}
