package extract

import (
	"embed"
	"errors"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The extraction copy is data: lang/<code>/extract.json (English fallback).
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the extraction line for key ("messages.extraction_started") in locale.
func T(key, locale string) string { return translator().Trans("extract."+key, nil, locale) }

func tp(key string, params map[string]string, locale string) string {
	return translator().Trans("extract."+key, params, locale)
}

func attributes(locale string) []string {
	v, ok := translator().Get("extract.attributes", locale)
	m, isMap := v.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if s, ok := m.Get(k); ok {
			if str, ok := s.(string); ok {
				kv = append(kv, k, str)
			}
		}
	}
	return kv
}

// Error codes.
const (
	CodeDocumentNotFound  = healthrecord.ErrorCodeDocumentNotFound
	CodeExtractionRunning = "extraction_running"
	CodeDocumentConfirmed = "document_confirmed"
	CodeNotInReview       = "not_in_review"
	CodeDatingUnavailable = "dating_unavailable"
)

// Handlers are the /api/v1/health-record/documents/{id}/{extract,review,dating} actions (CB-REC-02). Mount behind the
// locale middleware and auth RequireUser; POST …/extract also behind the AI gate (access.Guard.Chain of
// ai.FeatureDocExtract: throttles → plus.doc_ai → consent ai_documents → budgets → one reserved use). Owner-only.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return tehranSecond(clock.FromContext(c, h.clock).Now())
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func docID(c fiber.Ctx) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func conflict(code, locale string) error {
	return httpx.Fail(fiber.StatusConflict, T("messages."+code, locale), "error_code", code)
}

func (h *Handlers) fail(err error, locale string) error {
	var du *ErrDatingUnavailable
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.document_not_found", locale), "error_code", CodeDocumentNotFound)
	case errors.Is(err, ErrNoFiles):
		return failValidation(locale, jsonx.Obj("file_ids", []string{T("validation.no_files", locale)}))
	case errors.Is(err, ErrRunning):
		return conflict(CodeExtractionRunning, locale)
	case errors.Is(err, ErrConfirmed):
		return conflict(CodeDocumentConfirmed, locale)
	case errors.Is(err, ErrNotInReview):
		return conflict(CodeNotInReview, locale)
	case errors.As(err, &du):
		var reason any
		if du.Offer.Reason != "" {
			reason = du.Offer.Reason
		}
		return httpx.Fail(fiber.StatusConflict, T("messages.dating_unavailable", locale), "error_code", CodeDatingUnavailable,
			"state", du.Offer.State, "reason", reason)
	}
	return err
}

func (h *Handlers) detail(c fiber.Ctx, userID, id uint64, now time.Time) (*jsonx.OrderedMap, error) {
	doc, err := h.svc.docs.Get(c, userID, id)
	if err != nil {
		return nil, err
	}
	return h.svc.docs.Detail(c, userID, doc, now)
}

// Extract is POST /health-record/documents/{id}/extract (202 with the document; review_state pending, or already
// needs_review / failed when the queue runs inline). The AI gate before it answered 402 / 403 / 429 / 503 already.
func (h *Handlers) Extract(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := docID(c)
	if !ok {
		return h.fail(ErrNotFound, locale)
	}
	var reserved time.Time
	if access.ReservationFrom(c.Context()) != nil {
		reserved = now // the gate reserved the use at this request's time (same clock, same second)
	}
	if err := h.svc.Start(c.Context(), userID, id, locale, reserved, now); err != nil {
		return h.fail(err, locale)
	}
	// From here on the queued job owns the reserved Plus use (it refunds a failed extraction itself): this request must
	// not fail any more, or the gate would refund the same use a second time (security audit L2). A failed reload
	// answers 202 with the id only.
	msg := T("messages.extraction_started", locale)
	body, err := h.detail(c, userID, id, h.now(c))
	if err != nil {
		return httpx.JSON(c, fiber.StatusAccepted, httpx.Envelope(jsonx.Obj("id", id), msg))
	}
	return httpx.JSON(c, fiber.StatusAccepted, httpx.Envelope(body, msg))
}

