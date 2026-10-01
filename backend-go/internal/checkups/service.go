package checkups

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/checkups/store"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Plan is one user's evaluated screening plan plus the catalog rows behind it.
type Plan struct {
	Today  civildate.Date
	Input  engine.Input
	Result engine.Result
	// LifeMode is the user's effective life mode: shared rows are limited to its audience.
	LifeMode enums.LifeMode
	// Rows are the active catalog rows the user sees (shared for her life mode + her custom), by type id.
	Rows map[uint64]store.CheckupType
	// Types are the engine types, by id.
	Types map[uint64]engine.Type
	// items indexes Result.Items by type id.
	items map[uint64]int
	now   civildate.Nower
}

// LoadPlan evaluates the user's plan in three queries: the profile + pregnancy flag + stored life
// mode, the cycle history (the cycle engine's own inputs, read-only reuse of internal/cycle) and
// the catalog for her life mode (CB-MENO-01b audiences) joined with the user's settings and latest
// record per type.
func LoadPlan(ctx context.Context, db store.DBTX, userID uint64, now civildate.Nower) (*Plan, error) {
	q := store.New(db)
	uc, err := q.GetCheckupUserContext(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("checkups: load profile: %w", err)
	}
	histories, err := cyclestore.New(db).ListCycleHistoriesNewestFirst(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("checkups: load cycle history: %w", err)
	}
	mode := lifeModeOf(uc)
	rows, err := q.ListCheckupPlanRows(ctx, store.ListCheckupPlanRowsParams{UserID: int64(userID), LifeMode: string(mode)}) //nolint:gosec // ids fit int64
	if err != nil {
		return nil, fmt.Errorf("checkups: load plan: %w", err)
	}

	in := engine.Input{
		Types:    make([]engine.Type, 0, len(rows)),
		Records:  []engine.Record{},
		Settings: []engine.Setting{},
		Pregnant: uc.Pregnant != 0,
	}
	if uc.Birthday.Valid {
		in.Birthday = uc.Birthday.Date
	}
	var profile *cyclestore.UserProfile
	if uc.ProfileID.Valid {
		profile = &cyclestore.UserProfile{
			Birthday: uc.Birthday, LastPeriodStart: uc.LastPeriodStart,
			CycleDuration: uc.CycleDuration, PeriodDuration: uc.PeriodDuration, UserGoal: uc.UserGoal.String,
		}
	}
	if mode.TracksCycle() { // B-N2-11b: menopause has no cycle to place a window on — interval scheduling only
		in.Cycle = engine.CycleFromHistory(cycleservice.HistoriesFromRows(histories), cycleservice.ProfileFromRow(profile), now)
	}

	p := &Plan{
		Today:    civildate.Today(now),
		LifeMode: mode,
		now:      now,
		Rows:     make(map[uint64]store.CheckupType, len(rows)),
		Types:    make(map[uint64]engine.Type, len(rows)),
	}
	for _, r := range rows {
		t := TypeFromRow(r.CheckupType)
		in.Types = append(in.Types, t)
		p.Rows[t.ID] = r.CheckupType
		p.Types[t.ID] = t
		if r.SettingEnabled.Valid {
			in.Settings = append(in.Settings, engine.Setting{TypeID: t.ID, Enabled: r.SettingEnabled.Bool, Remind: r.SettingRemind.Bool})
		}
		if r.RecordID.Valid && r.RecordDoneOn.Valid {
			rec := engine.Record{ID: uint64(r.RecordID.Int64), TypeID: t.ID, DoneOn: r.RecordDoneOn.Date} //nolint:gosec // ids are positive
			if r.RecordNextDueOn.Valid {
				rec.NextDueOn = r.RecordNextDueOn.Date
			}
			in.Records = append(in.Records, rec)
		}
	}
	p.Input = in
	p.Result = engine.Evaluate(in, now)
	p.items = make(map[uint64]int, len(p.Result.Items))
	for i, it := range p.Result.Items {
		p.items[it.TypeID] = i
	}
	return p, nil
}

// lifeModeOf is the effective life mode of a user context row (enums.ResolveLifeMode: an active
// pregnancy wins, then a stored postpartum / menopause / teen mode, else ttc / cycle by user_goal).
func lifeModeOf(uc store.GetCheckupUserContextRow) enums.LifeMode {
	return enums.ResolveLifeMode(uc.LifeMode.String, uc.Pregnant != 0, uc.UserGoal.String)
}

// UserLifeMode is the user's effective life mode, the audience the shared catalog is filtered by
// (checkup_types.audiences NULL = everyone, else the list of modes that see the row).
func UserLifeMode(ctx context.Context, q store.Querier, userID uint64) (enums.LifeMode, error) {
	uc, err := q.GetCheckupUserContext(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("checkups: load life mode: %w", err)
	}
	return lifeModeOf(uc), nil
}

// Item is the evaluated item of a type the user sees. A type the engine left out of the plan
// (hidden while pregnant, past age_max) is evaluated on its own so its detail page still opens;
// ok=false when the type is not in the user's catalog at all.
func (p *Plan) Item(typeID uint64) (engine.Item, bool) {
	if i, ok := p.items[typeID]; ok {
		return p.Result.Items[i], true
	}
	t, ok := p.Types[typeID]
	if !ok {
		return engine.Item{}, false
	}
	t.HideInPregnancy, t.AgeMax = false, 0
	in := p.Input
	in.Types = []engine.Type{t}
	res := engine.Evaluate(in, p.now)
	if len(res.Items) == 0 {
		return engine.Item{}, false
	}
	return res.Items[0], true
}

// Remind is the type's reminder switch (on without a settings row).
func (p *Plan) Remind(typeID uint64) bool {
	for _, s := range p.Input.Settings {
		if s.TypeID == typeID {
			return s.Remind
		}
	}
	return true
}

// Enabled is the type's plan switch (on without a settings row).
func (p *Plan) Enabled(typeID uint64) bool {
	for _, s := range p.Input.Settings {
		if s.TypeID == typeID {
			return s.Enabled
		}
	}
	return true
}
