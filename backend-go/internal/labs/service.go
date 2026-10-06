package labs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/labs/files"
	"github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
)

// Service is the lab analysis domain: CRUD, the extraction / interpretation jobs (run by Runner) and the reads.
type Service struct {
	db       *sql.DB
	q        *store.Queries
	files    *files.Box
	client   *ai.Client
	consents *consent.Service
	plus     *plus.Service
	cat      catalogLoader
	logger   *slog.Logger
	clock    clock.Clock
	runner   *Runner
}

// Options wires a Service.
type Options struct {
	DB       *sql.DB
	Files    *files.Box
	AI       *ai.Client       // nil = AI unavailable: uploads are refused by the gate, summaries fall back to rules
	Consents *consent.Service // the AI consent is re-checked when a job runs
	Plus     *plus.Service    // refunds the reserved use of a failed extraction
	Catalog  *catalog.Reader  // the lab_markers group (nil = empty catalog)
	Logger   *slog.Logger
	Clock    clock.Clock
}

// NewService wires the service.
func NewService(o Options) *Service {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Clock == nil {
		o.Clock = clock.Real{}
	}
	return &Service{db: o.DB, q: store.New(o.DB), files: o.Files, client: o.AI, consents: o.Consents, plus: o.Plus,
		cat: catalogLoader{reader: o.Catalog, logger: o.Logger}, logger: o.Logger, clock: o.Clock}
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: tehranSecond(t), Valid: true} }

func nullString(s string) sql.NullString {
	if s = strings.TrimSpace(s); s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullDecimal(f *float64) sql.NullString {
	if f == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: decimalString(*f), Valid: true}
}

// Catalog is the marker catalog.
func (s *Service) Catalog(ctx context.Context) (*Catalog, error) { return s.cat.load(ctx) }

// busy reports whether the lab is being processed (no edits then).
func busy(status string) bool {
	return status == StatusQueued || status == StatusExtracting || status == StatusInterpreting
}

// Get is the user's lab (ErrNotFound for another user's or a missing one).
func (s *Service) Get(ctx context.Context, userID, id uint64) (store.LabReport, error) {
	lab, err := s.q.GetLabReport(ctx, store.GetLabReportParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return lab, ErrNotFound
	}
	if err != nil {
		return lab, fmt.Errorf("labs: get: %w", err)
	}
	return lab, nil
}

// List is the user's labs, newest first.
func (s *Service) List(ctx context.Context, userID uint64) ([]store.LabReport, error) {
	rows, err := s.q.ListLabReports(ctx, store.ListLabReportsParams{UserID: userID, Limit: MaxLabsListed})
	if err != nil {
		return nil, fmt.Errorf("labs: list: %w", err)
	}
	return rows, nil
}

