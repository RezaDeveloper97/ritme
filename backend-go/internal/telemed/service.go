package telemed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// Service is the directory's data access and slot logic.
type Service struct {
	db      *sql.DB
	q       *store.Queries
	catalog *catalog.Reader
	busy    Busy
	visits  VisitChecker
}

// NewService wires the service. busy / visits may be nil (NoBusy / NoVisits).
func NewService(conn *sql.DB, reader *catalog.Reader, busy Busy, visits VisitChecker) *Service {
	if busy == nil {
		busy = NoBusy{}
	}
	if visits == nil {
		visits = NoVisits{}
	}
	return &Service{db: conn, q: store.New(conn), catalog: reader, busy: busy, visits: visits}
}

// Queries exposes the store (admin handlers share the service).
func (s *Service) Queries() *store.Queries { return s.q }

// Catalog is the label reader.
func (s *Service) Catalog() *catalog.Reader { return s.catalog }

// Details is everything slot generation and a card need about one doctor.
type Details struct {
	Doctor   store.TelemedDoctor
	Visits   []store.TelemedVisitType // active ones, display order
	Rules    []Rule
	Blocked  []Interval // time off and busy intervals in the loaded window
	Insurers []string
}

// Visit is the active visit type of mode, if offered.
func (d *Details) Visit(mode string) (store.TelemedVisitType, bool) {
	for _, v := range d.Visits {
		if v.Mode == mode {
			return v, true
		}
	}
	return store.TelemedVisitType{}, false
}

// Lengths are the offered modes with their durations (all, or only mode when set).
func (d *Details) Lengths(mode string) []VisitLength {
	out := make([]VisitLength, 0, len(d.Visits))
	for _, v := range d.Visits {
		if mode == "" || v.Mode == mode {
			out = append(out, VisitLength{Mode: v.Mode, Duration: int(v.DurationMinutes)})
		}
	}
	return out
}

// RuleOf converts a stored rule.
func RuleOf(r store.TelemedAvailabilityRule) Rule {
	var modes []string
	if r.Modes.Valid {
		_ = json.Unmarshal(r.Modes.V, &modes)
	}
	return Rule{
		Weekday: time.Weekday(r.Weekday), StartMinute: int(r.StartMinute), EndMinute: int(r.EndMinute),
		SlotMinutes: int(r.SlotMinutes), Modes: modes,
	}
}

// window is the Tehran wall-clock span [from 00:00, from+days 00:00).
func window(from civildate.Date, days int) (time.Time, time.Time) {
	return from.TehranMidnight(), from.AddDays(days).TehranMidnight()
}

// LoadDetails reads visit types, rules, insurers, and the time off / busy intervals overlapping [from, from+days) of
// the given doctors.
func (s *Service) LoadDetails(ctx context.Context, doctors []store.TelemedDoctor, from civildate.Date, days int) (map[uint64]*Details, error) {
	out := make(map[uint64]*Details, len(doctors))
	if len(doctors) == 0 {
		return out, nil
	}
	ids := make([]uint64, 0, len(doctors))
	for _, d := range doctors {
		ids = append(ids, d.ID)
		out[d.ID] = &Details{Doctor: d}
	}
	visits, err := s.q.ListActiveVisitTypesByDoctors(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("telemed: visit types: %w", err)
	}
	for _, v := range visits {
		out[v.DoctorID].Visits = append(out[v.DoctorID].Visits, v)
	}
	rules, err := s.q.ListRulesByDoctors(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("telemed: rules: %w", err)
	}
	for _, r := range rules {
		out[r.DoctorID].Rules = append(out[r.DoctorID].Rules, RuleOf(r))
	}
	insurers, err := s.q.ListInsurersByDoctors(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("telemed: insurers: %w", err)
	}
	for _, i := range insurers {
		out[i.DoctorID].Insurers = append(out[i.DoctorID].Insurers, i.Insurer)
	}
	start, end := window(from, days)
	offs, err := s.q.ListTimeOffByDoctors(ctx, store.ListTimeOffByDoctorsParams{DoctorIds: ids, FromAt: start, ToAt: end})
	if err != nil {
		return nil, fmt.Errorf("telemed: time off: %w", err)
	}
	for _, o := range offs {
		out[o.DoctorID].Blocked = append(out[o.DoctorID].Blocked, Interval{Start: wall(o.StartsAt), End: wall(o.EndsAt)})
	}
	busy, err := s.busy.Busy(ctx, ids, start, end)
	if err != nil {
		return nil, fmt.Errorf("telemed: busy: %w", err)
	}
	for id, spans := range busy {
		if d, ok := out[id]; ok {
			d.Blocked = append(d.Blocked, spans...)
		}
	}
	return out, nil
}

