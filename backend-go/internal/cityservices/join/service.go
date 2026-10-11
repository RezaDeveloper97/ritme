package join

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/cityservices/join/store"
	"github.com/ritme/backend-go/internal/files"
)

// FileStore is the slice of internal/files the applications use (*files.Service).
type FileStore interface {
	Disabled() bool
	Purpose(name string) (files.Purpose, bool)
	Put(ctx context.Context, owner uint64, purpose string, data []byte, now time.Time) (files.File, error)
	Get(ctx context.Context, owner, id uint64) (files.File, error)
	Delete(ctx context.Context, owner, id uint64) error
	Link(f files.File, now time.Time, ttl time.Duration) (files.Link, error)
}

// Service runs the applications of one account at a time (every method takes the caller's user id).
type Service struct {
	db       *sql.DB
	q        *store.Queries
	files    FileStore
	notifier Notifier
	logger   *slog.Logger
	codes    func() (string, error)
}

// NewService wires the service; notifier nil = the fake one, logger nil = slog.Default().
func NewService(db *sql.DB, fs FileStore, notifier Notifier, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if notifier == nil {
		notifier = FakeNotifier{Logger: logger}
	}
	return &Service{db: db, q: store.New(db), files: fs, notifier: notifier, logger: logger, codes: NewTrackingCode}
}

