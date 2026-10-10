package telemed

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// DoctorAdmin serves /api/admin/v1/telemed/doctor/* (admin-api §19b): the doctor's side of the chat, prescription and
// video link, for the admin account linked by telemed_doctors.admin_id (403 doctor_required otherwise). A booking of
// another doctor is the uniform 404. Message contents are never logged; audit lines carry ids only.
type DoctorAdmin struct {
	ch *Chat
}

// NewDoctorAdmin wires the doctor endpoints.
func NewDoctorAdmin(ch *Chat) *DoctorAdmin { return &DoctorAdmin{ch: ch} }

// Routes mounts the module.
func (h *DoctorAdmin) Routes(route Route, kit *httpadmin.Kit) {
	a := kit.Admin
	p := "/telemed/doctor/bookings"
	route(fiber.MethodGet, p, a(h.Bookings))
	route(fiber.MethodGet, p+"/:id/messages", a(h.Messages))
	route(fiber.MethodPost, p+"/:id/messages", a(h.SendMessage))
	route(fiber.MethodGet, p+"/:id/prescription", a(h.Prescription))
	route(fiber.MethodPost, p+"/:id/prescription", a(h.StoreItem))
	route(fiber.MethodDelete, p+"/:id/prescription/:item", a(h.DestroyItem))
	route(fiber.MethodGet, p+"/:id/video", a(h.Video))
}

func failDoctor(err error) error {
	switch {
	case errors.Is(err, ErrDoctorRequired):
		return httpadmin.Fail(fiber.StatusForbidden, ErrorCodeDoctorRequired, "This account is not linked to a doctor.")
	case errors.Is(err, ErrBookingNotFound):
		return httpadmin.NotFound("Booking")
	case errors.Is(err, ErrPrescriptionNotFound):
		return httpadmin.NotFound("Prescription item")
	case errors.Is(err, ErrChatUnavailable):
		return httpadmin.Fail(fiber.StatusServiceUnavailable, ErrorCodeChatUnavailable, "Chat is unavailable.")
	case errors.Is(err, ErrChatClosed):
		return httpadmin.Fail(fiber.StatusConflict, ErrorCodeChatClosed, "The chat of this visit is closed.")
	case errors.Is(err, ErrMessageLimit):
		return httpadmin.Fail(fiber.StatusConflict, ErrorCodeMessageLimit, "This visit has reached its message limit.")
	case errors.Is(err, ErrPrescriptionLimit):
		return httpadmin.Fail(fiber.StatusConflict, ErrorCodePrescriptionLimit, "This visit has reached its prescription limit.")
	case errors.Is(err, ErrNotVideoVisit):
		return httpadmin.Fail(fiber.StatusConflict, ErrorCodeNotVideoVisit, "This is not a video visit.")
	case errors.Is(err, ErrVideoUnavailable):
		return httpadmin.Fail(fiber.StatusServiceUnavailable, ErrorCodeVideoUnavailable, "Video is unavailable.")
	}
	return err
}

func adminID(c fiber.Ctx) uint64 {
	if a := httpadmin.CurrentAdmin(c); a != nil {
		return a.ID
	}
	return 0
}

func (h *DoctorAdmin) party(c fiber.Ctx) (Party, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return Party{}, failDoctor(ErrBookingNotFound)
	}
	p, err := h.ch.DoctorParty(c.Context(), adminID(c), id)
	if err != nil {
		return Party{}, failDoctor(err)
	}
	return p, nil
}

// doctorBookingJSON is a visit as the doctor sees it in the list (no user data beyond what the visit needs).
func doctorBookingJSON(bk store.TelemedBooking, now time.Time) *jsonx.OrderedMap {
	return jsonx.Obj("id", bk.ID, "reference", bk.Reference, "mode", bk.Mode, "status", bk.Status,
		"starts_at", jsonx.ISO8601(wall(bk.StartsAt)), "ends_at", jsonx.ISO8601(wall(bk.EndsAt)),
		"for_whom", bk.ForWhom, "chat_open", ChatOpen(bk, now))
}

// Bookings is GET /telemed/doctor/bookings — {items}.
func (h *DoctorAdmin) Bookings(c fiber.Ctx) error {
	rows, err := h.ch.DoctorBookings(c.Context(), adminID(c))
	if err != nil {
		return failDoctor(err)
	}
	now := httpadmin.Now(c).In(civildate.Tehran)
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, doctorBookingJSON(r, now))
	}
	return httpadmin.OK(c, jsonx.Obj("items", jsonx.List(items)))
}

