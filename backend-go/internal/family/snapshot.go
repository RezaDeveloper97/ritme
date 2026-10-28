package family

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
)

// OwnerState is the owner's row of family_owner_state.
type OwnerState struct {
	Mode          string
	Epoch         uint32
	ReviewPending bool
}

// snapRow is the common shape of the snapshot statements (one row per (link, grant)).
type snapRow struct {
	linkID, ownerID, memberID uint64
	typ, role, status         string
	expiresAt                 sql.NullTime
	section, level            sql.NullString
	grantEpoch                sql.NullInt32
	stateMode                 sql.NullString
	stateEpoch                sql.NullInt32
	blocked                   bool
	gender                    sql.NullString
	// mode inputs (enums.ResolveLifeMode); haveInputs false for the locking statement
	haveInputs bool
	lifeMode   sql.NullString
	pregnant   bool
	goal       sql.NullString
}

func u64(v sql.NullInt64) uint64 {
	if !v.Valid || v.Int64 < 0 {
		return 0
	}
	return uint64(v.Int64)
}

func u32(v sql.NullInt32) (uint32, bool) {
	if !v.Valid || v.Int32 < 0 {
		return 0, false
	}
	return uint32(v.Int32), true
}

// resolveMode is the owner's effective mode from the snapshot inputs (enums.ResolveLifeMode: an active pregnancy
// wins, then the stored post-cycle modes, then the TTC goal, else cycle).
//
// A stored teen life mode wins over everything, an active pregnancy included (enums.ResolveLifeMode lets pregnancy win
// for the app's own screens): teen suspension of adult sharing must not be bypassed by a pregnancy flag (FAM-H1).
func resolveMode(lifeMode string, pregnant bool, goal string) string {
	if lifeMode == "teen" {
		return "teen"
	}
	return string(enums.ResolveLifeMode(lifeMode, pregnant, goal))
}

// build folds the rows of one (owner, member) pair into a Snapshot. Only the first link is used (an owner has at
// most one open link per member, enforced on accept).
func build(ownerID, actorID uint64, rows []snapRow) Snapshot {
	s := Snapshot{OwnerID: ownerID, ActorID: actorID}
	if len(rows) == 0 {
		return s
	}
	first := rows[0]
	s.HasLink = first.status == "active"
	s.LinkID = first.linkID
	s.Role = ParseRole(first.role, first.typ)
	if first.expiresAt.Valid {
		s.ExpiresAt = first.expiresAt.Time
	}
	s.OwnerBlocked = first.blocked
	s.OwnerMale = first.gender.Valid && first.gender.String == "male"
	if first.stateMode.Valid {
		s.OwnerMode = ModeOf(first.stateMode.String)
		s.OwnerEpoch, _ = u32(first.stateEpoch)
	}
	for _, r := range rows {
		if r.linkID != first.linkID || !r.section.Valid {
			continue
		}
		e, stamped := u32(r.grantEpoch)
		d, l, ok := grantRow(r.section.String, r.level.String)
		if !ok {
			continue // unknown code / level, symptoms, none: no grant (deny by default)
		}
		if !stamped && legacyEpochOne(r.section.String, r.level.String) {
			e, stamped = 1, true
		}
		s.Grants[d] = Grant{Level: l, Epoch: e, Stamped: stamped}
	}
	return s
}

// grantRow reads one stored grant: a v6 code and level, or a row the 00110 data migration has not converted yet
// (written by the v1 code or straight into a test database), read the way 00110 would convert it — cycle → cycle_phase
// view, pregnancy edit → view, meds / appointments edit → write; symptoms and none are no grant (F-03).
func grantRow(code, level string) (Domain, Level, bool) {
	if code == "cycle" {
		code = DomainCyclePhase.String()
	}
	d, ok := ParseDomain(code)
	if !ok {
		return DomainNone, LevelOff, false
	}
	l, ok := ParseLevel(level)
	if !ok && level == "edit" {
		l, ok = LevelWrite, true
	}
	if !ok || l == LevelOff {
		return DomainNone, LevelOff, false
	}
	if spec, _ := Lookup(d); l > spec.Max {
		l = spec.Max
	}
	return d, l, true
}

