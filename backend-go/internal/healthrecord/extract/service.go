package extract

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/healthrecord/extract/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Errors of the extraction API.
var (
	// ErrNotFound: no such document of this user (a foreign id is the same 404).
	ErrNotFound = healthrecord.ErrDocumentNotFound
	// ErrNoFiles: the document has no file to read.
	ErrNoFiles = errors.New("extract: document has no files")
	// ErrRunning: an extraction of the document is already queued or running.
	ErrRunning = errors.New("extract: extraction running")
	// ErrConfirmed: the user already confirmed the document's values.
	ErrConfirmed = errors.New("extract: document confirmed")
	// ErrNotInReview: the document has no extracted values waiting for review.
	ErrNotInReview = errors.New("extract: document not in review")
)

// FileReader is the slice of internal/files the jobs use (*files.Service).
type FileReader interface {
	Open(ctx context.Context, owner, id uint64) (files.File, []byte, error)
}

// ConsentChecker re-checks the consent when a job runs (*consent.Service).
type ConsentChecker interface {
	Require(ctx context.Context, userID uint64, code string) error
}

// Refunder gives a reserved Plus use back (*plus.Service).
type Refunder interface {
	Refund(ctx context.Context, userID uint64, key plus.Key, reservedAt, now time.Time) error
}

// Options wires a Service.
type Options struct {
	DB        *sql.DB
	Docs      *healthrecord.Documents // the document view and the «where used» links
	Files     FileReader
	AI        *ai.Client // nil = AI unavailable (a queued job fails with ai_unavailable)
	Consents  ConsentChecker
	Plus      Refunder // nil = nothing is refunded (no reservation was made either)
	Pregnancy pregstore.Querier
	Clock     clock.Clock
	Logger    *slog.Logger
}

// Service is the extraction, review and dating-offer logic.
type Service struct {
	db       *sql.DB
	q        *store.Queries
	docs     *healthrecord.Documents
	files    FileReader
	client   *ai.Client
	consents ConsentChecker
	plus     Refunder
	preg     pregstore.Querier
	clock    clock.Clock
	logger   *slog.Logger
	runner   *Runner
}