// Markers are the lab's markers in sheet order.
func (s *Service) Markers(ctx context.Context, userID, labID uint64) ([]store.LabMarker, error) {
	rows, err := s.q.ListLabMarkers(ctx, store.ListLabMarkersParams{LabID: labID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("labs: markers: %w", err)
	}
	return rows, nil
}

// Files are the lab's stored pages.
func (s *Service) Files(ctx context.Context, userID, labID uint64) ([]store.LabFile, error) {
	rows, err := s.q.ListLabFiles(ctx, store.ListLabFilesParams{LabID: labID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("labs: files: %w", err)
	}
	return rows, nil
}

// UserMarkers are every marker of the user's labs with the lab's date (trends, history counts).
func (s *Service) UserMarkers(ctx context.Context, userID uint64) ([]store.ListUserLabMarkersRow, error) {
	rows, err := s.q.ListUserLabMarkers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("labs: user markers: %w", err)
	}
	return rows, nil
}

// Meta is the user-editable part of a lab (upload form / PUT /labs/{id}).
type Meta struct {
	Category string
	Title    string
	TakenOn  civildate.NullDate
	Fasting  sql.NullBool
}

// Page is one uploaded file: sniffed MIME, re-encoded when an image.
type Page struct {
	Data []byte
	MIME string
}

// FilesDisabled reports whether uploads must be refused (production without LAB_FILE_KEY).
func (s *Service) FilesDisabled() bool { return s.files.Disabled() }

// CreateUpload stores the pages encrypted, creates the lab (queued) with its extract job and starts it. quotaAt is
// when the Plus use was reserved (nil = none). Any failure removes the files already written.
func (s *Service) CreateUpload(ctx context.Context, userID uint64, m Meta, pages []Page, quotaAt *time.Time, locale string, now time.Time) (uint64, error) {
	if s.files.Disabled() {
		return 0, ErrStorage
	}
	n, err := s.q.CountLabExtractCallsSince(ctx, store.CountLabExtractCallsSinceParams{
		UserID: sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // G115: ids fit int64
		Since:  nullTime(now.Add(-24 * time.Hour)),
	})
	if err != nil {
		return 0, fmt.Errorf("labs: daily count: %w", err)
	}
	if int(n)+len(pages) > PagesPerDay {
		return 0, ErrDailyLimit
	}
	var written []string
	cleanup := func() {
		for _, rel := range written {
			if err := s.files.Remove(userID, rel); err != nil {
				s.logger.ErrorContext(ctx, "labs: cleanup of an upload failed", slog.String("error", err.Error()))
			}
		}
	}
	for _, p := range pages {
		rel, err := s.files.Put(userID, p.Data)
		if err != nil {
			cleanup()
			if errors.Is(err, files.ErrDisabled) {
				return 0, ErrStorage
			}
			return 0, fmt.Errorf("labs: store page: %w", err)
		}
		written = append(written, rel)
	}
	var quota sql.NullTime
	if quotaAt != nil {
		quota = sql.NullTime{Time: *quotaAt, Valid: true}
	}
	var labID, jobID uint64
	err = s.inTx(ctx, func(q *store.Queries) error {
		id, err := q.CreateLabReport(ctx, store.CreateLabReportParams{UserID: userID, Source: SourceUpload, Category: m.Category,
			Title: nullString(m.Title), TakenOn: m.TakenOn, Fasting: m.Fasting, Status: StatusQueued, QuotaAt: quota, Now: nullTime(now)})
		if err != nil {
			return err
		}
		labID = uint64(id) //nolint:gosec // G115: auto-increment ids are positive
		for i, p := range pages {
			if _, err := q.CreateLabFile(ctx, store.CreateLabFileParams{LabID: labID, UserID: userID, Page: uint8(i + 1), //nolint:gosec // G115: ≤ MaxFiles
				Mime: p.MIME, SizeBytes: uint32(len(p.Data)), Path: written[i], Now: nullTime(now)}); err != nil { //nolint:gosec // G115: ≤ 10 MB
				return err
			}
		}
		jid, err := q.CreateLabJob(ctx, store.CreateLabJobParams{LabID: labID, UserID: userID, Kind: JobExtract,
			Locale: nullString(locale), AvailableAt: tehranSecond(now), Now: nullTime(now)})
		jobID = uint64(jid) //nolint:gosec // G115
		return err
	})
	if err != nil {
		cleanup()
		return 0, fmt.Errorf("labs: create upload: %w", err)
	}
	s.runner.dispatch(ctx, jobID)
	return labID, nil
}

// MarkerInput is one typed or edited marker (validated by the handler).
type MarkerInput struct {
	Name      string
	Value     *float64
	ValueText string
	Unit      string
	RefLow    *float64
	RefHigh   *float64
	RefText   string
}

func (in MarkerInput) code(cat *Catalog) sql.NullString {
	if mk, ok := cat.Match(in.Name); ok {
		return sql.NullString{String: mk.Code, Valid: true}
	}
	return sql.NullString{}
}

// refText is the printed range, or one built from the bounds.
func (in MarkerInput) refText() string {
	if t := strings.TrimSpace(in.RefText); t != "" {
		return t
	}
	switch {
	case in.RefLow != nil && in.RefHigh != nil:
		return fmtNum(in.RefLow) + "–" + fmtNum(in.RefHigh)
	case in.RefLow != nil:
		return "≥ " + fmtNum(in.RefLow)
	case in.RefHigh != nil:
		return "≤ " + fmtNum(in.RefHigh)
	}
	return ""
}

// CreateManual saves a typed-in lab: verified at once, interpreted by the rules (no AI, no Plus use).
func (s *Service) CreateManual(ctx context.Context, userID uint64, m Meta, markers []MarkerInput, locale, def string, now time.Time) (uint64, error) {
	cat, err := s.Catalog(ctx)
	if err != nil {
		return 0, err
	}
	var labID uint64
	err = s.inTx(ctx, func(q *store.Queries) error {
		id, err := q.CreateLabReport(ctx, store.CreateLabReportParams{UserID: userID, Source: SourceManual, Category: m.Category,
			Title: nullString(m.Title), TakenOn: m.TakenOn, Fasting: m.Fasting, Status: StatusInterpreting,
			VerifiedAt: nullTime(now), Now: nullTime(now)})
		if err != nil {
			return err
		}
		labID = uint64(id) //nolint:gosec // G115
		for i, in := range markers {
			if err := insertMarker(ctx, q, labID, userID, in, cat, MarkerManual, sql.NullString{}, uint16(i+1), now); err != nil { //nolint:gosec // G115: ≤ MaxMarkers
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("labs: create manual: %w", err)
	}
	if err := s.interpretRules(ctx, userID, labID, loc{Locale: locale, Default: def}, now); err != nil {
		return 0, err
	}
	return labID, nil
}

func insertMarker(ctx context.Context, q *store.Queries, labID, userID uint64, in MarkerInput, cat *Catalog, source string,
	confidence sql.NullString, sort uint16, now time.Time,
) error {
	_, err := q.CreateLabMarker(ctx, store.CreateLabMarkerParams{LabID: labID, UserID: userID, Code: in.code(cat),
		Name: strings.TrimSpace(in.Name), Value: nullDecimal(in.Value), ValueText: nullString(in.ValueText), Unit: nullString(in.Unit),
		RefLow: nullDecimal(in.RefLow), RefHigh: nullDecimal(in.RefHigh), RefText: nullString(in.refText()),
		Confidence: confidence, Source: source, SortOrder: sort, Now: nullTime(now)})
	return err
}

// UpdateMeta changes the lab's type, title, date and fasting flag.
func (s *Service) UpdateMeta(ctx context.Context, userID, id uint64, m Meta, now time.Time) error {
	n, err := s.q.UpdateLabMeta(ctx, store.UpdateLabMetaParams{Category: m.Category, Title: nullString(m.Title), TakenOn: m.TakenOn,
		Fasting: m.Fasting, Now: nullTime(now), ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("labs: update: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes the lab, its markers, jobs and files (rows first, then the files).
func (s *Service) Delete(ctx context.Context, userID, id uint64) error {
	fs, err := s.Files(ctx, userID, id)
	if err != nil {
		return err
	}
	n, err := s.q.DeleteLabReport(ctx, store.DeleteLabReportParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("labs: delete: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	s.removeFiles(ctx, userID, fs)
	return nil
}

func (s *Service) removeFiles(ctx context.Context, userID uint64, fs []store.LabFile) {
	for _, f := range fs {
		if err := s.files.Remove(userID, f.Path); err != nil {
			s.logger.ErrorContext(ctx, "labs: file removal failed", slog.Uint64("file_id", f.ID), slog.String("error", err.Error()))
		}
	}
}

// File is a decrypted page of the user's lab.
func (s *Service) File(ctx context.Context, userID, labID, fileID uint64) (store.LabFile, []byte, error) {
	f, err := s.q.GetLabFile(ctx, store.GetLabFileParams{ID: fileID, LabID: labID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return f, nil, ErrFileNotFound
	}
	if err != nil {
		return f, nil, fmt.Errorf("labs: file: %w", err)
	}
	data, err := s.files.Get(userID, f.Path)
	switch {
	case errors.Is(err, files.ErrDisabled):
		return f, nil, ErrStorage
	case err != nil:
		return f, nil, ErrFileNotFound
	}
	return f, data, nil
}

// DeleteFile removes one page (the markers read from it stay).
func (s *Service) DeleteFile(ctx context.Context, userID, labID, fileID uint64) error {
	lab, err := s.Get(ctx, userID, labID)
	if err != nil {
		return err
	}
	if busy(lab.Status) {
		return ErrBusy
	}
	f, err := s.q.GetLabFile(ctx, store.GetLabFileParams{ID: fileID, LabID: labID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFileNotFound
	}
	if err != nil {
		return fmt.Errorf("labs: file: %w", err)
	}
	if _, err := s.q.DeleteLabFile(ctx, store.DeleteLabFileParams{ID: fileID, LabID: labID, UserID: userID}); err != nil {
		return fmt.Errorf("labs: delete file: %w", err)
	}
	s.removeFiles(ctx, userID, []store.LabFile{f})
	return nil
}

// editable loads a lab the user may change the markers of.
func (s *Service) editable(ctx context.Context, userID, labID uint64) (store.LabReport, error) {
	lab, err := s.Get(ctx, userID, labID)
	if err != nil {
		return lab, err
	}
	if busy(lab.Status) || lab.Status == StatusFailed {
		return lab, ErrBusy
	}
	return lab, nil
}

// afterEdit: an uploaded lab goes back to review (its summary no longer matches); a manual one is re-interpreted by
// the rules at once.
func (s *Service) afterEdit(ctx context.Context, lab store.LabReport, l loc, now time.Time) error {
	if lab.Source == SourceManual {
		return s.interpretRules(ctx, lab.UserID, lab.ID, l, now)
	}
	if lab.Status == StatusReady {
		return s.q.SetLabStatus(ctx, store.SetLabStatusParams{Status: StatusNeedsReview, Progress: 100, Now: nullTime(now), ID: lab.ID})
	}
	return s.q.TouchLab(ctx, store.TouchLabParams{Now: nullTime(now), ID: lab.ID, UserID: lab.UserID})
}

// AddMarker adds a value the extractor missed («افزودن شاخص جاافتاده») or a typed one.
func (s *Service) AddMarker(ctx context.Context, userID, labID uint64, in MarkerInput, l loc, now time.Time) (uint64, error) {
	lab, err := s.editable(ctx, userID, labID)
	if err != nil {
		return 0, err
	}
	count, err := s.q.CountLabMarkers(ctx, store.CountLabMarkersParams{LabID: labID, UserID: userID})
	if err != nil {
		return 0, fmt.Errorf("labs: count markers: %w", err)
	}
	if count >= MaxMarkers {
		return 0, ErrTooManyMarkers
	}
	cat, err := s.Catalog(ctx)
	if err != nil {
		return 0, err
	}
	sort, err := s.q.NextLabMarkerSort(ctx, labID)
	if err != nil {
		return 0, fmt.Errorf("labs: marker sort: %w", err)
	}
	var id int64
	err = s.inTx(ctx, func(q *store.Queries) error {
		id, err = q.CreateLabMarker(ctx, store.CreateLabMarkerParams{LabID: labID, UserID: userID, Code: in.code(cat),
			Name: strings.TrimSpace(in.Name), Value: nullDecimal(in.Value), ValueText: nullString(in.ValueText), Unit: nullString(in.Unit),
			RefLow: nullDecimal(in.RefLow), RefHigh: nullDecimal(in.RefHigh), RefText: nullString(in.refText()),
			Source: MarkerManual, SortOrder: uint16(min(sort, math.MaxUint16)), Now: nullTime(now)}) //nolint:gosec // G115: bounded
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("labs: add marker: %w", err)
	}
	if err := s.afterEdit(ctx, lab, l, now); err != nil {
		return 0, err
	}
	return uint64(id), nil //nolint:gosec // G115
}

// UpdateMarker corrects a value (source becomes edited unless it was typed in).
func (s *Service) UpdateMarker(ctx context.Context, userID, labID, markerID uint64, in MarkerInput, l loc, now time.Time) error {
	lab, err := s.editable(ctx, userID, labID)
	if err != nil {
		return err
	}
	cur, err := s.q.GetLabMarker(ctx, store.GetLabMarkerParams{ID: markerID, LabID: labID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMarkerNotFound
	}
	if err != nil {
		return fmt.Errorf("labs: marker: %w", err)
	}
	cat, err := s.Catalog(ctx)
	if err != nil {
		return err
	}
	source := MarkerEdited
	if cur.Source == MarkerManual {
		source = MarkerManual
	}
	n, err := s.q.UpdateLabMarker(ctx, store.UpdateLabMarkerParams{Code: in.code(cat), Name: strings.TrimSpace(in.Name),
		Value: nullDecimal(in.Value), ValueText: nullString(in.ValueText), Unit: nullString(in.Unit), RefLow: nullDecimal(in.RefLow),
		RefHigh: nullDecimal(in.RefHigh), RefText: nullString(in.refText()), Source: source, Now: nullTime(now),
		ID: markerID, LabID: labID, UserID: userID})
	if err != nil {
		return fmt.Errorf("labs: update marker: %w", err)
	}
	if n == 0 { // unchanged row, or the lab started processing meanwhile (checked in SQL)
		if cur, err := s.Get(ctx, userID, labID); err != nil || busy(cur.Status) || cur.Status == StatusFailed {
			return ErrBusy
		}
	}
	return s.afterEdit(ctx, lab, l, now)
}

// DeleteMarker removes a misread row.
func (s *Service) DeleteMarker(ctx context.Context, userID, labID, markerID uint64, l loc, now time.Time) error {
	lab, err := s.editable(ctx, userID, labID)
	if err != nil {
		return err
	}
	n, err := s.q.DeleteLabMarker(ctx, store.DeleteLabMarkerParams{ID: markerID, LabID: labID, UserID: userID})
	if err != nil {
		return fmt.Errorf("labs: delete marker: %w", err)
	}
	if n == 0 {
		if _, gerr := s.q.GetLabMarker(ctx, store.GetLabMarkerParams{ID: markerID, LabID: labID, UserID: userID}); gerr == nil {
			return ErrBusy // the lab started processing meanwhile (checked in SQL)
		}
		return ErrMarkerNotFound
	}
	return s.afterEdit(ctx, lab, l, now)
}

// Verify confirms the values (nbl_Lab_Verify «تأیید و ادامه») and starts the interpretation.
func (s *Service) Verify(ctx context.Context, userID, labID uint64, locale string, now time.Time) error {
	lab, err := s.Get(ctx, userID, labID)
	if err != nil {
		return err
	}
	if lab.Source != SourceUpload || (lab.Status != StatusNeedsReview && lab.Status != StatusReady) {
		return ErrNotReviewable
	}
	count, err := s.q.CountLabMarkers(ctx, store.CountLabMarkersParams{LabID: labID, UserID: userID})
	if err != nil {
		return fmt.Errorf("labs: count markers: %w", err)
	}
	if count == 0 {
		return ErrNoMarkers
	}
	var jobID uint64
	err = s.inTx(ctx, func(q *store.Queries) error {
		n, err := q.MarkLabVerified(ctx, store.MarkLabVerifiedParams{Now: nullTime(now), Status: StatusInterpreting, ID: labID,
			UserID: userID, MaxCount: MaxInterpretations})
		if err != nil {
			return err
		}
		if n == 0 {
			if lab.InterpretCount >= MaxInterpretations {
				return ErrInterpretLimit
			}
			return ErrNotReviewable
		}
		jid, err := q.CreateLabJob(ctx, store.CreateLabJobParams{LabID: labID, UserID: userID, Kind: JobInterpret,
			Locale: nullString(locale), AvailableAt: tehranSecond(now), Now: nullTime(now)})
		jobID = uint64(jid) //nolint:gosec // G115
		return err
	})
	if errors.Is(err, ErrNotReviewable) || errors.Is(err, ErrInterpretLimit) {
		return err
	}
	if err != nil {
		return fmt.Errorf("labs: verify: %w", err)
	}
	s.runner.dispatch(ctx, jobID)
	return nil
}

// Feedback stores «این تحلیل مفید بود؟» (helpful = 👍).
func (s *Service) Feedback(ctx context.Context, userID, labID uint64, helpful bool, note string, now time.Time) error {
	v := int16(-1)
	if helpful {
		v = 1
	}
	n, err := s.q.SetLabFeedback(ctx, store.SetLabFeedbackParams{Feedback: sql.NullInt16{Int16: v, Valid: true},
		FeedbackNote: nullString(note), Now: nullTime(now), ID: labID, UserID: userID})
	if err != nil {
		return fmt.Errorf("labs: feedback: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// evaluateAll evaluates a lab's markers against the catalog.
func evaluateAll(rows []store.LabMarker, cat *Catalog) []Evaluated {
	out := make([]Evaluated, 0, len(rows))
	for _, r := range rows {
		out = append(out, evaluate(r, cat))
	}
	return out
}

// interpretRules stores a rules-only interpretation (manual labs, AI unavailable or refused).
func (s *Service) interpretRules(ctx context.Context, userID, labID uint64, l loc, now time.Time) error {
	return s.storeInterpretation(ctx, userID, labID, l, now, func([]Evaluated, UserContext) (string, string) { return "", SummaryRules })
}

// storeInterpretation evaluates the lab, asks summarize for a summary (empty → the rules summary) and stores it.
func (s *Service) storeInterpretation(ctx context.Context, userID, labID uint64, l loc, now time.Time,
	summarize func([]Evaluated, UserContext) (string, string),
) error {
	cat, err := s.Catalog(ctx)
	if err != nil {
		return err
	}
	rows, err := s.Markers(ctx, userID, labID)
	if err != nil {
		return err
	}
	evals := evaluateAll(rows, cat)
	uc, err := s.userContext(ctx, userID, civildate.InTehran(now))
	if err != nil {
		return err
	}
	in := Interpretation{Locale: l.Locale, GeneratedAt: tehranSecond(now)}
	in.Summary, in.Source = summarize(evals, uc)
	if in.Summary == "" {
		in.Summary, in.Source = rulesSummary(evals, l), SummaryRules
	}
	in.Context.Mode, in.Context.Phase, in.Context.Medications = string(uc.Mode), uc.Phase, len(uc.Medications)
	raw, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("labs: interpretation: %w", err)
	}
	if err := s.q.SetLabInterpretation(ctx, store.SetLabInterpretationParams{Interpretation: db.NullRawJSON{V: raw, Valid: true},
		Now: nullTime(now), ID: labID}); err != nil {
		return fmt.Errorf("labs: store interpretation: %w", err)
	}
	return nil
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// now is the clock of ctx (a request's pinned test clock travels with the job in sync mode), else the service's.
func (s *Service) now(ctx context.Context) time.Time { return clock.FromContext(ctx, s.clock).Now() }
