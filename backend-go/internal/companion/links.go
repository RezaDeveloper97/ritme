package companion

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/companion/store"
)

// OwnerLink is one of ownerID's invited or active links (ErrNotFound for anyone else's, a revoked one or a bad id),
// with the open invite's expiry and phone.
func (s *Service) OwnerLink(ctx context.Context, ownerID, companionID uint64) (Link, error) {
	q := store.New(s.conn)
	c, err := q.GetCompanion(ctx, companionID)
	if err != nil || c.OwnerID != ownerID || Status(c.Status) == StatusRevoked {
		return Link{}, notFound(err)
	}
	l, err := loadLink(ctx, q, companionID)
	if err != nil {
		return Link{}, err
	}
	invites, err := q.ListOpenInvitesForOwner(ctx, store.ListOpenInvitesForOwnerParams{OwnerID: ownerID, Now: nt(s.now())})
	if err != nil {
		return Link{}, fmt.Errorf("companion: invites: %w", err)
	}
	for _, inv := range invites {
		if inv.CompanionID == l.ID {
			l.InviteExpiresAt, l.InvitePhone = inv.ExpiresAt.Time, inv.Phone.String
			break
		}
	}
	return l, nil
}

// CompanionLink is an active link in which viewerID is the companion (ErrNotFound otherwise: another account's
// link, a pending or revoked one, a bad id — existence is not revealed).
func (s *Service) CompanionLink(ctx context.Context, viewerID, companionID uint64) (Link, error) {
	q := store.New(s.conn)
	c, err := q.GetCompanion(ctx, companionID)
	if err != nil || Status(c.Status) != StatusActive || !c.CompanionUserID.Valid ||
		uint64(c.CompanionUserID.Int64) != viewerID { //nolint:gosec // positive id
		return Link{}, notFound(err)
	}
	return loadLink(ctx, q, companionID)
}

// Names returns users.name for the given account ids (missing or unnamed accounts are absent).
func (s *Service) Names(ctx context.Context, ids ...uint64) (map[uint64]string, error) {
	out := map[uint64]string{}
	ids = compactIDs(ids)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := store.New(s.conn).GetUserNames(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("companion: names: %w", err)
	}
	for _, r := range rows {
		if n := strings.TrimSpace(r.Name.String); n != "" {
			out[r.ID] = n
		}
	}
	return out, nil
}

func compactIDs(ids []uint64) []uint64 {
	seen := map[uint64]bool{}
	out := ids[:0:0]
	for _, id := range ids {
		if id != 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// Notice is an owner-inbox event.
type Notice string

// Owner-inbox notices (user_notifications.type = companion). No health payload: the title names the companion and
// the kind of record, never its content.
const (
	NoticeAccepted            Notice = "accepted"
	NoticeRecordedMeds        Notice = "recorded_meds"
	NoticeRecordedAppointment Notice = "recorded_appointments"
	NoticeUpdatedMeds         Notice = "updated_meds"
	NoticeUpdatedAppointment  Notice = "updated_appointments"
)

// NotificationType is the user_notifications.type of every companion notice.
const NotificationType = "companion"

// Paths the notices open in the web app.
const (
	pathCompanions = "/companions"
	pathReminders  = "/reminders"
)

// NotifyOwner writes an owner-inbox row about link (the link's companion did something). languages are the active
// language codes (the `languages` table): the title/body JSON carries one entry per language. The companion is named
// by the owner's own label for the link, else the account name, else a neutral word.
func (s *Service) NotifyOwner(ctx context.Context, link Link, notice Notice, languages []string) error {
	name := strings.TrimSpace(link.DisplayName)
	if name == "" && link.CompanionUserID != 0 {
		names, err := s.Names(ctx, link.CompanionUserID)
		if err != nil {
			return err
		}
		name = names[link.CompanionUserID]
	}
	title, body := map[string]string{}, map[string]string{}
	bodyKey, url := "notices.recorded_body", pathReminders
	if notice == NoticeAccepted {
		bodyKey, url = "notices.accepted_body", pathCompanions
	}
	for _, code := range languages {
		n := name
		if n == "" {
			n = T("notices.someone", code)
		}
		title[code] = T("notices."+string(notice)+"_title", code, "name", n)
		body[code] = T(bodyKey, code)
	}
	rawTitle, err := json.Marshal(title)
	if err != nil {
		return fmt.Errorf("companion: notice title: %w", err)
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("companion: notice body: %w", err)
	}
	data, err := json.Marshal(map[string]any{"companion_id": link.ID, "event": string(notice)})
	if err != nil {
		return fmt.Errorf("companion: notice data: %w", err)
	}
	if err := store.New(s.conn).InsertUserNotification(ctx, store.InsertUserNotificationParams{
		UserID: link.OwnerID, Type: NotificationType, Title: json.RawMessage(rawTitle),
		Body: rootdb.NullRawJSON{V: rawBody, Valid: true}, ActionUrl: sql.NullString{String: url, Valid: true},
		Data: rootdb.NullRawJSON{V: data, Valid: true}, Now: nt(s.now()),
	}); err != nil {
		return fmt.Errorf("companion: notice: %w", err)
	}
	return nil
}
