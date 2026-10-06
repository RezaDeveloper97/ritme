package labs

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/plus"
)

// LabSchema is what the extractor reads from a lab sheet. Keys follow the platform's lab_panel fixture
// (internal/ai/fake_extract.go); "marker", never "name" (identity keys are refused by the Client, B-N6-05b).
var LabSchema = ai.ExtractSchema{
	Name: "lab_panel",
	Fields: []ai.FieldSpec{
		{Key: "date", Type: ai.FieldDate, Description: "the date the sample was taken or the report was issued (Gregorian)"},
		{Key: "lab_name", Type: ai.FieldString, Description: "the laboratory's name as printed (an institution, not a person)"},
	},
	Items: []ai.FieldSpec{
		{Key: "marker", Type: ai.FieldString, Description: "the test name exactly as printed (e.g. Hemoglobin, TSH, Ferritin)"},
		{Key: "value", Type: ai.FieldNumber, Description: "the numeric result"},
		{Key: "value_text", Type: ai.FieldString, Description: "a non-numeric result as printed (e.g. Negative, Trace); empty when numeric"},
		{Key: "unit", Type: ai.FieldString, Description: "the unit as printed (e.g. g/dL, ng/mL)"},
		{Key: "ref_low", Type: ai.FieldNumber, Description: "the lower bound of the printed reference range"},
		{Key: "ref_high", Type: ai.FieldNumber, Description: "the upper bound of the printed reference range"},
		{Key: "ref_text", Type: ai.FieldString, Description: "the reference range exactly as printed (e.g. 12-15.5, < 200)"},
	},
	MaxItems: MaxMarkers,
}

// jobError is a job failure: code is what the lab shows (error_code), retry whether another attempt may help.
type jobError struct {
	code  string
	retry bool
	err   error
}

func (e *jobError) Error() string { return "labs: job: " + e.code + ": " + errString(e.err) }
func (e *jobError) Unwrap() error { return e.err }

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Lab error codes (lab_reports.error_code).
const (
	CodeAIFailed      = "ai_failed"
	CodeAIUnavailable = "ai_unavailable"
	CodeAIBudget      = "ai_budget_exhausted"
	CodeUnreadable    = "unreadable"   // a stored file could not be read back
	CodeNothingRead   = "nothing_read" // the extractor found no values
	CodeConsent       = "consent_required"
	CodeInvalidFile   = "invalid_file"
	CodeInternal      = "internal"
)

