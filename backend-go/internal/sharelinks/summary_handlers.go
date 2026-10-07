package sharelinks

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Error codes of the record-sharing routes (CB-REC-03, D-71).
const (
	CodeCodeNotFound    = "share_code_not_found"
	CodeCodeLimit       = "share_code_limit"
	CodeCodeUnavailable = "share_code_unavailable"
	CodeTooMany         = "too_many_attempts"
	DefaultSummaryRange = healthrecord.ReportRange6M
	OverviewAccessItems = 10
)

// FamilyMember is one companion link as the sharing overview lists it (companion.Service.RecordScope).
type FamilyMember struct {
	ID                 uint64
	Type, Status, Name string
	Meds, Appointments string // none | view | edit
}

// FamilySource lists the owner's companion links with their record scope.
type FamilySource interface {
	Family(ctx context.Context, ownerID uint64) ([]FamilyMember, error)
}

// CodeBreaker is the global failed-code-lookup circuit breaker (internal/http implements it on Redis; nil = none).
type CodeBreaker interface {
	Open(ctx context.Context) (open bool, retryAfter int, err error)
	Failed(ctx context.Context) (tripped bool, err error)
}

// SharingHandlers are the record-sharing actions (owner routes under /api/v1/health-record, auth RequireUser; the
// public POST /api/v1/shared-reports/code is IP-throttled and breaker-guarded). No action logs a code, a token or a
// report.
type SharingHandlers struct {
	svc     *Service
	clock   clock.Clock
	family  FamilySource
	webURL  string
	breaker CodeBreaker
	onTrip  func(ctx context.Context)
}

// NewSharingHandlers wires the handlers. family may be nil (no family block), webURL "" (qr_url null), breaker nil.
func NewSharingHandlers(svc *Service, base clock.Clock, family FamilySource, webURL string, breaker CodeBreaker,
	onTrip func(ctx context.Context),
) *SharingHandlers {
	return &SharingHandlers{svc: svc, clock: base, family: family, webURL: webURL, breaker: breaker, onTrip: onTrip}
}

func (h *SharingHandlers) now(c fiber.Ctx) time.Time {
	return (&Handlers{clock: h.clock}).now(c)
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func attributes(locale string) []string {
	v, ok := translator().Get("sharelinks.attributes", locale)
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

func codeUnavailable(locale string) error {
	return httpx.Fail(fiber.StatusServiceUnavailable, T("messages.code_unavailable", locale), "error_code", CodeCodeUnavailable)
}

func codeNotFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.code_not_found", locale), "error_code", CodeCodeNotFound)
}

// parseSummary validates {label?, range?, from?, sections?, document_ids?}: range defaults to 6m and sections to every
// record section; the 422 is the controller envelope.
func parseSummary(body phpval.Map, locale string, now time.Time) (SummaryRequest, error) {
	data := phpval.NewMap()
	for _, k := range []string{"label", "document_ids"} {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		validation.F("label", "nullable", "string", "max:"+strconv.Itoa(MaxLabelLen)),
		validation.F("document_ids", "nullable", "array", "max:"+strconv.Itoa(MaxSharedDocuments)),
		validation.F("document_ids.*", "required", "integer", "min:1"),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return SummaryRequest{}, failValidation(locale, v.ErrorBag())
	}
	report := phpval.NewMap()
	for _, k := range []string{"range", "from", "sections"} {
		if x, ok := body.Get(k); ok && x != nil {
			report.Set(k, x)
		}
	}
	if _, ok := report.Get("range"); !ok {
		report.Set("range", DefaultSummaryRange)
	}
	if _, ok := report.Get("sections"); !ok {
		all := make([]any, 0, len(healthrecord.Sections))
		for _, s := range healthrecord.Sections {
			all = append(all, s)
		}
		report.Set("sections", all)
	}
	rr, err := healthrecord.ParseReportRequest(report, locale, now, false)
	if err != nil {
		return SummaryRequest{}, err
	}
	req := SummaryRequest{Report: rr}
	if l, ok := data.Get("label"); ok && l != nil {
		req.Label = phpval.ToString(l)
	}
	if ids, ok := data.Get("document_ids"); ok {
		_, vals := phpval.Entries(ids)
		seen := map[uint64]bool{}
		for _, x := range vals { // validated integers ≥ 1; duplicates dropped, first position kept
			id, ok := toUint(x)
			if !ok {
				n, err := strconv.ParseUint(phpval.ToString(x), 10, 64)
				id, ok = n, err == nil && n > 0
			}
			if ok && !seen[id] {
				seen[id] = true
				req.DocumentIDs = append(req.DocumentIDs, id)
			}
		}
	}
	return req, nil
}

// StoreCode is POST /health-record/share-codes {label?, range?, from?, sections?, document_ids?} → 201 {summary…,
// code, token, qr_path, qr_url, documents_count}. The code and the token are returned only here.
func (h *SharingHandlers) StoreCode(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	if h.svc.CodesDisabled() {
		return codeUnavailable(locale)
	}
	req, err := parseSummary(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	created, err := h.svc.CreateSummary(c, userID, req, now, locale, i18n.LanguagesOf(c).DefaultCode())
	var docErr *ErrDocument
	switch {
	case errors.As(err, &docErr):
		return failValidation(locale, jsonx.Obj("document_ids."+strconv.Itoa(docErr.Index),
			[]string{T("validation.document_not_found", locale)}))
	case errors.Is(err, ErrSummaryLimit):
		return httpx.Fail(fiber.StatusConflict,
			Tp("messages.code_limit", map[string]string{"max": strconv.Itoa(MaxActiveSummaries)}, locale),
			"error_code", CodeCodeLimit)
	case errors.Is(err, ErrCodesDisabled):
		return codeUnavailable(locale)
	case err != nil:
		return err
	}
	path := "/" + locale + "/shared/report/" + created.Token
	var qrURL any
	if h.webURL != "" {
		qrURL = h.webURL + path
	}
	body := created.Summary.JSON(now)
	body.Set("documents_count", created.Documents)
	body.Set("code", FormatCode(created.Code))
	body.Set("token", created.Token)
	body.Set("qr_path", path)
	body.Set("qr_url", qrURL)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return httpx.Created(c, body, T("messages.code_created", locale))
}

// DestroyCode is DELETE /health-record/share-codes/{id}: revokes the summary now (404 for an unknown or foreign id
// or a report link id).
func (h *SharingHandlers) DestroyCode(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, perr := strconv.ParseUint(c.Params("id"), 10, 64)
	if perr != nil || id == 0 {
		return codeNotFound(locale)
	}
	s, err := h.svc.RevokeSummary(c, userID, id, now)
	if errors.Is(err, ErrNotFound) {
		return codeNotFound(locale)
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, s.JSON(now), T("messages.code_revoked", locale))
}

func accessJSON(entries []AccessEntry, now time.Time) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.JSON(now))
	}
	return out
}

