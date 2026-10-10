package telemed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// Doctor chat, e-prescription and video link (bloom B-N7-04, deviation D-82):
//
//	GET    /api/v1/telemed/bookings/{id}/messages                       the thread (?after_id=), marks doctor lines read
//	POST   /api/v1/telemed/bookings/{id}/messages                       {body?, file_id?} (a telemed_attachment file)
//	GET    /api/v1/telemed/bookings/{id}/prescription                   the doctor's prescription lines
//	POST   /api/v1/telemed/bookings/{id}/prescription/{item}/reminder   one tap → care medication reminder
//	GET    /api/v1/telemed/bookings/{id}/video                          the room link, active 10 min before → end
//	/api/admin/v1/telemed/doctor/*                                      the same thread for the doctor (chat_admin.go)
//
// Only two parties ever reach a thread: the booking's user (GetUserBooking) and the admin linked to the booking's
// doctor (telemed_doctors.admin_id); anybody else gets the uniform 404. Message bodies are sealed with the
// PRIVATE_NOTE_KEY box (internal/loss NoteBox) bound to booking + sender, and never logged. Writing is open while the
// visit is confirmed and until ShareWindowEnd (72 h after it ends).

// Message senders (telemed_messages.sender).
const (
	SenderUser   = "user"
	SenderDoctor = "doctor"
)

// Chat limits.
const (
	MaxMessageLen        = 2000
	MaxMessages          = 300 // per booking
	MaxPrescriptionItems = 10  // per booking
	MaxDrugLen           = 120
	MaxDoseLen           = 50
	MaxInstructionsLen   = 500
	MaxDurationDays      = 365
	MaxTimesPerDay       = 4
	// VideoLead: the video link becomes active this long before the visit starts (and stays until it ends).
	VideoLead = 10 * time.Minute
	// AttachmentLinkTTL: signed attachment links expire after this.
	AttachmentLinkTTL = 5 * time.Minute
)

// Chat errors mapped by the handlers.
var (
	ErrDoctorRequired       = errors.New("telemed: admin is not linked to a doctor")
	ErrChatUnavailable      = errors.New("telemed: chat unavailable (no key)")
	ErrChatClosed           = errors.New("telemed: chat closed for this booking")
	ErrMessageLimit         = errors.New("telemed: message limit reached")
	ErrAttachmentInvalid    = errors.New("telemed: attachment not usable")
	ErrPrescriptionNotFound = errors.New("telemed: prescription item not found")
	ErrPrescriptionAdded    = errors.New("telemed: prescription item already added")
	ErrPrescriptionLimit    = errors.New("telemed: prescription item limit reached")
	ErrNotVideoVisit        = errors.New("telemed: not a video visit")
	ErrVideoUnavailable     = errors.New("telemed: video provider unavailable")
)

// Sealer encrypts message bodies at rest (*loss.NoteBox).
type Sealer interface {
	SealWith(plain string, ad []byte) (string, error)
	OpenWith(stored string, ad []byte) (string, error)
	Disabled() bool
}

// Attachments reads the user's stored files (*files.Service).
type Attachments interface {
	Get(ctx context.Context, owner, id uint64) (files.File, error)
	Link(f files.File, now time.Time, ttl time.Duration) (files.Link, error)
	Disabled() bool
}

// Chat is the per-booking messaging, prescription and video service.
type Chat struct {
	b     *Bookings
	q     *store.Queries
	box   Sealer
	files Attachments
	video VideoProvider
}

// NewChat wires the chat on the booking service. files / video may be nil (no attachments / no video provider).
func NewChat(b *Bookings, box Sealer, att Attachments, video VideoProvider) *Chat {
	return &Chat{b: b, q: b.q, box: box, files: att, video: video}
}

// Party is the caller of a thread: the booking's user or its doctor (as admin AdminID).
type Party struct {
	Sender  string
	AdminID uint64
	Booking store.TelemedBooking
}

