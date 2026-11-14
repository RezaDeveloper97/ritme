package assistant

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/assistant/store"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Error codes.
const (
	CodeDepartmentNotFound   = "assistant_department_not_found"
	CodeConversationNotFound = "assistant_conversation_not_found"
	CodePhotoNotFound        = "assistant_photo_not_found"
	CodeSummaryEmpty         = "summary_empty"
	CodeSummaryRequired      = "summary_required"
	CodeSummaryFailed        = "summary_failed"
	CodeShareConsent         = "share_consent_required"
	CodeBookingNotFound      = "booking_not_found"
	CodeBookingClosed        = "booking_closed"
	CodeStorage              = "storage_unavailable"
)

//go:embed lang/*/assistant.json
var langFS embed.FS

var messages = sync.OnceValue(func() map[string]map[string]string {
	out := map[string]map[string]string{}
	dirs, err := fs.ReadDir(langFS, "lang")
	if err != nil {
		panic(err)
	}
	for _, d := range dirs {
		raw, err := langFS.ReadFile(path.Join("lang", d.Name(), "assistant.json"))
		if err != nil {
			panic(err)
		}
		m := map[string]string{}
		if err := json.Unmarshal(raw, &m); err != nil {
			panic(err)
		}
		out[d.Name()] = m
	}
	return out
})

// T is the message key in locale (English fallback).
func T(key, locale string) string {
	m := messages()
	if s, ok := m[locale][key]; ok {
		return s
	}
	if s, ok := m["en"][key]; ok {
		return s
	}
	return key
}

// Handlers serve /api/v1/assistant.
type Handlers struct {
	svc   *Service
	quota Quota
	clock clock.Clock
}

// NewHandlers builds the handlers (plusSvc nil = no quota block in the hub).
func NewHandlers(svc *Service, quota Quota, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, quota: quota, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func idParam(c fiber.Ctx) uint64 {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || strconv.FormatUint(id, 10) != c.Params("id") {
		return 0
	}
	return id
}

func failf(status int, key, code, locale string) error {
	return httpx.Fail(status, T(key, locale), "error_code", code)
}

func fieldFail(locale, field, key string) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("validation_failed", locale), "errors", jsonx.Obj(field, []string{T(key, locale)}))
}

// fail maps service errors to responses (others pass through).
func fail(err error, locale string) error {
	switch {
	case errors.Is(err, ErrDepartmentNotFound):
		return failf(fiber.StatusNotFound, "department_not_found", CodeDepartmentNotFound, locale)
	case errors.Is(err, ErrConversationNotFound):
		return failf(fiber.StatusNotFound, "conversation_not_found", CodeConversationNotFound, locale)
	case errors.Is(err, ErrPhotoNotFound):
		return fieldFail(locale, "photo_id", "photo_not_found")
	case errors.Is(err, ErrSummaryEmpty):
		return failf(fiber.StatusUnprocessableEntity, "summary_empty", CodeSummaryEmpty, locale)
	case errors.Is(err, ErrSummaryRequired):
		return failf(fiber.StatusConflict, "summary_required", CodeSummaryRequired, locale)
	case errors.Is(err, ErrNoSummary):
		return failf(fiber.StatusServiceUnavailable, "summary_failed", CodeSummaryFailed, locale)
	case errors.Is(err, ErrShareConsent):
		return failf(fiber.StatusForbidden, "share_consent_required", CodeShareConsent, locale)
	case errors.Is(err, ErrBookingNotFound):
		return failf(fiber.StatusNotFound, "booking_not_found", CodeBookingNotFound, locale)
	case errors.Is(err, ErrBookingClosed):
		return failf(fiber.StatusConflict, "booking_closed", CodeBookingClosed, locale)
	case errors.Is(err, files.ErrType):
		return fieldFail(locale, "photo", "photo_type")
	case errors.Is(err, files.ErrTooLarge):
		return fieldFail(locale, "photo", "photo_too_large")
	case errors.Is(err, files.ErrQuota):
		return fieldFail(locale, "photo", "photo_quota")
	case errors.Is(err, files.ErrDisabled), errors.Is(err, files.ErrBusy), errors.Is(err, ErrSealUnavailable):
		return failf(fiber.StatusServiceUnavailable, "storage_unavailable", CodeStorage, locale)
	}
	return access.Error(err, locale)
}