// wall re-reads a DB datetime as Tehran wall-clock (the DSN already parses in Asia/Tehran; this pins the location).
func wall(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, civildate.Tehran)
}

// Card is one directory entry: the doctor with the slot facts of the query.
type Card struct {
	*Details
	Next     Slot
	NextMode string
	HasNext  bool
}

// Directory is the filtered directory in display order (all matches; the handler pages it).
func (s *Service) Directory(ctx context.Context, f Filter, now time.Time) ([]Card, error) {
	pattern, jsonPattern := "%", "%"
	if f.Q != "" {
		pattern, jsonPattern = form.Contains(f.Q), form.ContainsJSON(f.Q)
	}
	rows, err := s.q.ListDirectoryDoctors(ctx, store.ListDirectoryDoctorsParams{
		Mode: f.Mode, Kind: f.Kind, Specialty: f.Specialty, City: sql.NullString{String: f.City, Valid: true},
		Insurer: f.Insurance, Pattern: pattern, JsonPattern: jsonPattern, Limit: DirectoryCap,
	})
	if err != nil {
		return nil, fmt.Errorf("telemed: directory: %w", err)
	}
	today := civildate.InTehran(now)
	details, err := s.LoadDetails(ctx, rows, today, NextSlotDays)
	if err != nil {
		return nil, err
	}
	cards := make([]Card, 0, len(rows))
	for _, r := range rows {
		d := details[r.ID]
		next, mode, ok := FirstSlotAny(d.Rules, d.Lengths(f.Mode), today, NextSlotDays, now, d.Blocked)
		if f.Today && (!ok || civildate.InTehran(next.Start) != today) {
			continue
		}
		cards = append(cards, Card{Details: d, Next: next, NextMode: mode, HasNext: ok})
	}
	return cards, nil
}

// Listed reports ErrDoctorNotFound unless id is a listed doctor.
func (s *Service) Listed(ctx context.Context, id uint64) error {
	_, err := s.q.GetListedDoctor(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDoctorNotFound
	}
	if err != nil {
		return fmt.Errorf("telemed: doctor: %w", err)
	}
	return nil
}

// Doctor is one listed doctor with its details over the next-slot window (ErrDoctorNotFound otherwise).
func (s *Service) Doctor(ctx context.Context, id uint64, now time.Time) (Card, error) {
	row, err := s.q.GetListedDoctor(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Card{}, ErrDoctorNotFound
	}
	if err != nil {
		return Card{}, fmt.Errorf("telemed: doctor: %w", err)
	}
	today := civildate.InTehran(now)
	details, err := s.LoadDetails(ctx, []store.TelemedDoctor{row}, today, NextSlotDays)
	if err != nil {
		return Card{}, err
	}
	d := details[row.ID]
	next, mode, ok := FirstSlotAny(d.Rules, d.Lengths(""), today, NextSlotDays, now, d.Blocked)
	return Card{Details: d, Next: next, NextMode: mode, HasNext: ok}, nil
}

// SlotsResult is the slot grid of one mode.
type SlotsResult struct {
	Visit store.TelemedVisitType
	Days  []Day
	// Next is the first free slot from the window start on (within NextSlotDays), so the client can jump to it.
	Next    Slot
	HasNext bool
}