// reviewRules are the rules of POST …/review for kind's schema.
func reviewRules(schema ai.ExtractSchema) validation.Rules {
	rules := validation.Rules{
		validation.F("fields", "nullable", "array"),
		validation.F("items", "nullable", "array", "max:"+strconv.Itoa(MaxItems)),
		validation.F("items.*", "required", "array"),
	}
	text := "max:" + strconv.Itoa(MaxText)
	for _, f := range schema.Fields {
		key := "fields." + f.Key
		switch f.Type {
		case ai.FieldString:
			rules = append(rules, validation.F(key, "nullable", "string", text))
		case ai.FieldNumber:
			r := []any{"nullable", "numeric", "min:0", "max:1000"}
			switch f.Key {
			case "ga_weeks":
				r = []any{"nullable", "integer", "min:0", "max:45"}
			case "ga_days":
				r = []any{"nullable", "integer", "min:0", "max:6"}
			}
			rules = append(rules, validation.F(key, r...))
		case ai.FieldDate:
			r := []any{"nullable", "date", "after_or_equal:1950-01-01"}
			if f.Key == "date" || f.Key == "ended_on" {
				r = append(r, "before_or_equal:today")
			}
			rules = append(rules, validation.F(key, r...))
		case ai.FieldEnum:
			rules = append(rules, validation.F(key, "nullable", "string", validation.In(f.Values...)))
		}
	}
	for _, f := range schema.Items {
		rules = append(rules, validation.F("items.*."+f.Key, "nullable", "string", text))
	}
	return rules
}

func specOf(list []ai.FieldSpec, key string) (ai.FieldSpec, bool) {
	i := slices.IndexFunc(list, func(f ai.FieldSpec) bool { return f.Key == key })
	if i < 0 {
		return ai.FieldSpec{}, false
	}
	return list[i], true
}

// typed converts a validated value to what the extraction stores for its field type.
func typed(spec ai.FieldSpec, v any, now time.Time) any {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(phpval.ToString(v))
	switch spec.Type {
	case ai.FieldNumber:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		return f
	case ai.FieldDate:
		t, err := civildate.ParseLenient(s, now, civildate.Tehran)
		if err != nil {
			return nil
		}
		return civildate.InTehran(t).String()
	}
	if s == "" {
		return nil
	}
	return s
}