// Messages is GET /telemed/doctor/bookings/:id/messages?after_id= — {items, chat}.
func (h *DoctorAdmin) Messages(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	now := httpadmin.Now(c).In(civildate.Tehran)
	ms, err := h.ch.Messages(c.Context(), p, AfterID(validation.Query(c)), now)
	if err != nil {
		return failDoctor(err)
	}
	o, err := h.ch.ThreadJSON(ms, p, now)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, o)
}

// SendMessage is POST /telemed/doctor/bookings/:id/messages {body} — 201 {message}.
func (h *DoctorAdmin) SendMessage(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("body", "required", "string", "max:"+strconv.Itoa(MaxMessageLen)),
	})
	if err != nil {
		return err
	}
	now := httpadmin.Now(c).In(civildate.Tehran)
	m, err := h.ch.Send(c.Context(), p, httpadmin.String(data, "body"), 0, now)
	if err != nil {
		return failDoctor(err)
	}
	o, err := h.ch.MessageJSON(m, p, now)
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("message", o), "Message sent.")
}

// Prescription is GET /telemed/doctor/bookings/:id/prescription — {items}.
func (h *DoctorAdmin) Prescription(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	rows, err := h.ch.Prescription(c.Context(), p)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, ItemsJSON(rows))
}

// PrescriptionRules validate one prescription line {drug, dose?, form, times[1–4] "H:i", duration_days?, instructions?}.
func PrescriptionRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("drug", "required", "string", "max:"+strconv.Itoa(MaxDrugLen)),
		F("dose", "nullable", "string", "max:"+strconv.Itoa(MaxDoseLen)),
		F("form", "required", validation.In(care.Forms...)),
		F("times", "required", "array", "min:1", "max:"+strconv.Itoa(MaxTimesPerDay)),
		F("times.*", "required", "date_format:H:i"),
		F("duration_days", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxDurationDays)),
		F("instructions", "nullable", "string", "max:"+strconv.Itoa(MaxInstructionsLen)),
	}
}

// PrescriptionFrom reads validated prescription data.
func PrescriptionFrom(data phpval.Map) PrescriptionInput {
	in := PrescriptionInput{Drug: str(data, "drug"), Dose: str(data, "dose"), Form: str(data, "form"),
		Instructions: str(data, "instructions")}
	times, _ := data.Get("times")
	_, tv := phpval.Entries(times)
	seen := map[string]bool{}
	for _, t := range tv {
		if s := phpval.ToString(t); !seen[s] {
			seen[s] = true
			in.Times = append(in.Times, s)
		}
	}
	if d, _ := data.Get("duration_days"); d != nil {
		in.DurationDays = int(phpval.ToFloat(d))
	}
	return in
}

// StoreItem is POST /telemed/doctor/bookings/:id/prescription — 201 {item}.
func (h *DoctorAdmin) StoreItem(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	data, err := httpadmin.Validate(c, PrescriptionRules())
	if err != nil {
		return err
	}
	it, err := h.ch.AddItem(c.Context(), p, PrescriptionFrom(data), httpadmin.Now(c).In(civildate.Tehran))
	if err != nil {
		return failDoctor(err)
	}
	return httpadmin.Created(c, jsonx.Obj("item", ItemJSON(it)), "Prescription item added.")
}

// DestroyItem is DELETE /telemed/doctor/bookings/:id/prescription/:item.
func (h *DoctorAdmin) DestroyItem(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	itemID, ok := httpadmin.ID(c, "item")
	if !ok {
		return failDoctor(ErrPrescriptionNotFound)
	}
	if err := h.ch.DeleteItem(c.Context(), p, itemID); err != nil {
		return failDoctor(err)
	}
	return httpadmin.OK(c, jsonx.Obj("id", itemID), "Prescription item deleted.")
}

// Video is GET /telemed/doctor/bookings/:id/video — {video}.
func (h *DoctorAdmin) Video(c fiber.Ctx) error {
	p, err := h.party(c)
	if err != nil {
		return err
	}
	v, err := h.ch.Video(c.Context(), p, httpadmin.Now(c).In(civildate.Tehran))
	if err != nil {
		return failDoctor(err)
	}
	return httpadmin.OK(c, VideoJSON(v))
}
