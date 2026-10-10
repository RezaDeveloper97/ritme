package telemed

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// Chat error codes.
const (
	ErrorCodeChatUnavailable      = "chat_unavailable"
	ErrorCodeChatClosed           = "chat_closed"
	ErrorCodeMessageLimit         = "message_limit"
	ErrorCodePrescriptionNotFound = "prescription_item_not_found"
	ErrorCodePrescriptionAdded    = "prescription_already_added"
	ErrorCodePrescriptionLimit    = "prescription_limit"
	ErrorCodeNotVideoVisit        = "not_video_visit"
	ErrorCodeVideoUnavailable     = "video_unavailable"
	ErrorCodeMedicationLimit      = "medication_limit"
	ErrorCodeDoctorRequired       = "doctor_required"
	ErrorCodeDoctorHasBookings    = "doctor_has_bookings"
)

// ChatHandlers are the user's /api/v1/telemed/bookings/{id}/{messages,prescription,video} actions (auth RequireUser,
// locale middleware). Another user's booking is the uniform 404 booking_not_found.
type ChatHandlers struct {
	ch    *Chat
	clock clock.Clock
}

// NewChatHandlers wires the handlers; base is the fallback clock.
func NewChatHandlers(ch *Chat, base clock.Clock) *ChatHandlers {
	return &ChatHandlers{ch: ch, clock: base}
}

func (h *ChatHandlers) now(c fiber.Ctx) time.Time { return nowIn(c, h.clock) }

// failChat maps the chat errors (user side, localized).
func failChat(err error, locale string) error {
	switch {
	case errors.Is(err, ErrChatUnavailable):
		return coded(fiber.StatusServiceUnavailable, "chat_unavailable", ErrorCodeChatUnavailable, locale)
	case errors.Is(err, ErrChatClosed):
		return coded(fiber.StatusConflict, "chat_closed", ErrorCodeChatClosed, locale)
	case errors.Is(err, ErrMessageLimit):
		return coded(fiber.StatusConflict, "message_limit", ErrorCodeMessageLimit, locale)
	case errors.Is(err, ErrPrescriptionNotFound):
		return coded(fiber.StatusNotFound, "prescription_item_not_found", ErrorCodePrescriptionNotFound, locale)
	case errors.Is(err, ErrPrescriptionAdded):
		return coded(fiber.StatusConflict, "prescription_already_added", ErrorCodePrescriptionAdded, locale)
	case errors.Is(err, ErrNotVideoVisit):
		return coded(fiber.StatusConflict, "not_video_visit", ErrorCodeNotVideoVisit, locale)
	case errors.Is(err, ErrVideoUnavailable):
		return coded(fiber.StatusServiceUnavailable, "video_unavailable", ErrorCodeVideoUnavailable, locale)
	case errors.Is(err, care.ErrMedicationLimit):
		return coded(fiber.StatusUnprocessableEntity, "medication_limit", ErrorCodeMedicationLimit, locale)
	case errors.Is(err, ErrAttachmentInvalid):
		msg := lang.Default().Trans("validation.exists", map[string]string{"attribute": T("attributes.file_id", locale)}, locale)
		return failValidation(locale, jsonx.Obj("file_id", []string{msg}))
	}
	return failBooking(err, locale)
}

func (h *ChatHandlers) party(c fiber.Ctx) (Party, error) {
	userID, err := user(c)
	if err != nil {
		return Party{}, err
	}
	id, ok := idParam(c, "id")
	if !ok {
		return Party{}, failChat(ErrBookingNotFound, i18n.Locale(c))
	}
	p, err := h.ch.UserParty(c, userID, id)
	if err != nil {
		return Party{}, failChat(err, i18n.Locale(c))
	}
	return p, nil
}

// MessageJSON is one message for party p (attachment links are signed for AttachmentLinkTTL).
func (ch *Chat) MessageJSON(m Message, p Party, now time.Time) (*jsonx.OrderedMap, error) {
	var att any
	if m.File != nil {
		link, err := ch.AttachmentLink(*m.File, now)
		if err != nil {
			return nil, err
		}
		att = jsonx.Obj("id", m.File.ID, "mime", m.File.MIME, "size_bytes", m.File.Size, "url", link.URL,
			"url_expires_at", isoPtr(link.ExpiresAt))
	}
	var body any
	if m.Body != nil {
		body = *m.Body
	}
	var read any
	if m.Row.ReadAt.Valid {
		read = jsonx.ISO8601(wall(m.Row.ReadAt.Time))
	}
	return jsonx.Obj("id", m.Row.ID, "sender", m.Row.Sender, "mine", m.Row.Sender == p.Sender, "body", body,
		"attachment", att, "read_at", read, "created_at", jsonx.ISO8601(wall(m.Row.CreatedAt.Time))), nil
}

func isoPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return jsonx.ISO8601(*t)
}

