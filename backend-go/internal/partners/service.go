package partners

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/partners/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// Sealer encrypts the personal blob at rest (*loss.NoteBox, PRIVATE_NOTE_KEY).
type Sealer interface {
	SealWith(plain string, ad []byte) (string, error)
	OpenWith(stored string, ad []byte) (string, error)
	Disabled() bool
}

// ImportMeta describes the lab a result becomes.
type ImportMeta struct{ Title, Category string }

// Importer copies a result sheet into the user's labs through the lab upload flow (internal/labs: encrypted storage,
// extraction, verify), so the copy is a normal user-owned lab that she verifies like any other.
type Importer interface {
	ImportResult(ctx context.Context, userID uint64, m ImportMeta, pdf []byte, locale string, now time.Time) (uint64, error)
}

// Options wires a Service.
type Options struct {
	DB       *sql.DB
	Sealer   Sealer
	Importer Importer
	Adapters []LabPartner // by Name(); the fake partner is always available
	Logger   *slog.Logger
	Clock    clock.Clock
}

// Service is the home lab request domain.
type Service struct {
	db       *sql.DB
	q        *store.Queries
	seal     Sealer
	imp      Importer
	adapters map[string]LabPartner
	logger   *slog.Logger
	clock    clock.Clock
}

// NewService wires the service.
func NewService(o Options) *Service {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Clock == nil {
		o.Clock = clock.Real{}
	}
	s := &Service{db: o.DB, q: store.New(o.DB), seal: o.Sealer, imp: o.Importer, adapters: map[string]LabPartner{},
		logger: o.Logger, clock: o.Clock}
	s.adapters[DefaultPartner] = Fake{Lookup: func(ctx context.Context, ref string) (string, error) {
		st, err := s.q.GetPartnerStatus(ctx, sql.NullString{String: ref, Valid: true})
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		return st.String, nil
	}}
	for _, a := range o.Adapters {
		if a != nil {
			s.adapters[a.Name()] = a
		}
	}
	return s
}

// Queries is the store handle (admin handlers).
func (s *Service) Queries() *store.Queries { return s.q }

func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

// Payload is what the sealed blob holds: everything personal the user typed.
type Payload struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	City     string `json:"city"`
	Address  string `json:"address"`
	Note     string `json:"note,omitempty"`
}

func payloadAD(userID uint64, reference string) []byte {
	return []byte("partner-lab:" + strconv.FormatUint(userID, 10) + ":" + reference)
}

// Details opens the sealed blob of a request of userID (ok=false once purged or unreadable).
func (s *Service) Details(userID uint64, reference string, enc sql.NullString) (Payload, bool) {
	var p Payload
	if !enc.Valid || enc.String == "" || s.seal == nil || s.seal.Disabled() {
		return p, false
	}
	plain, err := s.seal.OpenWith(enc.String, payloadAD(userID, reference))
	if err != nil || json.Unmarshal([]byte(plain), &p) != nil {
		return Payload{}, false
	}
	return p, true
}

// Input is a validated booking form.
type Input struct {
	ServiceID     uint64
	Payload       Payload
	PreferredDate civildate.Date
	Window        string
	ConsentShare  bool
	ConsentImport bool
}

var refEnc = base32.StdEncoding.WithPadding(base32.NoPadding)

func newReference() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("partners: reference: %w", err)
	}
	return "LR" + refEnc.EncodeToString(b), nil
}

// Services are the active catalog services (the Today cards and the form).
func (s *Service) Services(ctx context.Context) ([]store.PartnerLabService, error) {
	rows, err := s.q.ListActiveServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("partners: services: %w", err)
	}
	return rows, nil
}