// Application is a stored row with its decoded draft.
type Application struct {
	Row   store.CityServiceApplication
	Draft Draft
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func nullJSON(v any, empty bool) sql.NullString {
	if empty {
		return sql.NullString{}
	}
	return sql.NullString{String: encode(v), Valid: true}
}

func decodeRow(r store.CityServiceApplication) Application {
	d := Draft{
		Step1: Step1{BusinessName: r.BusinessName.String, District: r.District.String, ContactName: r.ContactName.String,
			ContactPhone: r.ContactPhone.String, OwnerConfirmed: r.OwnerConfirmed},
		Step2: Step2{Lat: r.Lat.String, Lng: r.Lng.String, Address: r.Address.String, About: r.About.String},
		Step4: Step4{BookingMode: r.BookingMode.String, TermsAccepted: r.TermsAcceptedAt.Valid},
	}
	if r.CityID.Valid && r.CityID.Int64 > 0 {
		d.CityID = uint64(r.CityID.Int64)
	}
	decode(r.CategorySlugs.String, &d.CategorySlugs)
	decode(r.AgeRanges.String, &d.AgeRanges)
	decode(r.Amenities.String, &d.Amenities)
	decode(r.WeeklyHours.String, &d.WeeklyHours)
	decode(r.Services.String, &d.Services)
	decode(r.PhotoFileIds.String, &d.PhotoIDs)
	decode(r.Documents.String, &d.Documents)
	if d.Documents == nil {
		d.Documents = map[string]uint64{}
	}
	return Application{Row: r, Draft: d}
}

func (s *Service) get(ctx context.Context, userID, id uint64) (Application, error) {
	r, err := s.q.GetUserApplication(ctx, store.GetUserApplicationParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Application{}, ErrNotFound
	}
	if err != nil {
		return Application{}, fmt.Errorf("join: get: %w", err)
	}
	return decodeRow(r), nil
}

// Get is one of the user's applications (ErrNotFound for anyone else's).
func (s *Service) Get(ctx context.Context, userID, id uint64) (Application, error) {
	return s.get(ctx, userID, id)
}

// List is the user's applications, newest first.
func (s *Service) List(ctx context.Context, userID uint64) ([]Application, error) {
	rows, err := s.q.ListUserApplications(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("join: list: %w", err)
	}
	out := make([]Application, 0, len(rows))
	for _, r := range rows {
		out = append(out, decodeRow(r))
	}
	return out, nil
}

// Start returns the user's open draft (created = false) or opens a new one (ErrTooMany past MaxApplications).
func (s *Service) Start(ctx context.Context, userID uint64, now time.Time) (Application, bool, error) {
	r, err := s.q.GetUserDraft(ctx, userID)
	if err == nil {
		return decodeRow(r), false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Application{}, false, fmt.Errorf("join: draft: %w", err)
	}
	n, err := s.q.CountUserApplications(ctx, userID)
	if err != nil {
		return Application{}, false, fmt.Errorf("join: count: %w", err)
	}
	if n >= MaxApplications {
		return Application{}, false, ErrTooMany
	}
	id, err := s.q.CreateApplication(ctx, store.CreateApplicationParams{UserID: userID, CreatedAt: nullTime(now), UpdatedAt: nullTime(now)})
	if err != nil {
		return Application{}, false, fmt.Errorf("join: create: %w", err)
	}
	a, err := s.get(ctx, userID, uint64(id)) //nolint:gosec // G115: auto-increment id
	return a, err == nil, err
}

// Delete removes the user's draft and its files (a submitted application stays for the team).
func (s *Service) Delete(ctx context.Context, userID, id uint64) error {
	a, err := s.get(ctx, userID, id)
	if err != nil {
		return err
	}
	if a.Row.Status != StatusDraft {
		return ErrLocked
	}
	n, err := s.q.DeleteUserDraft(ctx, store.DeleteUserDraftParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("join: delete: %w", err)
	}
	if n == 0 {
		return ErrLocked
	}
	ids := append([]uint64(nil), a.Draft.PhotoIDs...)
	for _, fid := range a.Draft.Documents {
		ids = append(ids, fid)
	}
	for _, fid := range ids {
		s.dropFile(ctx, userID, fid, id)
	}
	return nil
}

// dropFile deletes one of the user's files; a failure only leaves an unreferenced file (the owner can still delete
// it under /files, and account deletion removes it).
func (s *Service) dropFile(ctx context.Context, userID, fileID, appID uint64) {
	if err := s.files.Delete(ctx, userID, fileID); err != nil && !errors.Is(err, files.ErrNotFound) {
		s.logger.WarnContext(ctx, "join: file delete failed", slog.Uint64("application_id", appID), slog.Uint64("file_id", fileID))
	}
}

// tx runs fn on the user's application row locked FOR UPDATE.
func (s *Service) tx(ctx context.Context, userID, id uint64, fn func(q *store.Queries, a Application) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("join: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	r, err := q.LockUserApplication(ctx, store.LockUserApplicationParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("join: lock: %w", err)
	}
	if err := fn(q, decodeRow(r)); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveStep stores one step (input is a Step1..Step4) and advances `step` to it at most. Only while editable.
func (s *Service) SaveStep(ctx context.Context, userID, id uint64, input any, now time.Time) (Application, error) {
	err := s.tx(ctx, userID, id, func(q *store.Queries, a Application) error {
		if !Editable(a.Row.Status) {
			return ErrLocked
		}
		var err error
		switch in := input.(type) {
		case Step1:
			city := sql.NullInt64{Int64: int64(in.CityID), Valid: in.CityID > 0} //nolint:gosec // G115: validated id
			err = q.SaveStep1(ctx, store.SaveStep1Params{BusinessName: nullStr(in.BusinessName),
				CategorySlugs: nullJSON(in.CategorySlugs, len(in.CategorySlugs) == 0), CityID: city,
				District: nullStr(in.District), ContactName: nullStr(in.ContactName), ContactPhone: nullStr(in.ContactPhone),
				OwnerConfirmed: in.OwnerConfirmed, UpdatedAt: nullTime(now), ID: id, UserID: userID})
		case Step2:
			err = q.SaveStep2(ctx, store.SaveStep2Params{Lat: nullStr(in.Lat), Lng: nullStr(in.Lng),
				Address: nullStr(in.Address), About: nullStr(in.About),
				AgeRanges: nullJSON(in.AgeRanges, len(in.AgeRanges) == 0), Amenities: nullJSON(in.Amenities, len(in.Amenities) == 0),
				WeeklyHours: nullJSON(in.WeeklyHours, len(in.WeeklyHours) == 0), UpdatedAt: nullTime(now), ID: id, UserID: userID})
		case Step3:
			err = q.SaveStep3(ctx, store.SaveStep3Params{Services: nullJSON(in.Services, len(in.Services) == 0),
				UpdatedAt: nullTime(now), ID: id, UserID: userID})
		case Step4:
			terms := a.Row.TermsAcceptedAt // keep the first acceptance time
			if !in.TermsAccepted {
				terms = sql.NullTime{}
			} else if !terms.Valid {
				terms = nullTime(now)
			}
			err = q.SaveStep4(ctx, store.SaveStep4Params{BookingMode: nullStr(in.BookingMode), TermsAcceptedAt: terms,
				UpdatedAt: nullTime(now), ID: id, UserID: userID})
		default:
			return fmt.Errorf("join: unknown step input %T", input)
		}
		if err != nil {
			return fmt.Errorf("join: save step: %w", err)
		}
		return nil
	})
	if err != nil {
		return Application{}, err
	}
	return s.get(ctx, userID, id)
}

func (s *Service) put(ctx context.Context, userID uint64, purpose string, data []byte, now time.Time) (files.File, error) {
	if s.files.Disabled() {
		return files.File{}, files.ErrDisabled
	}
	return s.files.Put(ctx, userID, purpose, data, now)
}

// AddPhoto stores data as a place_photo of the user and appends it to the application (MaxPhotos). Errors from
// files.Service.Put are returned as they are (the handler maps them like POST /files).
func (s *Service) AddPhoto(ctx context.Context, userID, id uint64, data []byte, now time.Time) (files.File, error) {
	a, err := s.get(ctx, userID, id)
	if err != nil {
		return files.File{}, err
	}
	if !PhotosOpen(a.Row.Status) {
		return files.File{}, ErrLocked
	}
	if len(a.Draft.PhotoIDs) >= MaxPhotos {
		return files.File{}, ErrPhotoLimit
	}
	f, err := s.put(ctx, userID, files.PurposePlacePhoto, data, now)
	if err != nil {
		return files.File{}, err
	}
	err = s.tx(ctx, userID, id, func(q *store.Queries, a Application) error {
		if !PhotosOpen(a.Row.Status) {
			return ErrLocked
		}
		if len(a.Draft.PhotoIDs) >= MaxPhotos {
			return ErrPhotoLimit
		}
		ids := append(slices.Clone(a.Draft.PhotoIDs), f.ID)
		return q.SetApplicationPhotos(ctx, store.SetApplicationPhotosParams{PhotoFileIds: nullJSON(ids, false),
			UpdatedAt: nullTime(now), ID: id, UserID: userID})
	})
	if err != nil {
		s.dropFile(ctx, userID, f.ID, id)
		return files.File{}, err
	}
	return f, nil
}

// RemovePhoto takes a photo off the draft and deletes its file (only while editable; ErrNotFound for an id that is
// not one of this application's photos).
func (s *Service) RemovePhoto(ctx context.Context, userID, id, fileID uint64, now time.Time) error {
	err := s.tx(ctx, userID, id, func(q *store.Queries, a Application) error {
		if !Editable(a.Row.Status) {
			return ErrLocked
		}
		i := slices.Index(a.Draft.PhotoIDs, fileID)
		if i < 0 {
			return ErrNotFound
		}
		ids := slices.Delete(slices.Clone(a.Draft.PhotoIDs), i, i+1)
		return q.SetApplicationPhotos(ctx, store.SetApplicationPhotosParams{PhotoFileIds: nullJSON(ids, len(ids) == 0),
			UpdatedAt: nullTime(now), ID: id, UserID: userID})
	})
	if err != nil {
		return err
	}
	s.dropFile(ctx, userID, fileID, id)
	return nil
}

// PutDocument stores data as a private place_licence file of the user in slot, replacing (and deleting) the slot's
// previous file. Only while editable.
func (s *Service) PutDocument(ctx context.Context, userID, id uint64, slot string, data []byte, now time.Time) (files.File, error) {
	a, err := s.get(ctx, userID, id)
	if err != nil {
		return files.File{}, err
	}
	if !Editable(a.Row.Status) {
		return files.File{}, ErrLocked
	}
	f, err := s.put(ctx, userID, files.PurposePlaceLicence, data, now)
	if err != nil {
		return files.File{}, err
	}
	var old uint64
	err = s.tx(ctx, userID, id, func(q *store.Queries, a Application) error {
		if !Editable(a.Row.Status) {
			return ErrLocked
		}
		old = a.Draft.Documents[slot]
		a.Draft.Documents[slot] = f.ID
		return q.SetApplicationDocuments(ctx, store.SetApplicationDocumentsParams{Documents: nullJSON(a.Draft.Documents, false),
			UpdatedAt: nullTime(now), ID: id, UserID: userID})
	})
	if err != nil {
		s.dropFile(ctx, userID, f.ID, id)
		return files.File{}, err
	}
	if old > 0 {
		s.dropFile(ctx, userID, old, id)
	}
	return f, nil
}

// RemoveDocument empties slot and deletes its file (ErrNotFound when the slot is empty). Only while editable.
func (s *Service) RemoveDocument(ctx context.Context, userID, id uint64, slot string, now time.Time) error {
	var old uint64
	err := s.tx(ctx, userID, id, func(q *store.Queries, a Application) error {
		if !Editable(a.Row.Status) {
			return ErrLocked
		}
		old = a.Draft.Documents[slot]
		if old == 0 {
			return ErrNotFound
		}
		delete(a.Draft.Documents, slot)
		return q.SetApplicationDocuments(ctx, store.SetApplicationDocumentsParams{
			Documents: nullJSON(a.Draft.Documents, len(a.Draft.Documents) == 0), UpdatedAt: nullTime(now), ID: id, UserID: userID})
	})
	if err != nil {
		return err
	}
	s.dropFile(ctx, userID, old, id)
	return nil
}

// LiveFiles resolves the application's photo and document ids to the user's files that still exist (a file deleted
// under /files simply drops out).
func (s *Service) LiveFiles(ctx context.Context, userID uint64, d Draft) (photos []files.File, docs map[string]files.File, err error) {
	photos = []files.File{}
	for _, fid := range d.PhotoIDs {
		f, err := s.files.Get(ctx, userID, fid)
		if errors.Is(err, files.ErrNotFound) || (err == nil && f.Purpose != files.PurposePlacePhoto) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("join: photo: %w", err)
		}
		photos = append(photos, f)
	}
	docs = map[string]files.File{}
	for _, slot := range Slots {
		fid := d.Documents[slot]
		if fid == 0 {
			continue
		}
		f, err := s.files.Get(ctx, userID, fid)
		if errors.Is(err, files.ErrNotFound) || (err == nil && f.Purpose != files.PurposePlaceLicence) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("join: document: %w", err)
		}
		docs[slot] = f
	}
	return photos, docs, nil
}