// legacyEpochOne reports whether an unstamped row is certainly an unconverted legacy grant that 00110 would stamp
// with epoch 1 (the epoch SyncMode starts an owner at): only a legacy-only code (cycle) or a legacy-only level on
// meds / appointments (edit). Everything else unstamped stays inactive (deny):
//   - pregnancy: may be a converted grant of an owner not pregnant at conversion time; a new pregnancy must never
//     inherit it (FAM-H3);
//   - meds / appointments at view: the code and level are the same in v1 and v6, so it cannot be told apart from a
//     v6 row written without an epoch. That is safe to deny: 00110 stamps every meds / appointments row it keeps
//     (COALESCE(state epoch, 1)), and every v6 writer (Grants.Apply) stamps, so a NULL-epoch row after 00110 is a
//     stray — failing closed costs at most a re-approve (L3).
func legacyEpochOne(code, level string) bool {
	switch code {
	case "cycle":
		return true
	case "meds", "appointments":
		return level == "edit"
	}
	return false
}

// Loader loads snapshots for one request: one statement per (owner, member) pair, one for every owner of a member,
// one for the actor's guardianships of a set of children. Results are memoised on the loader (Request keeps one per
// fiber request). The owner's mode is synced lazily (SyncMode) whenever the stored state is missing or stale, so a
// missed mode-change hook can only deny, never leak.
type Loader struct {
	db      store.DBTX
	pairs   map[[2]uint64]Snapshot
	members map[uint64][]Snapshot
	now     func() time.Time
}

// NewLoader returns a fresh loader on db; now stamps owner-state writes.
func NewLoader(db store.DBTX, now func() time.Time) *Loader {
	if now == nil {
		now = time.Now
	}
	return &Loader{db: db, pairs: map[[2]uint64]Snapshot{}, members: map[uint64][]Snapshot{}, now: now}
}

type loaderKey struct{}

const localsLoader = "family.loader"

// Request returns the loader memoised on the request (a fiber.Ctx) or a new one for any other context. A loader is
// never shared across requests: its snapshots are as fresh as the request.
func Request(ctx context.Context, db store.DBTX, now func() time.Time) *Loader {
	if c, ok := ctx.(fiber.Ctx); ok {
		if l, ok := c.Locals(localsLoader).(*Loader); ok && l.db == db {
			return l
		}
		l := NewLoader(db, now)
		c.Locals(localsLoader, l)
		return l
	}
	if l, ok := ctx.Value(loaderKey{}).(*Loader); ok && l.db == db {
		return l
	}
	return NewLoader(db, now)
}

// WithLoader returns ctx carrying a fresh loader (jobs, tests).
func WithLoader(ctx context.Context, db store.DBTX, now func() time.Time) context.Context {
	return context.WithValue(ctx, loaderKey{}, NewLoader(db, now))
}

