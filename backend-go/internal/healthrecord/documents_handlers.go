package healthrecord

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// ErrorCodeDocumentNotFound is the 404 code of an unknown or foreign record document.
const ErrorCodeDocumentNotFound = "record_document_not_found"

// minDocumentDate is the earliest accepted document / surgery date.
const minDocumentDate = "1950-01-01"

// RT is the records line for key ("messages.document_created") in locale (lang/<code>/records.json).
func RT(key, locale string) string { return translator().Trans("records."+key, nil, locale) }

// RTp is RT with :param replacements.
func RTp(key string, params map[string]string, locale string) string {
	return translator().Trans("records."+key, params, locale)
}

func recordAttributes(locale string) []string {
	v, ok := translator().Get("records.attributes", locale)
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

// DocumentHandlers are the /api/v1/health-record documents, extras, timeline and categories actions (CB-REC-01,
// D-70). Mount them behind the locale middleware and auth RequireUser. Owner-only: every action reads and writes the
// authenticated user's own rows (a foreign id is the uniform 404) and nothing is logged.
type DocumentHandlers struct {
	svc   *Documents
	clock clock.Clock
}

// NewDocumentHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewDocumentHandlers(svc *Documents, base clock.Clock) *DocumentHandlers {
	return &DocumentHandlers{svc: svc, clock: base}
}

func (h *DocumentHandlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func documentNotFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, RT("messages.document_not_found", locale), "error_code", ErrorCodeDocumentNotFound)
}

func fieldError(locale, field, line string) error {
	return failValidation(locale, jsonx.Obj(field, []string{line}))
}