// UserParty resolves the user's own booking (held / lapsed / failed bookings have no thread: ErrBookingNotFound).
func (ch *Chat) UserParty(ctx context.Context, userID, bookingID uint64) (Party, error) {
	bk, err := ch.q.GetUserBooking(ctx, store.GetUserBookingParams{ID: bookingID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Party{}, ErrBookingNotFound
	}
	if err != nil {
		return Party{}, fmt.Errorf("telemed: booking: %w", err)
	}
	switch bk.Status {
	case StatusConfirmed, StatusCompleted, StatusCancelled:
		return Party{Sender: SenderUser, Booking: bk}, nil
	}
	return Party{}, ErrBookingNotFound
}

// Doctor is the doctor linked to the admin (ErrDoctorRequired when none).
func (ch *Chat) Doctor(ctx context.Context, adminID uint64) (store.TelemedDoctor, error) {
	d, err := ch.q.GetDoctorByAdmin(ctx, sql.NullInt64{Int64: int64(adminID), Valid: adminID > 0}) //nolint:gosec // G115: admin id
	if errors.Is(err, sql.ErrNoRows) {
		return store.TelemedDoctor{}, ErrDoctorRequired
	}
	if err != nil {
		return store.TelemedDoctor{}, fmt.Errorf("telemed: doctor by admin: %w", err)
	}
	return d, nil
}

// DoctorParty resolves a confirmed or completed booking of the admin's doctor (anything else: ErrBookingNotFound).
func (ch *Chat) DoctorParty(ctx context.Context, adminID, bookingID uint64) (Party, error) {
	d, err := ch.Doctor(ctx, adminID)
	if err != nil {
		return Party{}, err
	}
	bk, err := ch.q.GetDoctorBooking(ctx, store.GetDoctorBookingParams{ID: bookingID, DoctorID: d.ID})
	if errors.Is(err, sql.ErrNoRows) {
		return Party{}, ErrBookingNotFound
	}
	if err != nil {
		return Party{}, fmt.Errorf("telemed: booking: %w", err)
	}
	if bk.Status != StatusConfirmed && bk.Status != StatusCompleted {
		return Party{}, ErrBookingNotFound
	}
	return Party{Sender: SenderDoctor, AdminID: adminID, Booking: bk}, nil
}

// DoctorBookings lists the confirmed and completed visits of the admin's doctor.
func (ch *Chat) DoctorBookings(ctx context.Context, adminID uint64) ([]store.TelemedBooking, error) {
	d, err := ch.Doctor(ctx, adminID)
	if err != nil {
		return nil, err
	}
	rows, err := ch.q.ListDoctorBookings(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("telemed: doctor bookings: %w", err)
	}
	return rows, nil
}

// ChatOpen reports whether messages and prescription lines may be written on bk at now.
func ChatOpen(bk store.TelemedBooking, now time.Time) bool {
	return (bk.Status == StatusConfirmed || bk.Status == StatusCompleted) && !now.After(ShareWindowEnd(bk))
}

func messageAD(bookingID uint64, sender string) []byte {
	return []byte("telemed-message:v1:" + strconv.FormatUint(bookingID, 10) + ":" + sender)
}

// Message is one decrypted message. Body is nil when the stored text does not open (rotated-out key).
type Message struct {
	Row  store.TelemedMessage
	Body *string
	File *files.File
}

func (ch *Chat) open(ctx context.Context, bk store.TelemedBooking, r store.TelemedMessage) Message {
	m := Message{Row: r}
	if plain, err := ch.box.OpenWith(r.Body, messageAD(bk.ID, r.Sender)); err == nil {
		m.Body = &plain
	}
	if r.FileID.Valid && ch.files != nil && !ch.files.Disabled() {
		if f, err := ch.files.Get(ctx, bk.UserID, uint64(r.FileID.Int64)); err == nil { //nolint:gosec // G115: positive id
			m.File = &f
		}
	}
	return m
}

// Messages lists the thread after afterID (oldest first, ≤ 200) and marks the other party's lines read.
func (ch *Chat) Messages(ctx context.Context, p Party, afterID uint64, now time.Time) ([]Message, error) {
	if ch.box == nil || ch.box.Disabled() {
		return nil, ErrChatUnavailable
	}
	rows, err := ch.q.ListMessages(ctx, store.ListMessagesParams{BookingID: p.Booking.ID, AfterID: afterID})
	if err != nil {
		return nil, fmt.Errorf("telemed: messages: %w", err)
	}
	if err := ch.q.MarkMessagesRead(ctx, store.MarkMessagesReadParams{Now: nt(now), BookingID: p.Booking.ID, Reader: p.Sender}); err != nil {
		return nil, fmt.Errorf("telemed: mark read: %w", err)
	}
	out := make([]Message, 0, len(rows))
	for _, r := range rows {
		out = append(out, ch.open(ctx, p.Booking, r))
	}
	return out, nil
}

// Send writes a message of p. Only the user attaches files (her own telemed_attachment files).
func (ch *Chat) Send(ctx context.Context, p Party, body string, fileID uint64, now time.Time) (Message, error) {
	if ch.box == nil || ch.box.Disabled() {
		return Message{}, ErrChatUnavailable
	}
	if !ChatOpen(p.Booking, now) {
		return Message{}, ErrChatClosed
	}
	n, err := ch.q.CountMessages(ctx, p.Booking.ID)
	if err != nil {
		return Message{}, fmt.Errorf("telemed: count messages: %w", err)
	}
	if n >= MaxMessages {
		return Message{}, ErrMessageLimit
	}
	file := sql.NullInt64{}
	if fileID > 0 {
		if p.Sender != SenderUser || ch.files == nil || ch.files.Disabled() {
			return Message{}, ErrAttachmentInvalid
		}
		f, err := ch.files.Get(ctx, p.Booking.UserID, fileID)
		if err != nil || f.Purpose != files.PurposeTelemedAttachment {
			return Message{}, ErrAttachmentInvalid
		}
		file = sql.NullInt64{Int64: int64(fileID), Valid: true} //nolint:gosec // G115: positive id
	}
	sealed, err := ch.box.SealWith(body, messageAD(p.Booking.ID, p.Sender))
	if err != nil {
		return Message{}, err
	}
	admin := sql.NullInt64{Int64: int64(p.AdminID), Valid: p.AdminID > 0} //nolint:gosec // G115: admin id
	id, err := ch.q.InsertMessage(ctx, store.InsertMessageParams{
		BookingID: p.Booking.ID, Sender: p.Sender, AdminID: admin, Body: sealed, FileID: file, Now: nt(now),
	})
	if err != nil {
		return Message{}, fmt.Errorf("telemed: insert message: %w", err)
	}
	row, err := ch.q.GetMessage(ctx, store.GetMessageParams{ID: uint64(id), BookingID: p.Booking.ID}) //nolint:gosec // G115: auto-increment id
	if err != nil {
		return Message{}, fmt.Errorf("telemed: message: %w", err)
	}
	return ch.open(ctx, p.Booking, row), nil
}

// AttachmentLink is a short-lived signed link to a message's file (issued only after the party check).
func (ch *Chat) AttachmentLink(f files.File, now time.Time) (files.Link, error) {
	return ch.files.Link(f, now, AttachmentLinkTTL)
}

// ---------------------------------------------------------------------------
// Prescription

// PrescriptionInput is a validated prescription line.
type PrescriptionInput struct {
	Drug, Dose, Form string
	Times            []string // "HH:MM", 1–MaxTimesPerDay
	DurationDays     int      // 0 = ongoing
	Instructions     string
}

// ItemTimes splits the stored times column.
func ItemTimes(it store.TelemedPrescriptionItem) []string {
	if it.Times == "" {
		return []string{}
	}
	return strings.Split(it.Times, ",")
}

// Prescription lists the booking's prescription lines.
func (ch *Chat) Prescription(ctx context.Context, p Party) ([]store.TelemedPrescriptionItem, error) {
	rows, err := ch.q.ListPrescriptionItems(ctx, p.Booking.ID)
	if err != nil {
		return nil, fmt.Errorf("telemed: prescription: %w", err)
	}
	return rows, nil
}

// AddItem adds a prescription line (doctor only, while the chat is open).
func (ch *Chat) AddItem(ctx context.Context, p Party, in PrescriptionInput, now time.Time) (store.TelemedPrescriptionItem, error) {
	if p.Sender != SenderDoctor {
		return store.TelemedPrescriptionItem{}, ErrBookingNotFound
	}
	if !ChatOpen(p.Booking, now) {
		return store.TelemedPrescriptionItem{}, ErrChatClosed
	}
	n, err := ch.q.CountPrescriptionItems(ctx, p.Booking.ID)
	if err != nil {
		return store.TelemedPrescriptionItem{}, fmt.Errorf("telemed: count prescription: %w", err)
	}
	if n >= MaxPrescriptionItems {
		return store.TelemedPrescriptionItem{}, ErrPrescriptionLimit
	}
	days := sql.NullInt16{Int16: int16(in.DurationDays), Valid: in.DurationDays > 0} //nolint:gosec // G115: ≤ 365
	id, err := ch.q.InsertPrescriptionItem(ctx, store.InsertPrescriptionItemParams{
		BookingID: p.Booking.ID, Drug: in.Drug, Dose: ns(in.Dose), Form: in.Form, Times: strings.Join(in.Times, ","),
		DurationDays: days, Instructions: ns(in.Instructions), Now: nt(now),
	})
	if err != nil {
		return store.TelemedPrescriptionItem{}, fmt.Errorf("telemed: insert prescription: %w", err)
	}
	return ch.q.GetPrescriptionItem(ctx, store.GetPrescriptionItemParams{ID: uint64(id), BookingID: p.Booking.ID}) //nolint:gosec // G115: auto-increment id
}

// DeleteItem removes a prescription line (doctor only).
func (ch *Chat) DeleteItem(ctx context.Context, p Party, itemID uint64) error {
	if p.Sender != SenderDoctor {
		return ErrBookingNotFound
	}
	n, err := ch.q.DeletePrescriptionItem(ctx, store.DeletePrescriptionItemParams{ID: itemID, BookingID: p.Booking.ID})
	if err != nil {
		return fmt.Errorf("telemed: delete prescription: %w", err)
	}
	if n == 0 {
		return ErrPrescriptionNotFound
	}
	return nil
}

// AddReminder turns a prescription line into the user's medication reminder (user only, once per line while the
// reminder exists). subtitleLocale is the default language.
func (ch *Chat) AddReminder(ctx context.Context, p Party, itemID uint64, subtitleLocale string, now time.Time) (store.TelemedPrescriptionItem, uint64, error) {
	if p.Sender != SenderUser {
		return store.TelemedPrescriptionItem{}, 0, ErrBookingNotFound
	}
	var reminderID uint64
	err := ch.b.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		it, err := q.LockPrescriptionItem(ctx, store.LockPrescriptionItemParams{ID: itemID, BookingID: p.Booking.ID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPrescriptionNotFound
		}
		if err != nil {
			return fmt.Errorf("telemed: prescription item: %w", err)
		}
		if it.ReminderID.Valid {
			return ErrPrescriptionAdded
		}
		days := 0
		if it.DurationDays.Valid {
			days = int(it.DurationDays.Int16)
		}
		reminderID, err = care.AddPrescribedMedication(ctx, carestore.New(tx), p.Booking.UserID, care.PrescribedMedication{
			Title: it.Drug, Dose: it.Dose.String, Form: it.Form, Times: ItemTimes(it), DurationDays: days,
			Notes: it.Instructions.String, StartsOn: civildate.InTehran(now),
		}, subtitleLocale, now)
		if err != nil {
			return err
		}
		return q.MarkPrescriptionAdded(ctx, store.MarkPrescriptionAddedParams{
			ReminderID: sql.NullInt64{Int64: int64(reminderID), Valid: true}, Now: nt(now), ID: it.ID, //nolint:gosec // G115: auto-increment id
		})
	})
	if err != nil {
		return store.TelemedPrescriptionItem{}, 0, err
	}
	it, err := ch.q.GetPrescriptionItem(ctx, store.GetPrescriptionItemParams{ID: itemID, BookingID: p.Booking.ID})
	return it, reminderID, err
}

// ---------------------------------------------------------------------------
// Video

// VideoLink is the video room of a booking: Active from OpensAt until ClosesAt, URL only while active.
type VideoLink struct {
	Active            bool
	OpensAt, ClosesAt time.Time
	URL               string
}

// Video is the booking's room link (ErrNotVideoVisit for phone / in-person visits).
func (ch *Chat) Video(ctx context.Context, p Party, now time.Time) (VideoLink, error) {
	bk := p.Booking
	if bk.Mode != ModeVideo {
		return VideoLink{}, ErrNotVideoVisit
	}
	v := VideoLink{OpensAt: wall(bk.StartsAt).Add(-VideoLead), ClosesAt: wall(bk.EndsAt)}
	v.Active = bk.Status == StatusConfirmed && !now.Before(v.OpensAt) && now.Before(v.ClosesAt)
	if !v.Active {
		return v, nil
	}
	if ch.video == nil {
		return VideoLink{}, ErrVideoUnavailable
	}
	url, err := ch.video.Room(ctx, bk)
	if err != nil {
		return VideoLink{}, fmt.Errorf("telemed: video room: %w", err)
	}
	v.URL = url
	return v, nil
}