// Pair is the snapshot of ownerID → actorID. The owner herself needs no statement.
func (l *Loader) Pair(ctx context.Context, ownerID, actorID uint64) (Snapshot, error) {
	if ownerID == 0 || actorID == 0 || ownerID == actorID {
		return Snapshot{OwnerID: ownerID, ActorID: actorID}, nil
	}
	key := [2]uint64{ownerID, actorID}
	if s, ok := l.pairs[key]; ok {
		return s, nil
	}
	if all, ok := l.members[actorID]; ok { // the member's full load already covers this pair
		for _, s := range all {
			if s.OwnerID == ownerID {
				return s, nil
			}
		}
		return Snapshot{OwnerID: ownerID, ActorID: actorID}, nil
	}
	rows, err := store.New(l.db).FamilySnapshotForPair(ctx, store.FamilySnapshotForPairParams{
		OwnerID: ownerID, MemberID: sql.NullInt64{Int64: int64(actorID), Valid: true}, //nolint:gosec // ids fit
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("family: snapshot: %w", err)
	}
	in := make([]snapRow, 0, len(rows))
	for _, r := range rows {
		in = append(in, snapRow{linkID: r.LinkID, ownerID: r.OwnerID, memberID: u64(r.MemberID), typ: r.Type,
			role: r.Role.String, status: r.Status, expiresAt: r.ExpiresAt, section: r.Section, level: r.Level,
			grantEpoch: r.GrantModeEpoch, stateMode: r.OwnerMode, stateEpoch: r.OwnerModeEpoch,
			blocked: r.OwnerBlockedAt.Valid, gender: r.OwnerGender, haveInputs: true, lifeMode: r.OwnerLifeMode,
			pregnant: r.OwnerPregnancyActive != 0, goal: r.OwnerUserGoal})
	}
	if err := l.sync(ctx, in); err != nil {
		return Snapshot{}, err
	}
	s := build(ownerID, actorID, in)
	l.pairs[key] = s
	return s, nil
}

// ForMember is one snapshot per owner of memberID (active links only), ordered by owner id — one statement.
func (l *Loader) ForMember(ctx context.Context, memberID uint64) ([]Snapshot, error) {
	if memberID == 0 {
		return nil, nil
	}
	if all, ok := l.members[memberID]; ok {
		return all, nil
	}
	rows, err := store.New(l.db).FamilySnapshotForMember(ctx, sql.NullInt64{Int64: int64(memberID), Valid: true}) //nolint:gosec // ids fit
	if err != nil {
		return nil, fmt.Errorf("family: member snapshot: %w", err)
	}
	in := make([]snapRow, 0, len(rows))
	for _, r := range rows {
		in = append(in, snapRow{linkID: r.LinkID, ownerID: r.OwnerID, memberID: u64(r.MemberID), typ: r.Type,
			role: r.Role.String, status: r.Status, expiresAt: r.ExpiresAt, section: r.Section, level: r.Level,
			grantEpoch: r.GrantModeEpoch, stateMode: r.OwnerMode, stateEpoch: r.OwnerModeEpoch,
			blocked: r.OwnerBlockedAt.Valid, gender: r.OwnerGender, haveInputs: true, lifeMode: r.OwnerLifeMode,
			pregnant: r.OwnerPregnancyActive != 0, goal: r.OwnerUserGoal})
	}
	if err := l.sync(ctx, in); err != nil {
		return nil, err
	}
	var out []Snapshot
	for start := 0; start < len(in); {
		end := start
		for end < len(in) && in[end].ownerID == in[start].ownerID {
			end++
		}
		out = append(out, build(in[start].ownerID, memberID, in[start:end]))
		start = end
	}
	l.members[memberID] = out
	return out, nil
}

// WithGuardians adds the actor's guardianships of childIDs to s (one statement).
func (l *Loader) WithGuardians(ctx context.Context, s Snapshot, childIDs ...uint64) (Snapshot, error) {
	if len(childIDs) == 0 || s.ActorID == 0 {
		return s, nil
	}
	rows, err := store.New(l.db).FamilyGuardiansOfUser(ctx, store.FamilyGuardiansOfUserParams{UserID: s.ActorID, ChildIds: childIDs})
	if err != nil {
		return s, fmt.Errorf("family: guardians: %w", err)
	}
	s.Guardians = make([]Guardianship, 0, len(rows))
	for _, r := range rows {
		s.Guardians = append(s.Guardians, guardianship(r.ChildID, r.Level, r.SubScopes.V, r.Status))
	}
	return s, nil
}

func guardianship(childID uint64, level string, subScopes json.RawMessage, status string) Guardianship {
	g := Guardianship{ChildID: childID, Active: status == "active", Limited: level != "write"}
	if g.Limited {
		var codes []string
		if len(subScopes) > 0 && json.Unmarshal(subScopes, &codes) == nil {
			for _, c := range codes {
				if sc := ParseSubScope(c); sc != ScopeOther {
					g.SubScopes |= sc
				}
			}
		}
	}
	return g
}

// sync runs SyncMode for every owner in rows whose stored state is missing or differs from the resolved mode, and
// patches the rows with the fresh state.
func (l *Loader) sync(ctx context.Context, rows []snapRow) error {
	done := map[uint64]OwnerState{}
	for i := range rows {
		r := &rows[i]
		if !r.haveInputs {
			continue
		}
		st, seen := done[r.ownerID]
		if !seen {
			want := resolveMode(r.lifeMode.String, r.pregnant, r.goal.String)
			epoch, _ := u32(r.stateEpoch)
			st = OwnerState{Mode: r.stateMode.String, Epoch: epoch}
			if !r.stateMode.Valid || r.stateMode.String != want {
				var err error
				if st, err = writeMode(ctx, l.db, r.ownerID, want, l.now()); err != nil {
					return err
				}
			}
			done[r.ownerID] = st
		}
		r.stateMode = sql.NullString{String: st.Mode, Valid: true}
		r.stateEpoch = sql.NullInt32{Int32: int32(st.Epoch), Valid: true} //nolint:gosec // epochs are small
	}
	return nil
}

// SyncMode resolves ownerID's effective mode (enums.ResolveLifeMode over the stored life mode, an active pregnancy
// and the goal) and stores it: a new owner starts at epoch 1; a changed mode bumps mode_epoch and sets review_pending
// (F-23: every epoch-bound grant pauses until the owner reviews). Call it after every mode write, inside the same
// transaction when there is one; snapshot loading also calls it lazily.
func SyncMode(ctx context.Context, db store.DBTX, ownerID uint64, now time.Time) (OwnerState, error) {
	in, err := store.New(db).FamilyOwnerModeInputs(ctx, ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OwnerState{}, nil // no such account: nothing to sync, every check denies
		}
		return OwnerState{}, fmt.Errorf("family: mode inputs: %w", err)
	}
	want := resolveMode(in.LifeMode.String, in.PregnancyActive != 0, in.UserGoal.String)
	if in.StateMode.Valid && in.StateMode.String == want {
		e, _ := u32(in.StateModeEpoch)
		return OwnerState{Mode: want, Epoch: e, ReviewPending: in.StateReviewPending.Valid && in.StateReviewPending.Bool}, nil
	}
	return writeMode(ctx, db, ownerID, want, now)
}