// Link is a fresh link to f (public for photos, 5-minute signed for documents).
func (s *Service) Link(f files.File, now time.Time) (files.Link, error) {
	return s.files.Link(f, now, 0)
}

// SubmitResult is a submit: the missing fields (nothing stored then) or the submitted application.
type SubmitResult struct {
	Missing     map[string]string
	Application Application
}

// Submit checks completeness (Missing) and moves a draft / needs_info application to submitted with a tracking code
// (kept on a resubmission), then sends the confirmation SMS (best effort: a failure is logged by id only).
func (s *Service) Submit(ctx context.Context, userID, id uint64, now time.Time) (SubmitResult, error) {
	a, err := s.get(ctx, userID, id)
	if err != nil {
		return SubmitResult{}, err
	}
	if !Editable(a.Row.Status) {
		return SubmitResult{}, ErrLocked
	}
	photos, docs, err := s.LiveFiles(ctx, userID, a.Draft)
	if err != nil {
		return SubmitResult{}, err
	}
	have := map[string]bool{}
	for slot := range docs {
		have[slot] = true
	}
	if m := Missing(a.Draft, len(photos), have); len(m) > 0 {
		return SubmitResult{Missing: m}, nil
	}
	code := a.Row.TrackingCode.String
	if code == "" {
		if code, err = s.freshCode(ctx); err != nil {
			return SubmitResult{}, err
		}
	}
	err = s.tx(ctx, userID, id, func(q *store.Queries, a Application) error {
		if !Editable(a.Row.Status) {
			return ErrLocked
		}
		return q.SubmitApplication(ctx, store.SubmitApplicationParams{TrackingCode: nullStr(code), SubmittedAt: nullTime(now),
			UpdatedAt: nullTime(now), ID: id, UserID: userID})
	})
	if err != nil {
		return SubmitResult{}, err
	}
	a, err = s.get(ctx, userID, id)
	if err != nil {
		return SubmitResult{}, err
	}
	if err := s.notifier.Submitted(ctx, Notice{ApplicationID: id, Mobile: a.Draft.ContactPhone,
		TrackingCode: a.Row.TrackingCode.String}); err != nil {
		s.logger.WarnContext(ctx, "join: submit sms failed", slog.Uint64("application_id", id))
	}
	return SubmitResult{Application: a}, nil
}