func validate(c fiber.Ctx, data phpval.Map, rules validation.Rules, now time.Time) error {
	v := validation.Make(lang.Default(), i18n.Locale(c), data, rules, validation.Now(now))
	if v.Fails() {
		return v.Errors()
	}
	return nil
}

func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

// --- JSON views ---------------------------------------------------------------------------------------------------

type loc struct{ locale, def string }

func locOf(c fiber.Ctx) loc {
	return loc{locale: i18n.Locale(c), def: i18n.LanguagesOf(c).DefaultCode()}
}

func (l loc) text(raw string) any {
	if raw == "" {
		return nil
	}
	s := i18n.PickString(json.RawMessage(raw), l.locale, l.def)
	if s == "" {
		return nil
	}
	return s
}

func (l loc) list(raw string) []string {
	out := []string{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal(i18n.Pick(json.RawMessage(raw), l.locale, l.def), &out)
	return out
}

func nullText(n sql.NullString) any {
	if !n.Valid {
		return nil
	}
	return n.String
}

func departmentJSON(d store.AssistantDepartment, l loc, full bool) *jsonx.OrderedMap {
	m := jsonx.Obj("code", d.Code, "icon", nullText(d.Icon), "name", l.text(d.Name), "tagline", l.text(d.Tagline.String))
	if full {
		m.Set("intro", l.text(d.Intro.String)).Set("does_not_do", l.list(d.DoesNotDo.String)).
			Set("quick_replies", l.list(d.QuickReplies.String))
	}
	return m
}

func dt(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}

func conversationJSON(c store.AssistantConversation, d store.AssistantDepartment, l loc) *jsonx.OrderedMap {
	return jsonx.Obj("id", c.ID, "department", departmentJSON(d, l, false), "title", nullText(c.Title),
		"triage_level", nullText(c.TriageLevel), "emergency", c.Emergency, "message_count", c.MessageCount,
		"has_summary", c.Summary.Valid, "last_message_at", dt(c.LastMessageAt), "created_at", dt(c.CreatedAt))
}

func emergencyCard(locale string) *jsonx.OrderedMap {
	return jsonx.Obj("type", CardEmergency, "phone", EmergencyPhone, "title", T("emergency_title", locale),
		"body", T("emergency_body", locale), "call_label", T("emergency_call", locale))
}

func messageJSON(m store.AssistantMessage, locale string) *jsonx.OrderedMap {
	var photo, card any
	if m.PhotoFileID.Valid {
		photo = jsonx.Obj("id", m.PhotoFileID.Int64)
	}
	if m.Card.Valid && m.Card.String == CardEmergency {
		card = emergencyCard(locale)
	}
	chips := []string{}
	if m.Chips.Valid {
		_ = json.Unmarshal([]byte(m.Chips.String), &chips)
	}
	return jsonx.Obj("id", m.ID, "role", m.Role, "text", m.Body, "photo", photo, "card", card,
		"triage_level", nullText(m.TriageLevel), "guardrail", nullText(m.Guardrail), "chips", chips, "status", m.Status,
		"created_at", dt(m.CreatedAt))
}

func summaryJSON(c store.AssistantConversation) any {
	if !c.Summary.Valid {
		return nil
	}
	return jsonx.Obj("text", c.Summary.String, "triage_level", nullText(c.TriageLevel), "created_at", dt(c.SummaryAt),
		"up_to_date", c.SummaryMessageCount == c.MessageCount)
}

func handoffJSON(hd store.AssistantHandoff) *jsonx.OrderedMap {
	return jsonx.Obj("id", hd.ID, "booking_id", hd.BookingID, "triage_level", nullText(hd.TriageLevel),
		"created_at", dt(hd.CreatedAt), "updated_at", dt(hd.UpdatedAt))
}

// --- Hub, departments, profile --------------------------------------------------------------------------------------

// Quota says what the user can still spend on the assistant (the AI credits of goose 00125): allowed = a running
// subscription with at least one message's cost left; limit / used are the period's credits, remaining everything
// spendable now (the period's plus the extra credits).
type Quota func(ctx context.Context, userID uint64, now time.Time) (QuotaState, error)

// QuotaState is the hub's `quota` object.
type QuotaState struct {
	Allowed   bool
	Limit     uint32
	Used      uint32
	Remaining uint32
}

func (h *Handlers) quotaJSON(c fiber.Ctx, userID uint64) (any, error) {
	if h.quota == nil {
		return nil, nil
	}
	q, err := h.quota(c, userID, h.now(c))
	if err != nil {
		return nil, fmt.Errorf("assistant: quota: %w", err)
	}
	return jsonx.Obj("unlimited", false, "allowed", q.Allowed, "limit", q.Limit, "used", q.Used, "remaining", q.Remaining), nil
}

// Hub is GET /assistant: departments, quota, recent conversations, profile chips.
func (h *Handlers) Hub(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	l := locOf(c)
	depts, err := h.svc.Departments(c)
	if err != nil {
		return err
	}
	byID := map[uint64]store.AssistantDepartment{}
	list := make([]*jsonx.OrderedMap, 0, len(depts))
	for _, d := range depts {
		byID[d.ID] = d
		list = append(list, departmentJSON(d, l, false))
	}
	convs, err := h.svc.List(c, userID, 0)
	if err != nil {
		return err
	}
	recent, err := h.conversations(c, convs, byID, l, 5)
	if err != nil {
		return err
	}
	q, err := h.quotaJSON(c, userID)
	if err != nil {
		return err
	}
	prof, err := h.profileView(c, userID, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("departments", list, "quota", q, "recent", recent, "profile", prof,
		"emergency", jsonx.Obj("phone", EmergencyPhone)))
}

func (h *Handlers) conversations(c fiber.Ctx, convs []store.AssistantConversation, byID map[uint64]store.AssistantDepartment, l loc, limit int,
) ([]*jsonx.OrderedMap, error) {
	out := []*jsonx.OrderedMap{}
	for _, cv := range convs {
		if limit > 0 && len(out) >= limit {
			break
		}
		d, ok := byID[cv.DepartmentID]
		if !ok {
			var err error
			if d, err = h.svc.DepartmentOf(c, cv); err != nil {
				return nil, err
			}
			byID[d.ID] = d
		}
		out = append(out, conversationJSON(cv, d, l))
	}
	return out, nil
}

// ShowDepartment is GET /assistant/departments/{code}: intro, «what it doesn't do», chips, what it knows about her.
func (h *Handlers) ShowDepartment(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	l := locOf(c)
	d, err := h.svc.Department(c, c.Params("code"))
	if err != nil {
		return fail(err, l.locale)
	}
	counts, err := h.svc.ConversationCounts(c, userID)
	if err != nil {
		return err
	}
	prof, err := h.profileView(c, userID, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("department", departmentJSON(d, l, true).Set("conversations_count", counts[d.ID]), "profile", prof))
}

// profileView is {consent, shared, items [{key, shared, empty, data}]} — data only for shared keys.
func (h *Handlers) profileView(c fiber.Ctx, userID uint64, l loc) (*jsonx.OrderedMap, error) {
	granted, err := h.svc.ProfileConsent(c, userID)
	if err != nil {
		return nil, err
	}
	shared, err := h.svc.Shared(c, userID)
	if err != nil {
		return nil, err
	}
	data, err := h.svc.Profile(c, userID, shared, h.now(c), l.locale)
	if err != nil {
		return nil, err
	}
	items := make([]*jsonx.OrderedMap, 0, len(ProfileKeys))
	for _, k := range ProfileKeys {
		on := false
		for _, s := range shared {
			on = on || s == k
		}
		v, has := data[k]
		items = append(items, jsonx.Obj("key", k, "shared", on, "empty", on && (!has || v == nil), "data", v))
	}
	return jsonx.Obj("consent", jsonx.Obj("code", "assistant_profile", "granted", granted), "shared", shared, "items", items), nil
}

// ShowProfile is GET /assistant/profile.
func (h *Handlers) ShowProfile(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	prof, err := h.profileView(c, userID, locOf(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("profile", prof))
}

// UpdateProfile is PUT /assistant/profile {shared: [key…]}.
func (h *Handlers) UpdateProfile(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	body := validation.Input(c)
	data := phpval.NewMap()
	if v, ok := body.Get("shared"); ok {
		data.Set("shared", v)
	}
	if err := validate(c, data, validation.Rules{
		validation.F("shared", "present", "array"),
		validation.F("shared.*", "required", "string", validation.In(ProfileKeys...)),
	}, now); err != nil {
		return err
	}
	var keys []string
	if v, ok := data.Get("shared"); ok {
		_, items := phpval.Entries(v)
		for _, it := range items {
			keys = append(keys, phpval.ToString(it))
		}
	}
	if _, err := h.svc.SetShared(c, userID, keys, now); err != nil {
		return err
	}
	prof, err := h.profileView(c, userID, locOf(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("profile", prof), T("saved", i18n.Locale(c)))
}

// --- Conversations --------------------------------------------------------------------------------------------------

// Index is GET /assistant/conversations?department=code.
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	l := locOf(c)
	var deptID uint64
	if code := strings.TrimSpace(c.Query("department")); code != "" {
		d, err := h.svc.Department(c, code)
		if err != nil {
			return fail(err, l.locale)
		}
		deptID = d.ID
	}
	convs, err := h.svc.List(c, userID, deptID)
	if err != nil {
		return err
	}
	out, err := h.conversations(c, convs, map[uint64]store.AssistantDepartment{}, l, 0)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("conversations", out))
}

// Store is POST /assistant/conversations {department} → 201.
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	l := locOf(c)
	body := validation.Input(c)
	data := phpval.NewMap()
	if v, ok := body.Get("department"); ok {
		data.Set("department", v)
	}
	if err := validate(c, data, validation.Rules{validation.F("department", "required", "string", "max:32")}, now); err != nil {
		return err
	}
	cv, err := h.svc.Create(c, userID, str(data, "department"), now)
	if err != nil {
		return fail(err, l.locale)
	}
	return h.showConversation(c, cv, l, fiber.StatusCreated)
}