// ThreadJSON is {items, chat: {open, closes_at}}.
func (ch *Chat) ThreadJSON(ms []Message, p Party, now time.Time) (*jsonx.OrderedMap, error) {
	items := make([]*jsonx.OrderedMap, 0, len(ms))
	for _, m := range ms {
		o, err := ch.MessageJSON(m, p, now)
		if err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return jsonx.Obj("items", jsonx.List(items), "chat", jsonx.Obj(
		"open", ChatOpen(p.Booking, now), "closes_at", jsonx.ISO8601(ShareWindowEnd(p.Booking)))), nil
}

// ItemJSON is one prescription line.
func ItemJSON(it store.TelemedPrescriptionItem) *jsonx.OrderedMap {
	var days, dose, instr, rem, added any
	if it.DurationDays.Valid {
		days = it.DurationDays.Int16
	}
	if it.Dose.Valid {
		dose = it.Dose.String
	}
	if it.Instructions.Valid {
		instr = it.Instructions.String
	}
	if it.ReminderID.Valid {
		rem = it.ReminderID.Int64
		if it.AddedAt.Valid {
			added = jsonx.ISO8601(wall(it.AddedAt.Time))
		}
	}
	return jsonx.Obj("id", it.ID, "drug", it.Drug, "dose", dose, "form", it.Form, "times", ItemTimes(it),
		"duration_days", days, "instructions", instr, "reminder_id", rem, "added_at", added,
		"created_at", jsonx.ISO8601(wall(it.CreatedAt.Time)))
}

// ItemsJSON is {items}.
func ItemsJSON(rows []store.TelemedPrescriptionItem) *jsonx.OrderedMap {
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, ItemJSON(r))
	}
	return jsonx.Obj("items", jsonx.List(items))
}

// VideoJSON is {video: {active, opens_at, closes_at, url}}.
func VideoJSON(v VideoLink) *jsonx.OrderedMap {
	var url any
	if v.URL != "" {
		url = v.URL
	}
	return jsonx.Obj("video", jsonx.Obj("active", v.Active, "opens_at", jsonx.ISO8601(v.OpensAt),
		"closes_at", jsonx.ISO8601(v.ClosesAt), "url", url))
}

// AfterID reads ?after_id= (absent or invalid → 0).
func AfterID(q phpval.Map) uint64 {
	n, _ := strconv.ParseUint(str(q, "after_id"), 10, 64)
	return n
}

// ValidateMessage is POST …/messages {body?: ≤ 2000, file_id?: integer} (a body or a file is required). withFile
// false (the doctor) ignores file_id.
func ValidateMessage(input phpval.Map, locale string, now time.Time, withFile bool) (string, uint64, error) {
	keys := []string{"body"}
	rules := validation.Rules{validation.F("body", "required_without:file_id", "nullable", "string", "max:"+strconv.Itoa(MaxMessageLen))}
	if withFile {
		keys = append(keys, "file_id")
		rules = append(rules, validation.F("file_id", "nullable", "integer", "min:1"))
	} else {
		rules[0] = validation.F("body", "required", "string", "max:"+strconv.Itoa(MaxMessageLen))
	}
	data := pick(input, keys)
	if err := validate(data, rules, locale, now); err != nil {
		return "", 0, err
	}
	return str(data, "body"), uintOf(data, "file_id"), nil
}

// Messages is GET /telemed/bookings/{id}/messages?after_id= — {items, chat}.
func (h *ChatHandlers) Messages(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	ms, err := h.ch.Messages(c, p, AfterID(validation.Query(c)), now)
	if err != nil {
		return failChat(err, i18n.Locale(c))
	}
	o, err := h.ch.ThreadJSON(ms, p, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, o)
}

// SendMessage is POST /telemed/bookings/{id}/messages {body?, file_id?} — 201 {message}.
func (h *ChatHandlers) SendMessage(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	body, fileID, err := ValidateMessage(validation.Input(c), locale, now, true)
	if err != nil {
		return err
	}
	m, err := h.ch.Send(c, p, body, fileID, now)
	if err != nil {
		return failChat(err, locale)
	}
	o, err := h.ch.MessageJSON(m, p, now)
	if err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("message", o), T("messages.message_sent", locale))
}

// Prescription is GET /telemed/bookings/{id}/prescription — {items}.
func (h *ChatHandlers) Prescription(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	rows, err := h.ch.Prescription(c, p)
	if err != nil {
		return err
	}
	return httpx.OK(c, ItemsJSON(rows))
}

// AddReminder is POST /telemed/bookings/{id}/prescription/{item}/reminder — 201 {item, medication_id}.
func (h *ChatHandlers) AddReminder(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	itemID, ok := idParam(c, "item")
	if !ok {
		return failChat(ErrPrescriptionNotFound, locale)
	}
	it, medID, err := h.ch.AddReminder(c, p, itemID, i18n.LanguagesOf(c).DefaultCode(), h.now(c))
	if err != nil {
		return failChat(err, locale)
	}
	return httpx.Created(c, jsonx.Obj("item", ItemJSON(it), "medication_id", medID), T("messages.prescription_added", locale))
}

// Video is GET /telemed/bookings/{id}/video — {video}.
func (h *ChatHandlers) Video(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	v, err := h.ch.Video(c, p, h.now(c))
	if err != nil {
		return failChat(err, i18n.Locale(c))
	}
	return httpx.OK(c, VideoJSON(v))
}

func nowIn(c fiber.Ctx, base clock.Clock) time.Time {
	return clock.FromContext(c, base).Now().In(civildate.Tehran).Truncate(time.Second)
}