func validateRecords(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now),
		validation.Attributes(recordAttributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func truthy(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	s := phpval.ToString(v)
	return s == "1" || s == "true"
}

func parseDate(s string, now time.Time) civildate.Date {
	if s == "" {
		return civildate.Date{}
	}
	t, err := civildate.ParseLenient(s, now, civildate.Tehran)
	if err != nil {
		return civildate.Date{}
	}
	return civildate.InTehran(t)
}

// --- categories / timeline ------------------------------------------------------------------------------------------

// Categories is GET /health-record/categories (nbl_Rec_Home grid counts).
func (h *DocumentHandlers) Categories(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	body, err := h.svc.Categories(c, userID, i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}

// Timeline is GET /health-record/timeline?kind=&before=&limit= (nbl_Rec_Timeline).
func (h *DocumentHandlers) Timeline(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	data := pick(validation.Input(c), "kind", "before", "limit")
	if err := validateRecords(data, validation.Rules{
		validation.F("kind", "nullable", "string", validation.In(TimelineFilters...)),
		validation.F("before", "nullable", "date"),
		validation.F("limit", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxTimelineLimit)),
	}, locale, now); err != nil {
		return err
	}
	tq := TimelineQuery{Kind: str(data, "kind"), Before: parseDate(str(data, "before"), now), Locale: locale,
		DefaultLocale: i18n.LanguagesOf(c).DefaultCode()}
	if s := str(data, "limit"); s != "" {
		tq.Limit, _ = strconv.Atoi(s)
	}
	body, err := h.svc.Timeline(c, userID, tq)
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}

// --- extras ---------------------------------------------------------------------------------------------------------

// ShowExtras is GET /health-record/extras.
func (h *DocumentHandlers) ShowExtras(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	ex, err := h.svc.GetExtras(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, ex.JSON())
}

// UpdateExtras is PUT /health-record/extras {allergies_on_emergency_card?, surgeries?, family_history?}: a present key
// replaces its value (null clears a list, [] = «ندارم»), an absent key keeps it. Returns the extras.
func (h *DocumentHandlers) UpdateExtras(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	data := pick(validation.Input(c), "allergies_on_emergency_card", "surgeries", "family_history")
	text := "max:" + strconv.Itoa(MaxDocumentText)
	if err := validateRecords(data, validation.Rules{
		validation.F("allergies_on_emergency_card", "sometimes", "required", "boolean"),
		validation.F("surgeries", "nullable", "array", "max:"+strconv.Itoa(MaxSurgeries)),
		validation.F("surgeries.*", "required", "array"),
		validation.F("surgeries.*.title", "required", "string", text),
		validation.F("surgeries.*.date", "nullable", "date", "after_or_equal:"+minDocumentDate, "before_or_equal:today"),
		validation.F("family_history", "nullable", "array", "max:"+strconv.Itoa(MaxFamilyHistory)),
		validation.F("family_history.*", "required", "array"),
		validation.F("family_history.*.condition", "required", "string", text),
		validation.F("family_history.*.relative", "nullable", "string", validation.In(Relatives...)),
	}, locale, now); err != nil {
		return err
	}
	in := ExtrasInput{}
	if v, ok := data.Get("allergies_on_emergency_card"); ok {
		in.SetAllergiesOnCard, in.AllergiesOnCard = true, truthy(v)
	}
	if v, ok := data.Get("surgeries"); ok {
		in.SetSurgeries = true
		if v != nil {
			in.Surgeries = []Surgery{}
			_, items := phpval.Entries(v)
			for i, x := range items {
				m, _ := x.(phpval.Map)
				title := str(m, "title")
				if title == "" {
					return fieldError(locale, "surgeries."+strconv.Itoa(i)+".title", RT("validation.item_blank", locale))
				}
				in.Surgeries = append(in.Surgeries, Surgery{Title: title, Date: surgeryDate(str(m, "date"), now)})
			}
		}
	}
	if v, ok := data.Get("family_history"); ok {
		in.SetFamilyHistory = true
		if v != nil {
			in.FamilyHistory = []FamilyItem{}
			_, items := phpval.Entries(v)
			for i, x := range items {
				m, _ := x.(phpval.Map)
				cond := str(m, "condition")
				if cond == "" {
					return fieldError(locale, "family_history."+strconv.Itoa(i)+".condition", RT("validation.item_blank", locale))
				}
				in.FamilyHistory = append(in.FamilyHistory, FamilyItem{Condition: cond, Relative: str(m, "relative")})
			}
		}
	}
	if err := h.svc.SaveExtras(c, userID, in, now); err != nil {
		return err
	}
	ex, err := h.svc.GetExtras(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, ex.JSON(), RT("messages.extras_saved", locale))
}

// --- documents ------------------------------------------------------------------------------------------------------

var documentFields = []string{"kind", "title", "date", "ended_on", "centre", "doctor", "note", "file_ids", "confirm"}

func documentRules(create bool) validation.Rules {
	kind := []any{"sometimes", "required", "string", validation.In(DocumentKinds...)}
	if create {
		kind = []any{"required", "string", validation.In(DocumentKinds...)}
	}
	text := "max:" + strconv.Itoa(MaxDocumentText)
	return validation.Rules{
		validation.F("kind", kind...),
		validation.F("title", "nullable", "string", text),
		validation.F("date", "nullable", "date", "after_or_equal:"+minDocumentDate, "before_or_equal:today"),
		validation.F("ended_on", "nullable", "date", "after_or_equal:"+minDocumentDate, "before_or_equal:today"),
		validation.F("centre", "nullable", "string", text),
		validation.F("doctor", "nullable", "string", text),
		validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxDocumentNote)),
		validation.F("file_ids", "nullable", "array", "max:"+strconv.Itoa(MaxDocumentFiles)),
		validation.F("file_ids.*", "required", "integer", "min:1"),
		validation.F("confirm", "nullable", "boolean"),
	}
}