func (h *Handlers) showConversation(c fiber.Ctx, cv store.AssistantConversation, l loc, status int) error {
	d, err := h.svc.DepartmentOf(c, cv)
	if err != nil {
		return err
	}
	msgs, err := h.svc.Messages(c, cv)
	if err != nil {
		return err
	}
	list := make([]*jsonx.OrderedMap, 0, len(msgs))
	for _, m := range msgs {
		list = append(list, messageJSON(m, l.locale))
	}
	hds, err := h.svc.Handoffs(c, cv.UserID, cv.ID)
	if err != nil {
		return err
	}
	handoffs := make([]*jsonx.OrderedMap, 0, len(hds))
	for _, hd := range hds {
		handoffs = append(handoffs, handoffJSON(hd))
	}
	body := jsonx.Obj("conversation", conversationJSON(cv, d, l), "department", departmentJSON(d, l, true), "messages", list,
		"summary", summaryJSON(cv), "handoffs", handoffs)
	if status == fiber.StatusCreated {
		return httpx.Created(c, body)
	}
	return httpx.OK(c, body)
}

// Show is GET /assistant/conversations/{id}.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	l := locOf(c)
	cv, err := h.svc.Conversation(c, userID, idParam(c))
	if err != nil {
		return fail(err, l.locale)
	}
	return h.showConversation(c, cv, l, fiber.StatusOK)
}