// Create stores the request (personal fields sealed) and submits the allow-listed payload to the partner. A partner
// that refuses the submission leaves no row behind.
func (s *Service) Create(ctx context.Context, userID uint64, in Input, now time.Time) (store.ListUserRequestsRow, error) {
	var zero store.ListUserRequestsRow
	if s.seal == nil || s.seal.Disabled() {
		return zero, ErrStorage
	}
	svc, err := s.q.GetService(ctx, in.ServiceID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !svc.IsActive) {
		return zero, ErrServiceUnavail
	}
	if err != nil {
		return zero, fmt.Errorf("partners: service: %w", err)
	}
	partner, ok := s.adapters[svc.PartnerCode]
	if !ok {
		return zero, ErrServiceUnavail
	}
	open, err := s.q.CountOpenUserRequests(ctx, userID)
	if err != nil {
		return zero, fmt.Errorf("partners: open count: %w", err)
	}
	if open >= MaxOpenRequests {
		return zero, ErrTooManyOpen
	}
	ref, err := newReference()
	if err != nil {
		return zero, err
	}
	blob, err := json.Marshal(in.Payload)
	if err != nil {
		return zero, fmt.Errorf("partners: payload: %w", err)
	}
	sealed, err := s.seal.SealWith(string(blob), payloadAD(userID, ref))
	if err != nil {
		return zero, fmt.Errorf("partners: seal: %w", err)
	}
	id, err := s.q.CreateRequest(ctx, store.CreateRequestParams{Reference: ref, UserID: userID, ServiceID: svc.ID,
		PreferredDate: in.PreferredDate, PreferredWindow: in.Window, ConsentShare: in.ConsentShare,
		ConsentImport: in.ConsentImport, PayloadEnc: sql.NullString{String: sealed, Valid: true},
		CreatedAt: nullTime(now), UpdatedAt: nullTime(now)})
	if err != nil {
		return zero, fmt.Errorf("partners: create: %w", err)
	}
	pref, err := partner.Submit(ctx, Submission{Reference: ref, ServiceCode: svc.Code, FullName: in.Payload.FullName,
		Phone: in.Payload.Phone, City: in.Payload.City, Address: in.Payload.Address, PreferredDate: in.PreferredDate,
		Window: in.Window, Note: in.Payload.Note})
	if err != nil {
		if derr := s.q.DeleteUnsubmittedRequest(ctx, store.DeleteUnsubmittedRequestParams{ID: uint64(id), UserID: userID}); derr != nil { //nolint:gosec // G115: ids are positive
			s.logger.ErrorContext(ctx, "partners: cleanup of a refused submission failed", slog.String("error", derr.Error()))
		}
		s.logger.WarnContext(ctx, "partners: submit failed", slog.String("partner", svc.PartnerCode), slog.String("error", err.Error()))
		return zero, ErrPartnerUnavail
	}
	if err := s.q.SetPartnerRef(ctx, store.SetPartnerRefParams{PartnerRef: sql.NullString{String: pref, Valid: true},
		UpdatedAt: nullTime(now), ID: uint64(id)}); err != nil { //nolint:gosec // G115
		return zero, fmt.Errorf("partners: partner ref: %w", err)
	}
	return s.Get(ctx, userID, uint64(id)) //nolint:gosec // G115
}

// List is the user's requests, newest first.
func (s *Service) List(ctx context.Context, userID uint64) ([]store.ListUserRequestsRow, error) {
	rows, err := s.q.ListUserRequests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("partners: list: %w", err)
	}
	return rows, nil
}

// Get is one request of the user (ErrNotFound for another user's, the same as a missing one).
func (s *Service) Get(ctx context.Context, userID, id uint64) (store.ListUserRequestsRow, error) {
	r, err := s.q.GetUserRequest(ctx, store.GetUserRequestParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.ListUserRequestsRow{}, ErrNotFound
	}
	if err != nil {
		return store.ListUserRequestsRow{}, fmt.Errorf("partners: get: %w", err)
	}
	return store.ListUserRequestsRow(r), nil
}

func (s *Service) adapterOf(ctx context.Context, serviceID uint64) (LabPartner, store.PartnerLabService, error) {
	svc, err := s.q.GetService(ctx, serviceID)
	if err != nil {
		return nil, svc, fmt.Errorf("partners: service: %w", err)
	}
	a, ok := s.adapters[svc.PartnerCode]
	if !ok {
		return nil, svc, ErrPartnerUnavail
	}
	return a, svc, nil
}

func closedAt(to string, now time.Time) sql.NullTime {
	if to == StatusResultReady || to == StatusRejected || to == StatusCancelled {
		return nullTime(now)
	}
	return sql.NullTime{}
}