// classify maps an AI / platform error to a job error.
func classify(err error) *jobError {
	var req *consent.RequiredError
	switch {
	case errors.As(err, &req):
		return &jobError{code: CodeConsent, err: err}
	case errors.Is(err, context.Canceled):
		// A shutdown (or the caller) cut the attempt short: never a final verdict on the lab (B-N6-06b, M2).
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

// subject is the AI subject of a job (the per-user cap and name redaction need it outside a request).
func (s *Service) subject(ctx context.Context, userID uint64) context.Context {
	sub := ai.Subject{UserID: userID}
	if row, err := s.q.GetLabUserContext(ctx, userID); err == nil && row.Name.Valid && strings.TrimSpace(row.Name.String) != "" {
		sub.Names = []string{row.Name.String}
	}
	return ai.WithSubject(ctx, sub)
}

// runExtract reads every page of the lab and stores the markers (needs_review). final = no retry will follow. Every
// write uses a context the shutdown cannot cancel (B-N6-06b); only the provider calls stop with ctx.
func (s *Service) runExtract(ctx context.Context, job store.LabJob, final bool) error {
	wctx := context.WithoutCancel(ctx)
	lab, err := s.q.GetLabReportByID(wctx, job.LabID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // deleted meanwhile
	}
	if err != nil {
		return classify(err)
	}
	if lab.Status != StatusQueued && lab.Status != StatusExtracting {
		return nil
	}
	fail := func(je *jobError) error {
		if (!je.retry || final) && !errors.Is(je, context.Canceled) {
			s.failLab(wctx, lab, je.code)
		}
		return je
	}
	now := s.now(wctx)
	if err := s.q.SetLabStatus(wctx, store.SetLabStatusParams{Status: StatusExtracting, Progress: 10, Now: nullTime(now), ID: lab.ID}); err != nil {
		return classify(err)
	}
	if s.client == nil {
		return fail(&jobError{code: CodeAIUnavailable, err: ai.ErrUnavailable})
	}
	if s.consents != nil {
		if err := s.consents.Require(wctx, lab.UserID, consent.AILabAnalysis); err != nil {
			return fail(classify(err))
		}
	}
	pages, err := s.q.ListLabFiles(wctx, store.ListLabFilesParams{LabID: lab.ID, UserID: lab.UserID})
	if err != nil {
		return classify(err)
	}
	if len(pages) == 0 {
		return fail(&jobError{code: CodeUnreadable, err: errors.New("no pages")})
	}
	actx := s.subject(ctx, lab.UserID)
	locale := job.Locale.String
	var merged []ai.Extraction
	for i, p := range pages {
		data, err := s.files.Get(lab.UserID, p.Path)
		if err != nil {
			return fail(&jobError{code: CodeUnreadable, err: err})
		}
		doc := ai.Document{Data: data, MIME: p.Mime}
		ex, err := s.client.Extract(actx, ai.FeatureLabAnalysis, ai.ExtractRequest{Document: doc, Schema: LabSchema, Language: locale,
			Hint: "a laboratory test sheet (" + lab.Category + "); it may be in Persian or English"})
		doc.Wipe()
		if err != nil {
			return fail(classify(err))
		}
		merged = append(merged, ex)
		progress := uint8(10 + 80*(i+1)/len(pages)) //nolint:gosec // G115: 10..90
		_ = s.q.SetLabStatus(wctx, store.SetLabStatusParams{Status: StatusExtracting, Progress: progress, Now: nullTime(s.now(wctx)), ID: lab.ID})
	}
	cat, err := s.Catalog(wctx)
	if err != nil {
		return classify(err)
	}
	today := civildate.InTehran(now)
	rows, date, labName := markersFrom(merged, today)
	if len(rows) == 0 {
		return fail(&jobError{code: CodeNothingRead, err: errors.New("no markers")})
	}
	err = s.inTx(wctx, func(q *store.Queries) error {
		if err := q.DeleteExtractedLabMarkers(wctx, lab.ID); err != nil {
			return err
		}
		for i, r := range rows {
			if err := insertMarker(wctx, q, lab.ID, lab.UserID, r.in, cat, MarkerExtracted,
				sql.NullString{String: strconv.FormatFloat(r.confidence, 'f', 3, 64), Valid: true}, uint16(i+1), now); err != nil { //nolint:gosec // G115
				return err
			}
		}
		return q.SetLabExtracted(wctx, store.SetLabExtractedParams{TakenOn: date, LabName: nullString(labName), Now: nullTime(now), ID: lab.ID})
	})
	if err != nil {
		return classify(err)
	}
	return nil
}

// RefundableExtractCalls caps the refunds of extractions the provider was paid for but that gave nothing usable
// (nothing_read, invalid_file): while the user made at most this many extraction calls (pages) this month — counted
// from the append-only AI usage log — the Plus use is given back; beyond it a blank upload keeps its use (B-N6-06b,
// M3), so deleting failed labs cannot buy unlimited free extractions. Provider failures always refund.
const RefundableExtractCalls = 30

// paidCodes are failures after the provider was paid.
var paidCodes = map[string]bool{CodeNothingRead: true, CodeInvalidFile: true}

// failLab marks the lab failed and gives the reserved Plus use back (at most once: ClearLabQuota).
func (s *Service) failLab(ctx context.Context, lab store.LabReport, code string) {
	ctx = context.WithoutCancel(ctx)
	now := s.now(ctx)
	if err := s.q.SetLabStatus(ctx, store.SetLabStatusParams{Status: StatusFailed, Progress: 0,
		ErrorCode: sql.NullString{String: code, Valid: true}, Now: nullTime(now), ID: lab.ID}); err != nil {
		s.logger.ErrorContext(ctx, "labs: mark failed", slog.String("error", err.Error()))
	}
	if !lab.QuotaAt.Valid || s.plus == nil {
		return
	}
	if paidCodes[code] {
		calls, err := s.q.CountLabExtractCallsSince(ctx, store.CountLabExtractCallsSinceParams{
			UserID: sql.NullInt64{Int64: int64(lab.UserID), Valid: true}, //nolint:gosec // G115: ids fit int64
			Since:  nullTime(plus.PeriodStart(tehranSecond(now)).TehranMidnight()),
		})
		if err != nil || calls > RefundableExtractCalls {
			return
		}
	}
	n, err := s.q.ClearLabQuota(ctx, lab.ID)
	if err != nil || n == 0 {
		return
	}
	if err := s.plus.Refund(ctx, lab.UserID, plus.LabAI, lab.QuotaAt.Time, tehranSecond(now)); err != nil {
		s.logger.ErrorContext(ctx, "labs: plus refund failed", slog.String("error", err.Error()))
	}
}

// finalize gives a job's lab its final state when no attempt will (a panic, an exhausted job left by a dead worker,
// a busy lab without a live job): a pending extraction fails (refund), a pending interpretation gets the rules
// summary.
func (s *Service) finalize(ctx context.Context, job store.LabJob, code string) {
	ctx = context.WithoutCancel(ctx)
	lab, err := s.q.GetLabReportByID(ctx, job.LabID)
	if err != nil {
		return
	}
	switch {
	case job.Kind == JobExtract && (lab.Status == StatusQueued || lab.Status == StatusExtracting):
		s.failLab(ctx, lab, code)
	case job.Kind == JobInterpret && lab.Status == StatusInterpreting:
		l := loc{Locale: job.Locale.String, Default: job.Locale.String}
		if err := s.interpretRules(ctx, lab.UserID, lab.ID, l, s.now(ctx)); err != nil {
			s.logger.ErrorContext(ctx, "labs: finalize interpretation", slog.String("error", err.Error()))
		}
	}
}

// extractedRow is one marker read from the sheet.
type extractedRow struct {
	in         MarkerInput
	confidence float64
}

// markersFrom merges the pages' extractions: one row per printed name (first page wins), the sheet date (not in
// the future, not before 2000) and the lab name.
func markersFrom(exs []ai.Extraction, today civildate.Date) ([]extractedRow, civildate.NullDate, string) {
	var (
		out     []extractedRow
		seen    = map[string]bool{}
		date    civildate.NullDate
		labName string
	)
	for _, ex := range exs {
		for _, f := range ex.Fields {
			switch f.Key {
			case "date":
				if s, ok := f.Value.(string); ok && !date.Valid {
					if d, err := civildate.Parse(s); err == nil && !d.After(today) && d.Year >= 2000 {
						date = civildate.NullDate{Date: d, Valid: true}
					}
				}
			case "lab_name":
				if s, ok := f.Value.(string); ok && labName == "" {
					labName = truncate(s, MaxNameLen)
				}
			}
		}
		for _, item := range ex.Items {
			r, ok := rowFrom(item)
			if !ok || len(out) >= MaxMarkers {
				continue
			}
			key := normalizeName(r.in.Name)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, r)
		}
	}
	return out, date, labName
}

func rowFrom(item []ai.ExtractedField) (extractedRow, bool) {
	r := extractedRow{confidence: 1}
	num := func(v any) *float64 {
		f, ok := v.(float64)
		if !ok || math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > MaxValue {
			return nil
		}
		return &f
	}
	for _, f := range item {
		switch f.Key {
		case "marker":
			s, _ := f.Value.(string)
			r.in.Name = truncate(s, MaxNameLen)
		case "value":
			r.in.Value = num(f.Value)
		case "value_text":
			s, _ := f.Value.(string)
			r.in.ValueText = truncate(s, MaxValueText)
		case "unit":
			s, _ := f.Value.(string)
			r.in.Unit = truncate(s, MaxUnitLen)
		case "ref_low":
			r.in.RefLow = num(f.Value)
		case "ref_high":
			r.in.RefHigh = num(f.Value)
		case "ref_text":
			s, _ := f.Value.(string)
			r.in.RefText = truncate(s, MaxRefTextLen)
		default:
			continue
		}
		if f.Key == "marker" || f.Key == "value" || f.Key == "value_text" {
			r.confidence = math.Min(r.confidence, f.Confidence)
		}
	}
	if r.in.Value != nil {
		r.in.ValueText = ""
	}
	if r.in.RefLow != nil && r.in.RefHigh != nil && *r.in.RefLow > *r.in.RefHigh {
		r.in.RefLow, r.in.RefHigh = nil, nil // nonsense bounds: keep only the printed text
	}
	return r, r.in.Name != "" && (r.in.Value != nil || r.in.ValueText != "")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > n {
		s = strings.TrimSpace(string([]rune(s)[:n]))
	}
	return s
}

// runInterpret writes the interpretation of a verified lab. The AI summary needs: an uploaded lab within its
// interpretation allowance, a client, the consent in force and the budgets; otherwise — or when the provider fails
// for good — the rules summary is stored. Never leaves the lab in interpreting.
func (s *Service) runInterpret(ctx context.Context, job store.LabJob, final bool) error {
	wctx := context.WithoutCancel(ctx) // writes survive a shutdown; only the provider call stops with ctx
	lab, err := s.q.GetLabReportByID(wctx, job.LabID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return classify(err)
	}
	if lab.Status != StatusInterpreting {
		return nil
	}
	now := s.now(wctx)
	_ = s.q.SetLabStatus(wctx, store.SetLabStatusParams{Status: StatusInterpreting, Progress: 50, Now: nullTime(now), ID: lab.ID})
	l := loc{Locale: job.Locale.String, Default: job.Locale.String}
	useAI := lab.Source == SourceUpload && s.client != nil
	if useAI && s.consents != nil && s.consents.Require(wctx, lab.UserID, consent.AILabAnalysis) != nil {
		useAI = false
	}
	var retryErr error
	err = s.storeInterpretation(wctx, lab.UserID, lab.ID, l, now, func(evals []Evaluated, uc UserContext) (string, string) {
		if !useAI || len(evals) == 0 {
			return "", SummaryRules
		}
		flags := map[uint64]RedFlag{}
		for _, e := range evals {
			if f, ok := redFlag(e, e.Row.Name); ok {
				flags[e.Row.ID] = f
			}
		}
		text, err := aiSummary(s.subject(ctx, lab.UserID), s.client, buildPrompt(evals, uc, l.Locale, flags), uc.Names)
		if err != nil {
			if je := classify(err); je.retry && !final {
				retryErr = je
			}
			return "", SummaryRules
		}
		return text, SummaryAI
	})
	if retryErr != nil {
		// Undo nothing: the lab goes back to interpreting so the retry writes the AI summary over the rules one.
		_ = s.q.SetLabStatus(wctx, store.SetLabStatusParams{Status: StatusInterpreting, Progress: 50, Now: nullTime(now), ID: lab.ID})
		return retryErr
	}
	if err != nil {
		return classify(err)
	}
	return nil
}