// Destroy is DELETE /assistant/conversations/{id} (hard delete).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id := idParam(c)
	if id == 0 {
		return fail(ErrConversationNotFound, locale)
	}
	if err := h.svc.Delete(c, userID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("deleted", true), T("deleted", locale))
}

// DestroyAll is DELETE /assistant/conversations (the whole history, hard delete).
func (h *Handlers) DestroyAll(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c, userID, 0); err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("deleted", true), T("deleted_all", i18n.Locale(c)))
}

// --- Photos ---------------------------------------------------------------------------------------------------------

const maxPhotoBody = MaxPhotoBytes + 64<<10

// readPhoto reads the one "photo" part of a multipart body (bounded, no temp files).
func readPhoto(c fiber.Ctx) ([]byte, bool) {
	raw := c.Body()
	if len(raw) > maxPhotoBody || c.Get(fiber.HeaderContentEncoding) != "" {
		return nil, false
	}
	mt, params, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		return nil, false
	}
	r := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
	var data []byte
	for parts := 0; ; parts++ {
		part, err := r.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || parts >= 2 || part.FormName() != "photo" || part.FileName() == "" || data != nil {
			clear(data)
			return nil, false
		}
		if data, err = io.ReadAll(io.LimitReader(part, MaxPhotoBytes+1)); err != nil {
			clear(data)
			return nil, false
		}
	}
	return data, data != nil
}