// Slots generates the free slots of one listed doctor. An unoffered mode is ErrModeNotOffered.
func (s *Service) Slots(ctx context.Context, id uint64, in SlotsInput, now time.Time) (SlotsResult, error) {
	row, err := s.q.GetListedDoctor(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return SlotsResult{}, ErrDoctorNotFound
	}
	if err != nil {
		return SlotsResult{}, fmt.Errorf("telemed: doctor: %w", err)
	}
	span := max(in.Days, NextSlotDays)
	details, err := s.LoadDetails(ctx, []store.TelemedDoctor{row}, in.From, span)
	if err != nil {
		return SlotsResult{}, err
	}
	return SlotsOf(details[row.ID], in, now)
}

// ErrModeNotOffered is a slot query for a mode the doctor does not offer.
var ErrModeNotOffered = errors.New("telemed: visit mode not offered")

// SlotsOf is Slots over loaded details (the admin preview shares it).
func SlotsOf(d *Details, in SlotsInput, now time.Time) (SlotsResult, error) {
	mode := in.Mode
	if mode == "" && len(d.Visits) > 0 {
		mode = d.Visits[0].Mode
	}
	v, ok := d.Visit(mode)
	if !ok {
		return SlotsResult{}, ErrModeNotOffered
	}
	q := SlotQuery{Rules: d.Rules, Mode: mode, Duration: int(v.DurationMinutes), From: in.From, Days: in.Days, Now: now, Blocked: d.Blocked}
	res := SlotsResult{Visit: v, Days: GenerateDays(q)}
	q.Days = NextSlotDays
	res.Next, res.HasNext = FirstSlot(q)
	return res, nil
}

// Review is one review as shown.
type Review struct {
	ID        uint64
	UserID    uint64
	Rating    int
	Body      string
	UserName  string
	CreatedAt time.Time
}

func reviewOf(id, userID uint64, rating uint8, body, name sql.NullString, created sql.NullTime) Review {
	r := Review{ID: id, UserID: userID, Rating: int(rating), Body: body.String, UserName: name.String}
	if created.Valid {
		r.CreatedAt = created.Time
	}
	return r
}

// Reviews is one page of a listed doctor's visible reviews and their total.
func (s *Service) Reviews(ctx context.Context, doctorID uint64, limit, offset int) ([]Review, int, error) {
	total, err := s.q.CountVisibleReviews(ctx, doctorID)
	if err != nil {
		return nil, 0, fmt.Errorf("telemed: count reviews: %w", err)
	}
	rows, err := s.q.ListVisibleReviews(ctx, store.ListVisibleReviewsParams{
		DoctorID: doctorID, Limit: int32(limit), Offset: int32(min(offset, 1<<30)), //nolint:gosec // G115: bounded
	})
	if err != nil {
		return nil, 0, fmt.Errorf("telemed: reviews: %w", err)
	}
	out := make([]Review, 0, len(rows))
	for _, r := range rows {
		out = append(out, reviewOf(r.ID, r.UserID, r.Rating, r.Body, r.UserName, r.CreatedAt))
	}
	return out, int(total), nil
}

// CanReview reports whether userID has a completed visit with the doctor still to review.
func (s *Service) CanReview(ctx context.Context, userID, doctorID uint64) (bool, error) {
	id, ok, err := s.visits.ReviewableVisit(ctx, userID, doctorID)
	if err != nil {
		return false, fmt.Errorf("telemed: reviewable visit: %w", err)
	}
	return ok && id > 0, nil
}

