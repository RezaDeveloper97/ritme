// Package children is the children API (bloom B-N5-02; artboards nbl_v15_Children / _AddChild, nbl_v16_ChildHome /
// _Growth / _Vaccines / _Milestones / _Learn): child profiles, growth measurements placed on the WHO Child Growth
// Standards (package growth, seeds/who), the Iran national immunisation schedule with per-child doses and 3-day
// reminders, milestone checks by age band and age-based learn tips.
//
// Built on what exists, nothing duplicated:
//   - the clinical catalogs (vaccine schedule, milestones, activities, doctor notes, age notes, learn tips) are
//     catalog_items groups (CB-CORE-03) — admin-editable through /api/admin/v1/catalog/{group}, cached by
//     catalog.Reader;
//   - family sharing is B-N4-01's families / family_children: a child the owner shares with her active spouse link
//     is visible to that spouse, read-only, and every such read is written to the owner's companion audit trail
//     (section children) before the data is built;
//   - reminders go through notifications (category checkups) — see reminders.go.
//
// Health data: every read and write is scoped to the owner or an active spouse family; nothing child-related is
// logged. Other domains (canvas directory bookings, insurance members, teen) reuse Service.Access / Service.List
// instead of reading the tables.
package children

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Codes.
var (
	Sexes         = []string{"girl", "boy"}
	DeliveryTypes = []string{"vaginal", "cesarean"}
)

// Limits.
const (
	MaxChildren   = 10 // per owner
	MaxNameLen    = 64
	MaxNoteLen    = 500
	MaxAgeYears   = 18
	MinWeightKg   = 0.3
	MaxWeightKg   = 60.0
	MinLengthCm   = 20.0
	MaxLengthCm   = 150.0
	MinHeadCm     = 15.0
	MaxHeadCm     = 70.0
	MaxBirthKg    = 7.0
	MaxBirthLenCm = 70.0
	MaxBirthHead  = 50.0
)

// Roles of the viewer on a child.
const (
	RoleOwner  = "owner"
	RoleShared = "shared" // the owner's spouse through a family (read-only)
)

// Errors.
var (
	// ErrNotFound: the child does not exist or the user may not see it (uniform for IDOR).
	ErrNotFound = errors.New("children: not found")
	// ErrReadOnly: the user sees the child through a family share and tried to change it.
	ErrReadOnly = errors.New("children: read-only")
	// ErrTooMany: the owner reached MaxChildren.
	ErrTooMany = errors.New("children: too many")
	// ErrMeasurementNotFound: no such measurement on the child.
	ErrMeasurementNotFound = errors.New("children: measurement not found")
	// ErrDateTaken: another measurement already sits on that day.
	ErrDateTaken = errors.New("children: measurement date taken")
	// ErrUnknownBand: ?month= is not a milestone band.
	ErrUnknownBand = errors.New("children: unknown milestone band")
)

// Access is a child the user may see and how.
type Access struct {
	Child       store.Child
	Role        string
	CompanionID uint64 // the spouse link (shared only)
}

// CanEdit reports whether the viewer may change the child.
func (a Access) CanEdit() bool { return a.Role == RoleOwner }

// Service is the children logic.
type Service struct {
	db  *sql.DB // nil in tests that pass a transaction
	q   *store.Queries
	cat ItemReader
}

// NewService returns a Service on db (a *sql.DB or a transaction) reading the catalogs through cat.
func NewService(db store.DBTX, cat ItemReader) *Service {
	s := &Service{q: store.New(db), cat: cat}
	if pool, ok := db.(*sql.DB); ok {
		s.db = pool
	}
	return s
}

// nid is a positive id as a nullable column value.
func nid(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: true} } //nolint:gosec // G115: auto-increment id