// freshCode draws tracking codes until one is unused (60 random bits: a clash is practically impossible; the unique
// key is the last word).
func (s *Service) freshCode(ctx context.Context) (string, error) {
	for range 5 {
		code, err := s.codes()
		if err != nil {
			return "", fmt.Errorf("join: tracking code: %w", err)
		}
		taken, err := s.q.TrackingCodeTaken(ctx, nullStr(code))
		if err != nil {
			return "", fmt.Errorf("join: tracking code: %w", err)
		}
		if !taken {
			return code, nil
		}
	}
	return "", errors.New("join: no free tracking code")
}

// Refs are the reference lists step validation checks against (active categories, catalog chips).
type Refs struct {
	Categories []string
	AgeRanges  []string
	Amenities  []string
}

// LoadRefs reads the active categories and the step-2 catalog codes.
func (s *Service) LoadRefs(ctx context.Context) (Refs, error) {
	var r Refs
	var err error
	if r.Categories, err = s.q.ActiveCategorySlugs(ctx); err != nil {
		return r, fmt.Errorf("join: categories: %w", err)
	}
	if r.AgeRanges, err = s.q.ActiveCatalogCodes(ctx, GroupAgeRanges); err != nil {
		return r, fmt.Errorf("join: age ranges: %w", err)
	}
	if r.Amenities, err = s.q.ActiveCatalogCodes(ctx, GroupAmenities); err != nil {
		return r, fmt.Errorf("join: amenities: %w", err)
	}
	return r, nil
}

// CityActive reports whether id is an active city.
func (s *Service) CityActive(ctx context.Context, id uint64) (bool, error) {
	ok, err := s.q.ActiveCityExists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("join: city: %w", err)
	}
	return ok, nil
}