// StorePhoto is POST /assistant/photos (multipart "photo": JPEG / PNG / WebP ≤ 5 MB) → 201 {photo {id, mime, size}}.
func (h *Handlers) StorePhoto(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if len(c.Body()) > maxPhotoBody {
		return fail(files.ErrTooLarge, locale)
	}
	data, ok := readPhoto(c)
	if !ok {
		return fieldFail(locale, "photo", "photo_required")
	}
	defer clear(data)
	if len(data) > MaxPhotoBytes {
		return fail(files.ErrTooLarge, locale)
	}
	if ai.SniffImage(data) == "" {
		return fail(files.ErrType, locale)
	}
	f, err := h.svc.SavePhoto(c, userID, data, h.now(c))
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, jsonx.Obj("photo", jsonx.Obj("id", f.ID, "mime", f.MIME, "size", f.Size)))
}

// --- Chat (SSE) -----------------------------------------------------------------------------------------------------

type inputKey struct{}

type messageInput struct {
	conv    store.AssistantConversation
	text    string
	photoID uint64
	terms   Terms
}

// sse writes one event and flushes it.
func sse(w *bufio.Writer, event string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw); err != nil {
		return err
	}
	return w.Flush()
}

func sseHeaders(c fiber.Ctx) {
	c.Set(fiber.HeaderContentType, "text/event-stream; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "no-cache, no-store")
	c.Set("X-Accel-Buffering", "no")
	c.Status(fiber.StatusOK)
}

// PrepareMessage validates POST /assistant/conversations/{id}/messages {text, photo_id?} and checks the conversation
// and photo belong to the user. It runs BEFORE the AI gate: an emergency message is answered here with the 115 card
// (SSE, no model call, no consent, no quota); anything else continues to the gate and StreamMessage.
func (h *Handlers) PrepareMessage(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	locale := i18n.Locale(c)
	cv, err := h.svc.Conversation(c, userID, idParam(c))
	if err != nil {
		return fail(err, locale)
	}
	body := validation.Input(c)
	data := phpval.NewMap()
	for _, k := range []string{"text", "photo_id"} {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	if err := validate(c, data, validation.Rules{
		validation.F("text", "required", "string", "max:"+strconv.Itoa(MaxMessageRunes)),
		validation.F("photo_id", "nullable", "integer", "min:1"),
	}, now); err != nil {
		return err
	}
	in := &messageInput{conv: cv, text: str(data, "text")}
	if p := str(data, "photo_id"); p != "" {
		in.photoID, _ = strconv.ParseUint(p, 10, 64)
	}
	if err := h.svc.CheckPhoto(c, userID, in.photoID); err != nil {
		return fail(err, locale)
	}
	if in.terms, err = h.svc.Terms(c); err != nil {
		return err
	}
	if !in.terms.IsEmergency(in.text) {
		c.Locals(inputKey{}, in)
		return c.Next()
	}
	ids, err := h.svc.AddTurns(c, cv, now,
		Turn{Role: RoleUser, Body: in.text, PhotoID: in.photoID},
		Turn{Role: RoleCard, Body: T("emergency_body", locale), Card: CardEmergency, Triage: TriageEmergency})
	if err != nil {
		return err
	}
	card := store.AssistantMessage{ID: ids[1], Role: RoleCard, Body: T("emergency_body", locale), Card: ns(CardEmergency),
		TriageLevel: ns(TriageEmergency), Status: StatusComplete, CreatedAt: nt(now)}
	sseHeaders(c)
	return c.SendStreamWriter(func(w *bufio.Writer) {
		_ = sse(w, "meta", map[string]any{"conversation_id": cv.ID, "user_message_id": ids[0]})
		_ = sse(w, "card", map[string]any{"message": messageJSON(card, locale)})
		_ = sse(w, "done", map[string]any{"message": messageJSON(card, locale), "triage_level": TriageEmergency, "emergency": true})
	})
}

// StreamTimeout bounds one streamed answer end to end (B-N7-06b M1): the provider call is cancelled after it and
// the in-flight slot is released. requestSlack bounds the DB work around a model call (L4).
const (
	StreamTimeout = 90 * time.Second
	requestSlack  = 10 * time.Second
)

// StreamMessage is the model answer of POST /assistant/conversations/{id}/messages, after the AI gate (consent, Plus
// quota reserved, budgets). SSE events: meta {conversation_id, user_message_id}, delta {text}…, card {message} when
// the model rated the turn an emergency, then done {message, triage_level, emergency} or error {error_code, message}.
// Control lines and links are stripped from the stream. The AI gate's in-flight slot is held until the stream writer
// ends (not the handler), and the whole answer is bounded by StreamTimeout.
func (h *Handlers) StreamMessage(c fiber.Ctx) error {
	in, ok := c.Locals(inputKey{}).(*messageInput)
	if !ok {
		return httpx.ServerError()
	}
	now := h.now(c)
	locale := i18n.Locale(c)
	release := access.DetachRelease(c.Context())
	streamCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Context()), StreamTimeout)
	started := false
	defer func() {
		if !started {
			cancel()
			release()
		}
	}()
	prep, err := h.svc.Prepare(streamCtx, in.conv, in.text, in.photoID, in.terms, locale, now)
	if err != nil {
		return err
	}
	stream, err := h.svc.AI().Chat(streamCtx, ai.FeatureAssistant, prep.Request)
	if err != nil {
		return access.Error(err, locale)
	}
	ids, err := h.svc.AddTurns(streamCtx, in.conv, now, Turn{Role: RoleUser, Body: in.text, PhotoID: in.photoID})
	if err != nil {
		stream.Close()
		return fail(err, locale)
	}
	res := access.ReservationFrom(c.Context())
	conv := in.conv
	clk := clock.FromContext(c, h.clock)
	sseHeaders(c)
	started = true
	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer release()
		defer cancel()
		defer stream.Close()
		// Writes after the stream: off the (possibly expired) stream deadline, but bounded.
		ctx, done := context.WithTimeout(context.WithoutCancel(streamCtx), requestSlack)
		defer done()
		gone := sse(w, "meta", map[string]any{"conversation_id": conv.ID, "user_message_id": ids[0]}) != nil
		var (
			f       streamFilter
			got     bool
			failed  error
			finish  string
			endedOK bool
		)
		emit := func(out string) {
			if out == "" || gone {
				return
			}
			if sse(w, "delta", map[string]any{"text": out}) != nil {
				gone = true
				stream.Close() // the client went away: stop the provider now
			}
		}
		for ev := range stream.Events {
			if ev.Err != nil {
				failed = ev.Err
				break
			}
			if ev.Delta != "" {
				got = true
				emit(f.push(ev.Delta))
			}
			if ev.Done {
				finish, endedOK = ev.Finish, true
				break
			}
		}
		if failed == nil && !endedOK {
			failed = context.DeadlineExceeded // the stream ended without Done: timed out / cancelled
		}
		emit(f.flush())
		visible, ctl := splitControls(f.text())
		if !got || (failed != nil && visible == "") {
			if !got {
				res.Refund(ctx)
			}
			if !gone {
				_ = sse(w, "error", map[string]any{"error_code": access.CodeFailed, "message": T("stream_failed", locale)})
			}
			return
		}
		status := StatusComplete
		if failed != nil {
			status = StatusPartial
		}
		if finish == ai.FinishSafety && visible == "" {
			visible = T("safety_fallback", locale)
		}
		at := clk.Now().In(civildate.Tehran).Truncate(time.Second)
		turns := []Turn{{Role: RoleAssistant, Body: visible, Triage: ctl.Triage, Guardrail: prep.Guardrail, Chips: ctl.Chips, Status: status}}
		emergency := conv.Emergency || ctl.Triage == TriageEmergency
		if ctl.Triage == TriageEmergency { // M3: the model rated it an emergency → the 115 card too
			turns = append(turns, Turn{Role: RoleCard, Body: T("emergency_body", locale), Card: CardEmergency, Triage: TriageEmergency})
		}
		mids, err := h.svc.AddTurns(ctx, conv, at, turns...)
		if err != nil || gone {
			return
		}
		msg := store.AssistantMessage{ID: mids[0], Role: RoleAssistant, Body: visible, TriageLevel: ns(ctl.Triage),
			Guardrail: ns(prep.Guardrail), Status: status, CreatedAt: nt(at)}
		if len(ctl.Chips) > 0 {
			raw, _ := json.Marshal(ctl.Chips)
			msg.Chips = ns(string(raw))
		}
		if len(mids) > 1 {
			card := store.AssistantMessage{ID: mids[1], Role: RoleCard, Body: T("emergency_body", locale), Card: ns(CardEmergency),
				TriageLevel: ns(TriageEmergency), Status: StatusComplete, CreatedAt: nt(at)}
			_ = sse(w, "card", map[string]any{"message": messageJSON(card, locale)})
		}
		_ = sse(w, "done", map[string]any{"message": messageJSON(msg, locale), "triage_level": nullText(msg.TriageLevel),
			"emergency": emergency})
	})
}