func tehranNow(now time.Time) sql.NullTime {
	return sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

// Access loads child for userID: the owner, or the owner's active spouse through a shared family (audited as a
// read in the owner's companion trail before anything is returned). Anything else is ErrNotFound.
func (s *Service) Access(ctx context.Context, userID, childID uint64, now time.Time) (Access, error) {
	c, err := s.q.GetChild(ctx, childID)
	if errors.Is(err, sql.ErrNoRows) {
		return Access{}, ErrNotFound
	}
	if err != nil {
		return Access{}, fmt.Errorf("children: get: %w", err)
	}
	if c.OwnerID == userID {
		return Access{Child: c, Role: RoleOwner}, nil
	}
	cid, err := s.q.GetSharedLink(ctx, store.GetSharedLinkParams{ChildID: childID, UserID: nid(userID)})
	if errors.Is(err, sql.ErrNoRows) {
		return Access{}, ErrNotFound
	}
	if err != nil {
		return Access{}, fmt.Errorf("children: shared link: %w", err)
	}
	if err := s.audit(ctx, c.OwnerID, userID, cid, now); err != nil {
		return Access{}, err
	}
	return Access{Child: c, Role: RoleShared, CompanionID: cid}, nil
}

// Editable is Access for a write: a shared child is ErrReadOnly.
func (s *Service) Editable(ctx context.Context, userID, childID uint64, now time.Time) (Access, error) {
	a, err := s.Access(ctx, userID, childID, now)
	if err != nil {
		return Access{}, err
	}
	if !a.CanEdit() {
		return Access{}, ErrReadOnly
	}
	return a, nil
}

func (s *Service) audit(ctx context.Context, ownerID, actorID, companionID uint64, now time.Time) error {
	if err := s.q.InsertChildAudit(ctx, store.InsertChildAuditParams{
		OwnerID:     ownerID,
		ActorID:     nid(actorID),
		CompanionID: nid(companionID),
		Now:         tehranNow(now),
	}); err != nil {
		return fmt.Errorf("children: audit: %w", err) // fail closed: no audit, no read
	}
	return nil
}

// List is every child the user sees: their own (youngest first), then the ones shared with them. Shared reads are
// audited once per link.
func (s *Service) List(ctx context.Context, userID uint64, now time.Time) ([]Access, error) {
	own, err := s.q.ListOwnedChildren(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("children: list: %w", err)
	}
	shared, err := s.q.ListSharedChildren(ctx, store.ListSharedChildrenParams{UserID: nid(userID)})
	if err != nil {
		return nil, fmt.Errorf("children: list shared: %w", err)
	}
	out := make([]Access, 0, len(own)+len(shared))
	for _, c := range own {
		out = append(out, Access{Child: c, Role: RoleOwner})
	}
	audited := map[uint64]bool{}
	for _, r := range shared {
		if !audited[r.CompanionID] {
			if err := s.audit(ctx, r.OwnerID, userID, r.CompanionID, now); err != nil {
				return nil, err
			}
			audited[r.CompanionID] = true
		}
		out = append(out, Access{Role: RoleShared, CompanionID: r.CompanionID, Child: store.Child{
			ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, BirthDate: r.BirthDate, Sex: r.Sex, BirthWeightKg: r.BirthWeightKg,
			BirthLengthCm: r.BirthLengthCm, BirthHeadCm: r.BirthHeadCm, DeliveryType: r.DeliveryType,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}})
	}
	return out, nil
}

// SharedWith are the children an owner shares with one spouse viewer (the companion home card), without auditing:
// the caller audits.
func (s *Service) SharedWith(ctx context.Context, viewerID, ownerID uint64) ([]store.Child, error) {
	rows, err := s.q.ListSharedChildren(ctx, store.ListSharedChildrenParams{UserID: nid(viewerID)})
	if err != nil {
		return nil, fmt.Errorf("children: list shared: %w", err)
	}
	var out []store.Child
	for _, r := range rows {
		if r.OwnerID != ownerID {
			continue
		}
		out = append(out, store.Child{
			ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, BirthDate: r.BirthDate, Sex: r.Sex, BirthWeightKg: r.BirthWeightKg,
			BirthLengthCm: r.BirthLengthCm, BirthHeadCm: r.BirthHeadCm, DeliveryType: r.DeliveryType,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

// OwnsChildren reports whether every id is one of ownerID's children (companion.ChildOwnership: the spouse
// shared-children picker).
func (s *Service) OwnsChildren(ctx context.Context, ownerID uint64, ids []uint64) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	own, err := s.q.ListOwnedChildIDs(ctx, ownerID)
	if err != nil {
		return false, fmt.Errorf("children: owned ids: %w", err)
	}
	set := make(map[uint64]bool, len(own))
	for _, id := range own {
		set[id] = true
	}
	for _, id := range ids {
		if !set[id] {
			return false, nil
		}
	}
	return true, nil
}

// OwnerName is the owner's display name ("" when unset).
func (s *Service) OwnerName(ctx context.Context, ownerID uint64) (string, error) {
	n, err := s.q.GetUserName(ctx, ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("children: owner name: %w", err)
	}
	return n.String, nil
}

// ChildInput is a validated child form.
type ChildInput struct {
	Name          string
	BirthDate     civildate.Date
	Sex           sql.NullString
	BirthWeightKg sql.NullString
	BirthLengthCm sql.NullString
	BirthHeadCm   sql.NullString
	DeliveryType  sql.NullString
}

// Create adds a child for ownerID (ErrTooMany past MaxChildren).
func (s *Service) Create(ctx context.Context, ownerID uint64, in ChildInput, now time.Time) (uint64, error) {
	n, err := s.q.CountOwnedChildren(ctx, ownerID)
	if err != nil {
		return 0, fmt.Errorf("children: count: %w", err)
	}
	if n >= MaxChildren {
		return 0, ErrTooMany
	}
	id, err := s.q.CreateChild(ctx, store.CreateChildParams{
		OwnerID: ownerID, Name: in.Name, BirthDate: in.BirthDate, Sex: in.Sex, BirthWeightKg: in.BirthWeightKg,
		BirthLengthCm: in.BirthLengthCm, BirthHeadCm: in.BirthHeadCm, DeliveryType: in.DeliveryType, Now: tehranNow(now),
	})
	if err != nil {
		return 0, fmt.Errorf("children: create: %w", err)
	}
	return uint64(id), nil //nolint:gosec // G115: auto-increment id
}

// Update replaces the child's profile (owner only; the caller checked Editable).
func (s *Service) Update(ctx context.Context, a Access, in ChildInput, now time.Time) error {
	if err := s.q.UpdateChild(ctx, store.UpdateChildParams{
		Name: in.Name, BirthDate: in.BirthDate, Sex: in.Sex, BirthWeightKg: in.BirthWeightKg, BirthLengthCm: in.BirthLengthCm,
		BirthHeadCm: in.BirthHeadCm, DeliveryType: in.DeliveryType, Now: tehranNow(now), ID: a.Child.ID, OwnerID: a.Child.OwnerID,
	}); err != nil {
		return fmt.Errorf("children: update: %w", err)
	}
	return nil
}

// Delete removes the child with its measurements, doses, checks and family shares (FK cascade).
func (s *Service) Delete(ctx context.Context, a Access) error {
	if _, err := s.q.DeleteChild(ctx, store.DeleteChildParams{ID: a.Child.ID, OwnerID: a.Child.OwnerID}); err != nil {
		return fmt.Errorf("children: delete: %w", err)
	}
	return nil
}

// Measurements are the child's measurements, newest first.
func (s *Service) Measurements(ctx context.Context, childID uint64) ([]store.ChildMeasurement, error) {
	rows, err := s.q.ListMeasurements(ctx, childID)
	if err != nil {
		return nil, fmt.Errorf("children: measurements: %w", err)
	}
	return rows, nil
}

// MeasurementInput is a validated measurement; Null fields were not sent.
type MeasurementInput struct {
	MeasuredOn civildate.Date
	WeightKg   sql.NullString
	LengthCm   sql.NullString
	HeadCm     sql.NullString
}

// SaveMeasurement stores the day's measurement (a second save of the same day merges) and returns it.
func (s *Service) SaveMeasurement(ctx context.Context, childID uint64, in MeasurementInput, now time.Time) (store.ChildMeasurement, error) {
	if err := s.q.UpsertMeasurement(ctx, store.UpsertMeasurementParams{
		ChildID: childID, MeasuredOn: in.MeasuredOn, WeightKg: in.WeightKg, LengthCm: in.LengthCm, HeadCm: in.HeadCm, Now: tehranNow(now),
	}); err != nil {
		return store.ChildMeasurement{}, fmt.Errorf("children: save measurement: %w", err)
	}
	m, err := s.q.GetMeasurementOn(ctx, store.GetMeasurementOnParams{ChildID: childID, MeasuredOn: in.MeasuredOn})
	if err != nil {
		return store.ChildMeasurement{}, fmt.Errorf("children: reload measurement: %w", err)
	}
	return m, nil
}

// Measurement loads one measurement of the child.
func (s *Service) Measurement(ctx context.Context, childID, id uint64) (store.ChildMeasurement, error) {
	m, err := s.q.GetMeasurement(ctx, store.GetMeasurementParams{ID: id, ChildID: childID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.ChildMeasurement{}, ErrMeasurementNotFound
	}
	if err != nil {
		return store.ChildMeasurement{}, fmt.Errorf("children: measurement: %w", err)
	}
	return m, nil
}

// UpdateMeasurement replaces a measurement (ErrDateTaken when another row has the new day).
func (s *Service) UpdateMeasurement(ctx context.Context, m store.ChildMeasurement, in MeasurementInput, now time.Time) (store.ChildMeasurement, error) {
	if in.MeasuredOn != m.MeasuredOn {
		other, err := s.q.GetMeasurementOn(ctx, store.GetMeasurementOnParams{ChildID: m.ChildID, MeasuredOn: in.MeasuredOn})
		switch {
		case err == nil && other.ID != m.ID:
			return store.ChildMeasurement{}, ErrDateTaken
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return store.ChildMeasurement{}, fmt.Errorf("children: measurement day: %w", err)
		}
	}
	if err := s.q.UpdateMeasurement(ctx, store.UpdateMeasurementParams{
		MeasuredOn: in.MeasuredOn, WeightKg: in.WeightKg, LengthCm: in.LengthCm, HeadCm: in.HeadCm, Now: tehranNow(now),
		ID: m.ID, ChildID: m.ChildID,
	}); err != nil {
		return store.ChildMeasurement{}, fmt.Errorf("children: update measurement: %w", err)
	}
	return s.Measurement(ctx, m.ChildID, m.ID)
}

// DeleteMeasurement removes one measurement of the child.
func (s *Service) DeleteMeasurement(ctx context.Context, childID, id uint64) error {
	n, err := s.q.DeleteMeasurement(ctx, store.DeleteMeasurementParams{ID: id, ChildID: childID})
	if err != nil {
		return fmt.Errorf("children: delete measurement: %w", err)
	}
	if n == 0 {
		return ErrMeasurementNotFound
	}
	return nil
}

// Schedule is the child's vaccine schedule on today.
func (s *Service) Schedule(ctx context.Context, c store.Child, today civildate.Date) (Schedule, error) {
	es, err := entries(ctx, s.cat, GroupVaccines)
	if err != nil {
		return Schedule{}, err
	}
	given, err := s.q.ListDoses(ctx, c.ID)
	if err != nil {
		return Schedule{}, fmt.Errorf("children: doses: %w", err)
	}
	return BuildSchedule(es, given, c.BirthDate, today), nil
}

// MarkDoses records doses as given on day (with an optional note, replacing an earlier record of the same dose).
func (s *Service) MarkDoses(ctx context.Context, childID uint64, codes []string, day civildate.Date, note sql.NullString, now time.Time) error {
	return s.inTx(ctx, func(q *store.Queries) error {
		for _, code := range codes {
			if err := q.UpsertDose(ctx, store.UpsertDoseParams{
				ChildID: childID, DoseCode: code, GivenOn: day, Note: note, Now: tehranNow(now),
			}); err != nil {
				return fmt.Errorf("children: mark dose: %w", err)
			}
		}
		return nil
	})
}

// UnmarkDose removes a dose record (no error when there was none).
func (s *Service) UnmarkDose(ctx context.Context, childID uint64, code string) error {
	if _, err := s.q.DeleteDose(ctx, store.DeleteDoseParams{ChildID: childID, DoseCode: code}); err != nil {
		return fmt.Errorf("children: unmark dose: %w", err)
	}
	return nil
}

// Checks are the child's milestone checks by code.
func (s *Service) Checks(ctx context.Context, childID uint64) (map[string]civildate.Date, error) {
	rows, err := s.q.ListMilestoneChecks(ctx, childID)
	if err != nil {
		return nil, fmt.Errorf("children: checks: %w", err)
	}
	out := make(map[string]civildate.Date, len(rows))
	for _, r := range rows {
		out[r.MilestoneCode] = r.CheckedOn
	}
	return out, nil
}

// SetCheck marks (or unmarks) a milestone as seen on day.
func (s *Service) SetCheck(ctx context.Context, childID uint64, code string, checked bool, day civildate.Date, now time.Time) error {
	if !checked {
		if _, err := s.q.DeleteMilestoneCheck(ctx, store.DeleteMilestoneCheckParams{ChildID: childID, MilestoneCode: code}); err != nil {
			return fmt.Errorf("children: uncheck: %w", err)
		}
		return nil
	}
	if err := s.q.UpsertMilestoneCheck(ctx, store.UpsertMilestoneCheckParams{
		ChildID: childID, MilestoneCode: code, CheckedOn: day, Now: tehranNow(now),
	}); err != nil {
		return fmt.Errorf("children: check: %w", err)
	}
	return nil
}

// Catalog reads a child catalog group as typed entries.
func (s *Service) Catalog(ctx context.Context, group string) ([]Entry, error) {
	return entries(ctx, s.cat, group)
}

// LearnArticles are the published articles linked by slug.
func (s *Service) LearnArticles(ctx context.Context, slugs []string) (map[string]store.ListLearnArticlesRow, error) {
	out := map[string]store.ListLearnArticlesRow{}
	if len(slugs) == 0 {
		return out, nil
	}
	rows, err := s.q.ListLearnArticles(ctx, slugs)
	if err != nil {
		return nil, fmt.Errorf("children: learn articles: %w", err)
	}
	for _, r := range rows {
		out[r.Slug] = r
	}
	return out, nil
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	if s.db == nil {
		return fn(s.q)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("children: begin: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("children: commit: %w", err)
	}
	return nil
}