// documentInput validates the body and merges it onto base (the stored document on PUT, empty on POST).
func (h *DocumentHandlers) documentInput(c fiber.Ctx, base DocumentInput, create bool, locale string, now time.Time,
) (DocumentInput, error) {
	data := pick(validation.Input(c), documentFields...)
	if err := validateRecords(data, documentRules(create), locale, now); err != nil {
		return base, err
	}
	in := base
	in.SetFiles, in.FileIDs = create, base.FileIDs
	set := func(key string, dst *string) {
		if _, ok := data.Get(key); ok {
			*dst = str(data, key)
		}
	}
	set("kind", &in.Kind)
	set("title", &in.Title)
	set("centre", &in.Centre)
	set("doctor", &in.Doctor)
	set("note", &in.Note)
	if _, ok := data.Get("date"); ok {
		in.Date = parseDate(str(data, "date"), now)
	}
	if _, ok := data.Get("ended_on"); ok {
		in.EndedOn = parseDate(str(data, "ended_on"), now)
	}
	if v, ok := data.Get("confirm"); ok && v != nil {
		in.Confirm = truthy(v)
	}
	if !in.EndedOn.IsZero() && !in.Date.IsZero() && in.EndedOn.Before(in.Date) {
		return base, fieldError(locale, "ended_on", RT("validation.ended_before_date", locale))
	}
	if v, ok := data.Get("file_ids"); ok {
		in.SetFiles, in.FileIDs = true, []uint64{}
		if v != nil {
			_, items := phpval.Entries(v)
			for i, x := range items {
				id, _ := strconv.ParseUint(phpval.ToString(x), 10, 64)
				for _, prev := range in.FileIDs {
					if prev == id {
						return base, fieldError(locale, "file_ids."+strconv.Itoa(i), RT("validation.file_duplicate", locale))
					}
				}
				in.FileIDs = append(in.FileIDs, id)
			}
		}
	}
	if in.FileIDs == nil {
		in.FileIDs = []uint64{}
	}
	return in, nil
}

func (h *DocumentHandlers) writeError(err error, locale string) error {
	var fe *FileError
	switch {
	case errors.As(err, &fe):
		return fieldError(locale, "file_ids."+strconv.Itoa(fe.Index), RT("validation."+fe.Code, locale))
	case errors.Is(err, ErrTooManyDocuments):
		return fieldError(locale, "kind", RTp("validation.too_many_documents",
			map[string]string{"max": strconv.Itoa(MaxDocuments)}, locale))
	case errors.Is(err, ErrDocumentNotFound):
		return documentNotFound(locale)
	}
	return err
}

// StoreDocument is POST /health-record/documents {kind, title?, date?, ended_on?, centre?, doctor?, note?, file_ids?}
// (201 with the document detail). file_ids are the user's own record_document files (POST /files) not attached to
// another document.
func (h *DocumentHandlers) StoreDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := h.documentInput(c, DocumentInput{}, true, locale, now)
	if err != nil {
		return err
	}
	doc, err := h.svc.Create(c, userID, in, now)
	if err != nil {
		return h.writeError(err, locale)
	}
	body, err := h.svc.Detail(c, userID, doc, now)
	if err != nil {
		return err
	}
	return httpx.Created(c, body, RT("messages.document_created", locale))
}

// ShowDocument is GET /health-record/documents/{id} (nbl_Rec_Doc).
func (h *DocumentHandlers) ShowDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := entryID(c)
	if !ok {
		return documentNotFound(locale)
	}
	doc, err := h.svc.Get(c, userID, id)
	if err != nil {
		return h.writeError(err, locale)
	}
	body, err := h.svc.Detail(c, userID, doc, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}

// UpdateDocument is PUT /health-record/documents/{id}: a present key replaces its value (file_ids replaces the files;
// dropped files are deleted), an absent key keeps it; confirm=true moves needs_review → confirmed.
func (h *DocumentHandlers) UpdateDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := entryID(c)
	if !ok {
		return documentNotFound(locale)
	}
	cur, err := h.svc.Get(c, userID, id)
	if err != nil {
		return h.writeError(err, locale)
	}
	base := DocumentInput{ // NULL columns scan to "" / the zero date
		Kind: cur.Kind, Title: cur.Title.String, Centre: cur.Centre.String, Doctor: cur.Doctor.String,
		Note: cur.Note.String, Date: cur.DocumentDate.Date, EndedOn: cur.EndedOn.Date,
	}
	in, err := h.documentInput(c, base, false, locale, now)
	if err != nil {
		return err
	}
	doc, err := h.svc.Update(c, userID, id, in, now)
	if err != nil {
		return h.writeError(err, locale)
	}
	body, err := h.svc.Detail(c, userID, doc, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, body, RT("messages.document_updated", locale))
}

// DestroyDocument is DELETE /health-record/documents/{id}: the document, its links and its files.
func (h *DocumentHandlers) DestroyDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := entryID(c)
	if !ok {
		return documentNotFound(locale)
	}
	if err := h.svc.Delete(c, userID, id); err != nil {
		return h.writeError(err, locale)
	}
	return httpx.OK(c, nil, RT("messages.document_deleted", locale))
}