func (s *Service) move(ctx context.Context, id uint64, from, to string, now time.Time) error {
	n, err := s.q.MoveRequestStatus(ctx, store.MoveRequestStatusParams{ToStatus: to, ClosedAt: closedAt(to, now),
		Now: nullTime(now), ID: id, FromStatus: from})
	if err != nil {
		return fmt.Errorf("partners: move: %w", err)
	}
	if n == 0 {
		return ErrBadTransition
	}
	return nil
}

// Cancel withdraws the user's request while it is requested or accepted (and tells the partner, best effort).
func (s *Service) Cancel(ctx context.Context, userID, id uint64, now time.Time) error {
	r, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if !CanCancel(r.Status) {
		return ErrNotCancellable
	}
	if err := s.move(ctx, r.ID, r.Status, StatusCancelled, now); err != nil {
		if errors.Is(err, ErrBadTransition) {
			return ErrNotCancellable
		}
		return err
	}
	if r.PartnerRef.Valid {
		if a, _, aerr := s.adapterOf(ctx, r.ServiceID); aerr == nil {
			if cerr := a.Cancel(ctx, r.PartnerRef.String); cerr != nil {
				s.logger.WarnContext(ctx, "partners: partner cancel failed", slog.String("error", cerr.Error()))
			}
		}
	}
	return nil
}

// Advance is the admin queue's manual step for adapters that take them (the fake partner): the console reports the
// status as the partner would, then the same sync that polls real partners applies it.
func (s *Service) Advance(ctx context.Context, id uint64, to string, now time.Time) error {
	r, err := s.q.GetRequest(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("partners: get: %w", err)
	}
	a, _, err := s.adapterOf(ctx, r.ServiceID)
	if err != nil {
		return err
	}
	if m, ok := a.(ManualStatus); !ok || !m.ManualStatus() || !r.PartnerRef.Valid {
		return ErrManualUnsupported
	}
	if !CanMove(r.Status, to) {
		return ErrBadTransition
	}
	if err := s.q.SetPartnerStatus(ctx, store.SetPartnerStatusParams{PartnerStatus: sql.NullString{String: to, Valid: true},
		UpdatedAt: nullTime(now), ID: r.ID}); err != nil {
		return fmt.Errorf("partners: partner status: %w", err)
	}
	return s.syncOne(ctx, r, a, now)
}

func (s *Service) syncOne(ctx context.Context, r store.PartnerLabRequest, a LabPartner, now time.Time) error {
	st, err := a.Status(ctx, r.PartnerRef.String)
	if errors.Is(err, ErrNoUpdate) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("partners: status: %w", err)
	}
	if st == r.Status || !CanMove(r.Status, st) {
		return nil
	}
	return s.move(ctx, r.ID, r.Status, st, now)
}

// Sync asks every adapter for the status of the open requests and applies forward moves. Returns how many moved.
func (s *Service) Sync(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.q.ListOpenPartnerRequests(ctx)
	if err != nil {
		return 0, fmt.Errorf("partners: open requests: %w", err)
	}
	moved := 0
	for _, r := range rows {
		a, _, aerr := s.adapterOf(ctx, r.ServiceID)
		if aerr != nil {
			continue
		}
		before := r.Status
		if err := s.syncOne(ctx, r, a, now); err != nil {
			if !errors.Is(err, ErrBadTransition) {
				s.logger.WarnContext(ctx, "partners: sync failed", slog.Uint64("request", r.ID), slog.String("error", err.Error()))
			}
			continue
		}
		if cur, gerr := s.q.GetRequest(ctx, r.ID); gerr == nil && cur.Status != before {
			moved++
		}
	}
	return moved, nil
}

// Purge wipes the sealed personal blob of requests closed more than PurgeAfter ago. Returns how many.
func (s *Service) Purge(ctx context.Context, now time.Time) (int64, error) {
	n, err := s.q.PurgeRequestPayloads(ctx, store.PurgeRequestPayloadsParams{Now: nullTime(now), Cutoff: nullTime(now.Add(-PurgeAfter))})
	if err != nil {
		return 0, fmt.Errorf("partners: purge: %w", err)
	}
	return n, nil
}