// --- Summary and handoff --------------------------------------------------------------------------------------------

// Summary is POST /assistant/conversations/{id}/summary: writes (or returns) the summary document.
func (h *Handlers) Summary(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	l := locOf(c)
	ctx, cancel := context.WithTimeout(c.Context(), summaryTimeout+requestSlack) // L4: per-request deadline
	defer cancel()
	cv, err := h.svc.Summarise(ctx, userID, idParam(c), l.locale, h.now(c))
	if err != nil {
		return fail(err, l.locale)
	}
	d, err := h.svc.DepartmentOf(c, cv)
	if err != nil {
		return err
	}
	photos := 0
	msgs, err := h.svc.Messages(c, cv)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		if m.PhotoFileID.Valid {
			photos++
		}
	}
	return httpx.OK(c, jsonx.Obj("conversation", conversationJSON(cv, d, l), "summary", summaryJSON(cv),
		"photos_count", photos, "triage_levels", TriageLevels))
}

// Handoff is POST /assistant/conversations/{id}/handoff {booking_id}: the summary goes to that visit's doctor.
func (h *Handlers) Handoff(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	locale := i18n.Locale(c)
	body := validation.Input(c)
	data := phpval.NewMap()
	if v, ok := body.Get("booking_id"); ok {
		data.Set("booking_id", v)
	}
	if err := validate(c, data, validation.Rules{validation.F("booking_id", "required", "integer", "min:1")}, now); err != nil {
		return err
	}
	bookingID, _ := strconv.ParseUint(str(data, "booking_id"), 10, 64)
	hd, err := h.svc.Handoff(c, userID, idParam(c), bookingID, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("handoff", handoffJSON(hd)), T("handoff_sent", locale))
}