// CreateReview stores the review of the user's reviewable visit and refreshes the doctor's rating.
func (s *Service) CreateReview(ctx context.Context, userID, doctorID uint64, in ReviewInput, now time.Time) (Review, error) {
	if err := s.Listed(ctx, doctorID); err != nil {
		return Review{}, err
	}
	bookingID, ok, err := s.visits.ReviewableVisit(ctx, userID, doctorID)
	if err != nil {
		return Review{}, fmt.Errorf("telemed: reviewable visit: %w", err)
	}
	if !ok || bookingID == 0 {
		return Review{}, ErrReviewNotAllowed
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Review{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	id, err := q.InsertReview(ctx, store.InsertReviewParams{
		DoctorID: doctorID, UserID: userID, BookingID: sql.NullInt64{Int64: int64(bookingID), Valid: true}, //nolint:gosec // G115: auto-increment id
		Rating: uint8(in.Rating), Body: sql.NullString{String: in.Body, Valid: in.Body != ""}, //nolint:gosec // G115: validated 1–5
		Now: sql.NullTime{Time: now, Valid: true},
	})
	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == 1062 {
		return Review{}, ErrReviewExists
	}
	if err != nil {
		return Review{}, fmt.Errorf("telemed: insert review: %w", err)
	}
	if err := q.RefreshDoctorRating(ctx, doctorID); err != nil {
		return Review{}, fmt.Errorf("telemed: refresh rating: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Review{}, err
	}
	r, err := s.q.GetUserReview(ctx, store.GetUserReviewParams{ID: uint64(id), DoctorID: doctorID, UserID: userID}) //nolint:gosec // G115: auto-increment id
	if err != nil {
		return Review{}, fmt.Errorf("telemed: reload review: %w", err)
	}
	return reviewOf(r.ID, r.UserID, r.Rating, r.Body, r.UserName, r.CreatedAt), nil
}

// DeleteReview removes one of the user's own reviews (any other id is ErrReviewNotFound) and refreshes the rating.
func (s *Service) DeleteReview(ctx context.Context, userID, doctorID, reviewID uint64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	n, err := q.DeleteUserReview(ctx, store.DeleteUserReviewParams{ID: reviewID, DoctorID: doctorID, UserID: userID})
	if err != nil {
		return fmt.Errorf("telemed: delete review: %w", err)
	}
	if n == 0 {
		return ErrReviewNotFound
	}
	if err := q.RefreshDoctorRating(ctx, doctorID); err != nil {
		return fmt.Errorf("telemed: refresh rating: %w", err)
	}
	return tx.Commit()
}

// Labels are the catalog titles of the specialty / city / insurer codes, per group.
type Labels map[string]map[string]json.RawMessage

// Title is the raw translatable title of code in group (nil when the catalog has no active item).
func (l Labels) Title(group, code string) json.RawMessage { return l[group][code] }

// LoadLabels reads the three catalog groups (cached per group by the reader).
func (s *Service) LoadLabels(ctx context.Context) (Labels, error) {
	out := Labels{}
	for _, g := range []string{GroupSpecialties, GroupCities, GroupInsurers} {
		items, err := s.catalog.Items(ctx, g)
		if err != nil {
			return nil, err
		}
		m := make(map[string]json.RawMessage, len(items))
		for _, it := range items {
			m[it.Code] = it.Title
		}
		out[g] = m
	}
	return out, nil
}

// FilterCounts are the listed doctors per specialty / city / insurer code.
type FilterCounts struct {
	Specialties, Cities, Insurers map[string]int64
}

// Counts reads FilterCounts.
func (s *Service) Counts(ctx context.Context) (FilterCounts, error) {
	fc := FilterCounts{Specialties: map[string]int64{}, Cities: map[string]int64{}, Insurers: map[string]int64{}}
	sp, err := s.q.CountListedBySpecialty(ctx)
	if err != nil {
		return fc, fmt.Errorf("telemed: specialty counts: %w", err)
	}
	for _, r := range sp {
		fc.Specialties[r.Specialty] = r.Doctors
	}
	ci, err := s.q.CountListedByCity(ctx)
	if err != nil {
		return fc, fmt.Errorf("telemed: city counts: %w", err)
	}
	for _, r := range ci {
		fc.Cities[r.City.String] = r.Doctors
	}
	in, err := s.q.CountListedByInsurer(ctx)
	if err != nil {
		return fc, fmt.Errorf("telemed: insurer counts: %w", err)
	}
	for _, r := range in {
		fc.Insurers[r.Insurer] = r.Doctors
	}
	return fc, nil
}