// Review is POST /health-record/documents/{id}/review {fields?: {key: value|null}, items?: [{…}]}: the user checked
// (and maybe corrected) the values read; the document becomes confirmed. Returns {document, dating}.
func (h *Handlers) Review(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := docID(c)
	if !ok {
		return h.fail(ErrNotFound, locale)
	}
	doc, err := h.svc.docs.Get(c, userID, id)
	if err != nil {
		return h.fail(err, locale)
	}
	schema := Schemas[doc.Kind]
	body := validation.Input(c)
	data := phpval.NewMap()
	for _, k := range []string{"fields", "items"} {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, reviewRules(schema), validation.Now(now),
		validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	in := ReviewInput{Fields: map[string]any{}}
	if raw, ok := data.Get("fields"); ok && raw != nil {
		m, isMap := raw.(phpval.Map)
		if !isMap || m == nil { // a list (even `[]`) is not a set of named values
			return failValidation(locale, jsonx.Obj("fields", []string{T("validation.not_object", locale)}))
		}
		for _, k := range m.Keys() {
			spec, known := specOf(schema.Fields, k)
			if !known {
				return failValidation(locale, jsonx.Obj("fields."+k, []string{T("validation.unknown_field", locale)}))
			}
			val, _ := m.Get(k)
			in.Fields[k] = typed(spec, val, now)
		}
	}
	if raw, ok := data.Get("items"); ok {
		if len(schema.Items) == 0 {
			return failValidation(locale, jsonx.Obj("items", []string{T("validation.no_items", locale)}))
		}
		in.SetItems, in.Items = true, []map[string]any{}
		if raw != nil {
			_, rows := phpval.Entries(raw)
			for i, r := range rows {
				m, isMap := r.(phpval.Map)
				if !isMap || m == nil {
					return failValidation(locale, jsonx.Obj("items."+strconv.Itoa(i), []string{T("validation.not_object", locale)}))
				}
				row := map[string]any{}
				for _, k := range m.Keys() {
					spec, known := specOf(schema.Items, k)
					if !known {
						return failValidation(locale, jsonx.Obj("items."+strconv.Itoa(i)+"."+k, []string{T("validation.unknown_field", locale)}))
					}
					val, _ := m.Get(k)
					if t := typed(spec, val, now); t != nil {
						row[k] = t
					}
				}
				if len(row) > 0 {
					in.Items = append(in.Items, row)
				}
			}
		}
	}
	if err := h.svc.Review(c.Context(), userID, id, in, now); err != nil {
		return h.fail(err, locale)
	}
	detail, err := h.detail(c, userID, id, now)
	if err != nil {
		return h.fail(err, locale)
	}
	offer, err := h.svc.DatingOffer(c.Context(), userID, id, civildate.InTehran(now))
	if err != nil {
		return h.fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("document", detail, "dating", offerJSON(offer, locale)), T("messages.review_confirmed", locale))
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

// offerJSON is the dating offer as the client shows it.
func offerJSON(o Offer, locale string) *jsonx.OrderedMap {
	var reason, proposed, current, diff any
	msgKey := o.State
	if o.State == OfferUnavailable {
		reason, msgKey = o.Reason, o.Reason
	}
	if !o.Due.IsZero() {
		today := o.AtScan + o.ScanDate.DiffDays(o.Today)
		proposed = jsonx.Obj("ga_weeks", o.AtScan/7, "ga_days", o.AtScan%7, "due_date", o.Due.String(),
			"weeks_today", today/7, "days_today", today%7)
	}
	if !o.CurrentDue.IsZero() {
		current = jsonx.Obj("source", o.CurrentSource, "due_date", o.CurrentDue.String())
		if !o.Due.IsZero() {
			diff = o.CurrentDue.DiffDays(o.Due)
		}
	}
	msg := tp("offer."+msgKey, map[string]string{
		"weeks": strconv.Itoa(o.AtScan / 7), "days": strconv.Itoa(o.AtScan % 7), "date": o.ScanDate.String(),
	}, locale)
	return jsonx.Obj("state", o.State, "reason", reason, "message", msg, "scan_date", dateOrNil(o.ScanDate),
		"proposed", proposed, "current", current, "difference_days", diff)
}

// Dating is GET /health-record/documents/{id}/dating: the pregnancy dating offer of a confirmed imaging document
// (state offered | applied | dismissed | unavailable with a reason). Nothing changes until POST … {confirm: true}.
func (h *Handlers) Dating(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := docID(c)
	if !ok {
		return h.fail(ErrNotFound, locale)
	}
	o, err := h.svc.DatingOffer(c.Context(), userID, id, civildate.InTehran(now))
	if err != nil {
		return h.fail(err, locale)
	}
	return httpx.OK(c, offerJSON(o, locale))
}

// ApplyDating is POST /health-record/documents/{id}/dating {confirm: true}: the user's explicit yes re-dates the
// pregnancy from the scan (409 dating_unavailable when the offer is not open).
func (h *Handlers) ApplyDating(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := docID(c)
	if !ok {
		return h.fail(ErrNotFound, locale)
	}
	data := phpval.NewMap()
	if v, ok := validation.Input(c).Get("confirm"); ok {
		data.Set("confirm", v)
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{validation.F("confirm", "required", "accepted")},
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	o, err := h.svc.ApplyDating(c.Context(), userID, id, now)
	if err != nil {
		return h.fail(err, locale)
	}
	return httpx.OK(c, offerJSON(o, locale), T("messages.dating_applied", locale))
}

// DismissDating is DELETE /health-record/documents/{id}/dating: «نه، همین بماند» — the offer closes, the pregnancy is
// untouched.
func (h *Handlers) DismissDating(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := docID(c)
	if !ok {
		return h.fail(ErrNotFound, locale)
	}
	o, err := h.svc.DismissDating(c.Context(), userID, id, now)
	if err != nil {
		return h.fail(err, locale)
	}
	return httpx.OK(c, offerJSON(o, locale), T("messages.dating_dismissed", locale))
}