// Access is GET /health-record/share-access?link_id=&limit= (nbl_Rec_Share «سابقه دسترسی»): the owner's access log,
// newest first, of every link or of one of hers (404 for a foreign link id).
func (h *SharingHandlers) Access(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	data := phpval.NewMap()
	q := validation.Query(c)
	for _, k := range []string{"link_id", "limit"} {
		if v, ok := q.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		validation.F("link_id", "nullable", "integer", "min:1"),
		validation.F("limit", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxAccessLimit)),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	var linkID uint64
	limit := DefaultAccessLimit
	if x, ok := data.Get("link_id"); ok && x != nil {
		linkID, _ = strconv.ParseUint(phpval.ToString(x), 10, 64)
	}
	if x, ok := data.Get("limit"); ok && x != nil {
		if n, err := strconv.Atoi(phpval.ToString(x)); err == nil {
			limit = n
		}
	}
	entries, err := h.svc.Access(c, userID, linkID, limit)
	if errors.Is(err, ErrNotFound) {
		return notFound(locale)
	}
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("items", accessJSON(entries, now)))
}

// Overview is GET /health-record/sharing (nbl_Rec_Share «چه کسی پرونده‌ات را می‌بیند؟»): the doctor codes of the last
// 30 days, the companion links with their record scope (meds & appointments only, documents always false) and the
// latest access log entries.
func (h *SharingHandlers) Overview(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	summaries, err := h.svc.ListSummaries(c, userID, now)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(summaries))
	active := 0
	for _, s := range summaries {
		if s.Status(now) == StatusActive {
			active++
		}
		items = append(items, s.JSON(now))
	}
	family := []*jsonx.OrderedMap{}
	if h.family != nil {
		members, err := h.family.Family(c, userID)
		if err != nil {
			return err
		}
		for _, m := range members {
			var name any
			if m.Name != "" {
				name = m.Name
			}
			family = append(family, jsonx.Obj("companion_id", m.ID, "type", m.Type, "status", m.Status, "name", name,
				"meds", m.Meds, "appointments", m.Appointments, "documents", false))
		}
	}
	entries, err := h.svc.Access(c, userID, 0, OverviewAccessItems)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj(
		"doctor", jsonx.Obj("available", !h.svc.CodesDisabled(), "ttl_hours", int(SummaryTTL/time.Hour),
			"active_count", active, "max_active", MaxActiveSummaries, "items", items),
		"family", family,
		"access_log", accessJSON(entries, now),
	))
}

// OpenCode is the public POST /shared-reports/code {code}: the summary behind a typed code, the same body as
// GET /shared-reports/{token}. Every refusal — malformed, unknown, expired, revoked — is the same 404 and counts
// against the global breaker; while the breaker is open every lookup is 429.
func (h *SharingHandlers) OpenCode(c fiber.Ctx) error {
	now, locale := h.now(c), i18n.Locale(c)
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	c.Set(fiber.HeaderReferrerPolicy, "no-referrer")
	if h.svc.CodesDisabled() {
		return codeUnavailable(locale)
	}
	if h.breaker != nil {
		open, retry, err := h.breaker.Open(c.Context())
		if err != nil {
			return err // fail closed
		}
		if open {
			return httpx.Fail(fiber.StatusTooManyRequests, T("messages.too_many_attempts", locale),
				"error_code", CodeTooMany, "retry_after", retry).WithHeader(fiber.HeaderRetryAfter, strconv.Itoa(retry))
		}
	}
	data := phpval.NewMap()
	if x, ok := validation.Input(c).Get("code"); ok {
		data.Set("code", x)
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{
		validation.F("code", "required", "string", "max:32"),
	}, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	code, _ := data.Get("code")
	opened, err := h.svc.OpenCode(c, phpval.ToString(code), now, ViewerOf(ViaCode, c.Get(fiber.HeaderUserAgent)))
	switch {
	case errors.Is(err, ErrNotFound):
		h.failed(c)
		return codeNotFound(locale)
	case errors.Is(err, ErrCodesDisabled):
		return codeUnavailable(locale)
	case err != nil:
		return err
	}
	opened.Report.Set("expires_at", jsonx.ISO8601(opened.ExpiresAt))
	return httpx.OK(c, opened.Report)
}

func (h *SharingHandlers) failed(c fiber.Ctx) {
	if h.breaker == nil {
		return
	}
	tripped, err := h.breaker.Failed(c.Context())
	if err == nil && tripped && h.onTrip != nil {
		h.onTrip(c.Context())
	}
}