func writeMode(ctx context.Context, db store.DBTX, ownerID uint64, mode string, now time.Time) (OwnerState, error) {
	q := store.New(db)
	if _, err := q.UpsertFamilyOwnerState(ctx, store.UpsertFamilyOwnerStateParams{
		OwnerID: ownerID, Mode: mode, Now: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return OwnerState{}, fmt.Errorf("family: sync mode: %w", err)
	}
	row, err := q.GetFamilyOwnerState(ctx, ownerID)
	if err != nil {
		return OwnerState{}, fmt.Errorf("family: owner state: %w", err)
	}
	return OwnerState{Mode: row.Mode, Epoch: row.ModeEpoch, ReviewPending: row.ReviewPending}, nil
}

// RecheckTx re-reads the link and grant rows of ownerID → actorID (and the actor's guardian rows of childIDs) with
// shared locks inside the caller's write transaction (CMP-L4): a revoke or grant change committed after the request's
// first check wins, and none can commit until this transaction ends. The owner state is the stored one (the
// request's earlier snapshot synced it); a missing state denies every epoch-bound domain.
func RecheckTx(ctx context.Context, tx store.DBTX, ownerID, actorID uint64, childIDs ...uint64) (Snapshot, error) {
	s := Snapshot{OwnerID: ownerID, ActorID: actorID}
	if ownerID != 0 && actorID != 0 && ownerID != actorID {
		rows, err := store.New(tx).FamilyLockPair(ctx, store.FamilyLockPairParams{
			OwnerID: ownerID, MemberID: sql.NullInt64{Int64: int64(actorID), Valid: true}, //nolint:gosec // ids fit
		})
		if err != nil {
			return s, fmt.Errorf("family: recheck: %w", err)
		}
		in := make([]snapRow, 0, len(rows))
		for _, r := range rows {
			in = append(in, snapRow{linkID: r.LinkID, ownerID: r.OwnerID, memberID: u64(r.MemberID), typ: r.Type,
				role: r.Role.String, status: r.Status, expiresAt: r.ExpiresAt, section: r.Section, level: r.Level,
				grantEpoch: r.GrantModeEpoch, stateMode: r.OwnerMode, stateEpoch: r.OwnerModeEpoch,
				blocked: r.OwnerBlockedAt.Valid, gender: r.OwnerGender})
		}
		s = build(ownerID, actorID, in)
	}
	if len(childIDs) > 0 && actorID != 0 {
		rows, err := store.New(tx).FamilyLockGuardiansOfUser(ctx, store.FamilyLockGuardiansOfUserParams{UserID: actorID, ChildIds: childIDs})
		if err != nil {
			return s, fmt.Errorf("family: recheck guardians: %w", err)
		}
		for _, r := range rows {
			s.Guardians = append(s.Guardians, guardianship(r.ChildID, r.Level, r.SubScopes.V, r.Status))
		}
	}
	return s, nil
}