// NewService wires the service.
func NewService(o Options) *Service {
	if o.Clock == nil {
		o.Clock = clock.Real{}
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	return &Service{db: o.DB, q: store.New(o.DB), docs: o.Docs, files: o.Files, client: o.AI, consents: o.Consents,
		plus: o.Plus, preg: o.Pregnancy, clock: o.Clock, logger: o.Logger}
}

// Docs is the documents service the handlers render with.
func (s *Service) Docs() *healthrecord.Documents { return s.docs }

func tehranSecond(t time.Time) time.Time { return t.In(civildate.Tehran).Truncate(time.Second) }

func (s *Service) now(ctx context.Context) time.Time {
	return tehranSecond(clock.FromContext(ctx, s.clock).Now())
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("extract: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("extract: commit: %w", err)
	}
	return nil
}

func lockDoc(ctx context.Context, q *store.Queries, userID, id uint64) (store.RecordDocument, error) {
	doc, err := q.LockExtractDocument(ctx, store.LockExtractDocumentParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return doc, ErrNotFound
	}
	if err != nil {
		return doc, fmt.Errorf("extract: lock document: %w", err)
	}
	return doc, nil
}

// Start queues the extraction of the user's document (Plus use already reserved by the AI gate when reserved is
// non-zero; it is given back when the extraction fails). The job runs on the workers (or now, in sync mode).
func (s *Service) Start(ctx context.Context, userID, docID uint64, locale string, reserved, now time.Time) error {
	err := s.inTx(ctx, func(q *store.Queries) error {
		doc, err := lockDoc(ctx, q, userID, docID)
		if err != nil {
			return err
		}
		if _, ok := Schemas[doc.Kind]; !ok {
			return ErrNotFound // not a record document kind (never stored, defensive)
		}
		switch doc.ReviewState {
		case healthrecord.ReviewPending:
			return ErrRunning
		case healthrecord.ReviewConfirmed:
			return ErrConfirmed
		}
		ids, err := q.ListExtractFileIDs(ctx, store.ListExtractFileIDsParams{DocumentID: docID, UserID: userID})
		if err != nil {
			return fmt.Errorf("extract: files: %w", err)
		}
		if len(ids) == 0 {
			return ErrNoFiles
		}
		job := &Job{State: jobQueued, AvailableAt: now.Unix(), Locale: locale}
		if !reserved.IsZero() {
			job.ReservedAt = isoTime(reserved)
		}
		st := &Stored{Schema: doc.Kind, Status: StatusPending, RequestedAt: isoTime(now), Job: job}
		raw, err := st.raw()
		if err != nil {
			return err
		}
		if _, err := q.SetExtractState(ctx, store.SetExtractStateParams{ReviewState: healthrecord.ReviewPending,
			Extracted: raw, Now: nullTime(now), ID: docID, UserID: userID}); err != nil {
			return fmt.Errorf("extract: queue: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.runner.dispatch(ctx, userID, docID)
	return nil
}

// --- the job --------------------------------------------------------------------------------------------------------

// Job error codes (Stored.ErrorCode).
const (
	CodeAIFailed      = "ai_failed"
	CodeAIUnavailable = "ai_unavailable"
	CodeAIBudget      = "ai_budget_exhausted"
	CodeConsent       = "consent_required"
	CodeUnreadable    = "unreadable"   // a stored file could not be read back
	CodeInvalidFile   = "invalid_file" // the provider refused the file
	CodeNothingRead   = "nothing_read" // nothing usable was read
	CodeInternal      = "internal"
)

// jobError is a job failure: code is what the document shows, retry whether another attempt may help.
type jobError struct {
	code  string
	retry bool
	err   error
}

func (e *jobError) Error() string {
	if e.err == nil {
		return "extract: " + e.code
	}
	return "extract: " + e.code + ": " + e.err.Error()
}

func (e *jobError) Unwrap() error { return e.err }

func classify(err error) *jobError {
	var req *consent.RequiredError
	var je *jobError
	switch {
	case errors.As(err, &je):
		return je
	case errors.As(err, &req):
		return &jobError{code: CodeConsent, err: err}
	case errors.Is(err, context.Canceled):
		return &jobError{code: CodeInternal, retry: true, err: err}
	case errors.Is(err, ai.ErrUpstream), errors.Is(err, context.DeadlineExceeded):
		return &jobError{code: CodeAIFailed, retry: true, err: err}
	case errors.Is(err, ai.ErrUnavailable):
		return &jobError{code: CodeAIUnavailable, err: err}
	case errors.Is(err, ai.ErrBudgetExceeded), errors.Is(err, ai.ErrUserBudgetExceeded):
		return &jobError{code: CodeAIBudget, retry: true, err: err}
	case errors.Is(err, ai.ErrInvalidRequest):
		return &jobError{code: CodeInvalidFile, err: err}
	}
	return &jobError{code: CodeInternal, retry: true, err: err}
}

// claimed is a job a worker took.
type claimed struct {
	docID, userID uint64
	kind, locale  string
	token         int
}

// read is what the provider read from the document's files.
type read struct {
	fields map[string]Value
	items  []map[string]Value
}

// subject is the AI subject of a job (per-user cap and name redaction outside a request).
func (s *Service) subject(ctx context.Context, userID uint64) context.Context {
	sub := ai.Subject{UserID: userID}
	if name, err := s.q.GetExtractUserName(ctx, userID); err == nil && name.Valid && strings.TrimSpace(name.String) != "" {
		sub.Names = []string{name.String}
	}
	return ai.WithSubject(ctx, sub)
}

// extract reads the document's files (the first MaxFiles) with the kind's schema. The first value of a key wins
// (files in their position order); prescription rows are appended up to MaxItems.
func (s *Service) extract(ctx context.Context, job claimed) (read, error) {
	wctx := context.WithoutCancel(ctx)
	out := read{fields: map[string]Value{}, items: []map[string]Value{}}
	schema, ok := Schemas[job.kind]
	if !ok {
		return out, &jobError{code: CodeInternal}
	}
	if s.client == nil {
		return out, &jobError{code: CodeAIUnavailable, err: ai.ErrUnavailable}
	}
	ids, err := s.q.ListExtractFileIDs(wctx, store.ListExtractFileIDsParams{DocumentID: job.docID, UserID: job.userID})
	if err != nil {
		return out, classify(err)
	}
	if len(ids) == 0 {
		return out, &jobError{code: CodeUnreadable}
	}
	actx := s.subject(ctx, job.userID)
	for _, id := range ids[:min(len(ids), MaxFiles)] {
		// The consent is checked before every provider call: a withdrawal stops the next file (security audit L5).
		if s.consents != nil {
			if err := s.consents.Require(wctx, job.userID, consent.AIDocuments); err != nil {
				return out, classify(err)
			}
		}
		f, data, err := s.files.Open(wctx, job.userID, id)
		if err != nil {
			return out, &jobError{code: CodeUnreadable, err: err}
		}
		doc := ai.Document{Data: data, MIME: f.MIME}
		ex, err := s.client.Extract(actx, ai.FeatureDocExtract, ai.ExtractRequest{Document: doc, Schema: schema,
			Language: job.locale, Hint: hint(job.kind)})
		doc.Wipe()
		if err != nil {
			return out, classify(err)
		}
		for _, f := range ex.Fields {
			if _, seen := out.fields[f.Key]; !seen {
				out.fields[f.Key] = Value{Value: f.Value, Confidence: f.Confidence}
			}
		}
		for _, row := range ex.Items {
			if len(out.items) == MaxItems {
				break
			}
			m := map[string]Value{}
			for _, f := range row {
				m[f.Key] = Value{Value: f.Value, Confidence: f.Confidence}
			}
			out.items = append(out.items, m)
		}
	}
	if len(out.fields) == 0 && len(out.items) == 0 {
		return out, &jobError{code: CodeNothingRead}
	}
	return out, nil
}

// RefundableExtractCalls caps the refunds of extractions the provider was paid for but that gave nothing usable
// (nothing_read, invalid_file): while the user made at most this many extraction calls this month (append-only AI
// usage log) the Plus use is given back; beyond it a blank document keeps its use, so deleting and re-sending cannot
// buy unlimited free calls. Provider failures always refund (labs' rule, B-N6-06b M3).
const RefundableExtractCalls = 30

var paidCodes = map[string]bool{CodeNothingRead: true, CodeInvalidFile: true}

// refund gives the reserved use of a failed extraction back.
func (s *Service) refund(ctx context.Context, userID uint64, reservedAt time.Time, code string) {
	if s.plus == nil || reservedAt.IsZero() {
		return
	}
	ctx = context.WithoutCancel(ctx)
	now := s.now(ctx)
	if paidCodes[code] {
		calls, err := s.q.CountDocExtractCallsSince(ctx, store.CountDocExtractCallsSinceParams{
			UserID: sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // G115: ids fit int64
			Since:  nullTime(plus.PeriodStart(now).TehranMidnight()),
		})
		if err != nil || calls > RefundableExtractCalls {
			return
		}
	}
	if err := s.plus.Refund(ctx, userID, plus.DocAI, reservedAt, now); err != nil {
		s.logger.ErrorContext(ctx, "extract: plus refund failed", slog.String("error", err.Error()))
	}
}

// RefundDeleted is the healthrecord.PendingDeleteHook of the documents service (security audit L3): a document deleted
// while its extraction job waits in the queue (not running — a running attempt may already have paid the provider, and
// the job then finds the document gone and refunds nothing) gives its reserved Plus use back. Exactly one side refunds:
// the delete for a queued job, the job for a failed attempt.
func RefundDeleted(p Refunder, c clock.Clock, logger *slog.Logger) healthrecord.PendingDeleteHook {
	if c == nil {
		c = clock.Real{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, userID uint64, extracted db.NullRawJSON) {
		st := parseStored(extracted)
		if p == nil || st == nil || st.Job == nil || st.Job.State != jobQueued {
			return
		}
		at, ok := st.reservedAt()
		if !ok {
			return
		}
		now := tehranSecond(clock.FromContext(ctx, c).Now())
		if err := p.Refund(ctx, userID, plus.DocAI, at, now); err != nil {
			logger.ErrorContext(ctx, "extract: plus refund on delete failed", slog.String("error", err.Error()))
		}
	}
}

// outcome is how an attempt ends.
type outcome int

const (
	outDone outcome = iota
	outRetry
	outRelease
	outFail
)

// finish records an attempt's end on the document, only while the document still carries this claim (pending, same
// token). A failure refunds the reserved Plus use after the commit.
func (s *Service) finish(ctx context.Context, job claimed, out outcome, res read, code string, retryAt time.Time) {
	ctx = context.WithoutCancel(ctx)
	now := s.now(ctx)
	var refundAt time.Time
	err := s.inTx(ctx, func(q *store.Queries) error {
		doc, err := lockDoc(ctx, q, job.userID, job.docID)
		if err != nil {
			return err
		}
		st := parseStored(doc.Extracted)
		if doc.ReviewState != healthrecord.ReviewPending || st == nil || st.Job == nil || st.Job.Attempts != job.token {
			return nil // deleted, re-queued or finished by someone else meanwhile
		}
		switch out {
		case outRetry, outRelease:
			if out == outRelease {
				st.Job.Attempts-- // a shutdown cut the attempt short: it does not count
			}
			st.Job.State, st.Job.AvailableAt, st.Job.LockedUntil = jobQueued, retryAt.Unix(), 0
			raw, err := st.raw()
			if err != nil {
				return err
			}
			_, err = q.SetExtractJob(ctx, store.SetExtractJobParams{Extracted: raw, Now: nullTime(now), ID: job.docID, UserID: job.userID})
			return err
		case outDone:
			st.Status, st.ErrorCode, st.Fields, st.Items = StatusDone, nil, res.fields, res.items
			st.ExtractedAt = strPtr(isoTime(now))
		default:
			st.Status, st.ErrorCode = StatusFailed, strPtr(code)
			refundAt, _ = st.reservedAt()
		}
		state := healthrecord.ReviewNeedsReview
		if out == outFail {
			state = healthrecord.ReviewFailed
		}
		st.Job = nil
		raw, err := st.raw()
		if err != nil {
			return err
		}
		_, err = q.SetExtractState(ctx, store.SetExtractStateParams{ReviewState: state, Extracted: raw, Now: nullTime(now),
			ID: job.docID, UserID: job.userID})
		return err
	})
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.ErrorContext(ctx, "extract: finish job", slog.Uint64("document_id", job.docID), slog.String("error", err.Error()))
		}
		return
	}
	if out == outFail {
		s.refund(ctx, job.userID, refundAt, code)
	}
}