// Background jobs (T-M9-11, docs/engine-v6/jobs.md): the status sync every 10 minutes and the payload purge hourly,
// both on the low queue, both also at start-up (when this process becomes the scheduling leader).
const (
	JobSync   = "partners:sync"
	JobPurge  = "partners:purge"
	SyncCron  = "*/10 * * * *"
	PurgeCron = "5 * * * *"
)

// RegisterJobs puts the sync and purge jobs on q. Errors are logged by the queue without request data.
func (s *Service) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	return errors.Join(
		q.RegisterPeriodic(queue.Job{
			Type: JobSync, Priority: queue.Low, MaxTries: 2, Backoff: []time.Duration{time.Minute}, Timeout: 5 * time.Minute,
		}, s.SyncJob(now), SyncCron, queue.Unique(9*time.Minute), queue.RunAtStart()),
		q.RegisterPeriodic(queue.Job{
			Type: JobPurge, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{time.Minute, 5 * time.Minute}, Timeout: 2 * time.Minute,
		}, s.PurgeJob(now), PurgeCron, queue.Unique(55*time.Minute), queue.RunAtStart()),
	)
}

// SyncJob is the partners:sync handler. Idempotent: a request only moves along its state machine (a repeated status
// is a no-op, ErrBadTransition is skipped).
func (s *Service) SyncJob(now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		_, err := s.Sync(ctx, now())
		return err
	}
}

// PurgeJob is the partners:purge handler. Idempotent: it nulls blobs that are already past the cutoff.
func (s *Service) PurgeJob(now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		_, err := s.Purge(ctx, now())
		return err
	}
}

// Import copies the partner's result into the user's labs (a normal lab she verifies). It needs the consent ticked on
// the request AND confirm=true now, the request in result_ready and no earlier import.
func (s *Service) Import(ctx context.Context, userID, id uint64, confirm bool, locale, def string, now time.Time) (uint64, error) {
	r, err := s.Get(ctx, userID, id)
	if err != nil {
		return 0, err
	}
	switch {
	case r.Status != StatusResultReady || !r.PartnerRef.Valid:
		return 0, ErrNoResult
	case r.ImportedLabID.Valid:
		return 0, ErrAlreadyImported
	case !r.ConsentImport || !confirm:
		return 0, ErrImportConsent
	case s.imp == nil:
		return 0, ErrStorage
	}
	a, _, err := s.adapterOf(ctx, r.ServiceID)
	if err != nil {
		return 0, err
	}
	n, err := s.q.ClaimRequestImport(ctx, store.ClaimRequestImportParams{UpdatedAt: nullTime(now), ID: r.ID, UserID: userID})
	if err != nil {
		return 0, fmt.Errorf("partners: claim import: %w", err)
	}
	if n == 0 {
		return 0, ErrAlreadyImported
	}
	release := func() {
		if cerr := s.q.ClearRequestImported(ctx, store.ClearRequestImportedParams{ID: r.ID, UserID: userID}); cerr != nil {
			s.logger.ErrorContext(ctx, "partners: releasing an import claim failed", slog.String("error", cerr.Error()))
		}
	}
	pdf, err := a.FetchResult(ctx, r.PartnerRef.String)
	if err != nil || len(pdf) == 0 {
		release()
		return 0, ErrPartnerUnavail
	}
	cat := "other"
	if r.ServiceKind == KindHomeDraw {
		cat = "blood"
	}
	labID, err := s.imp.ImportResult(ctx, userID, ImportMeta{Title: PickText(r.ServiceTitle, locale, def), Category: cat}, pdf, locale, now)
	clear(pdf)
	if err != nil {
		release()
		return 0, err
	}
	if err := s.q.SetRequestImportedLab(ctx, store.SetRequestImportedLabParams{ImportedLabID: sql.NullInt64{Int64: int64(labID), Valid: true}, //nolint:gosec // G115
		UpdatedAt: nullTime(now), ID: r.ID, UserID: userID}); err != nil {
		return 0, fmt.Errorf("partners: mark imported: %w", err)
	}
	return labID, nil
}
