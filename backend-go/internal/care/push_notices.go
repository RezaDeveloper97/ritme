package care

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The in-app inbox (user_notifications, the bell) by Web Push: every row a domain writes there — companion and
// family notices, an opened course, the message engine's push slot, postpartum visits — also reaches the browsers
// the user subscribed. The writers stay unaware of push: the sender polls the rows whose pushed_at is NULL (goose
// 00123) and claims each one before it sends, so the API and the worker never announce a row twice.
//
// A row is announced once, when it is new, unread, its category is on (notifications.DecideNotice) and the user
// has a subscription. Inside quiet hours it waits for the end of the window; everything else is claimed and stays
// silent. «متن خنثی» replaces the text with «اعلان تازه» (notifications.RenderNotice).

const (
	// NoticeLookback is how old an inbox row may be and still be pushed: long enough to outlast a night of quiet
	// hours, short enough that a restart or a late subscription does not announce stale news.
	NoticeLookback = 12 * time.Hour
	// noticeBatch is the page size of one TickNotices read.
	noticeBatch = 200
	// noticeFallbackPath is opened by a tap when the row has no in-app path of its own (the home, with the bell).
	noticeFallbackPath = "/home"
)

// noticeText picks the text of a {lang: text} JSON column for locale: that language, else fa, else en, else the
// first language in code order. A bare JSON string is taken as is; anything else is "".
func noticeText(raw json.RawMessage, locale string) string {
	var byLang map[string]string
	if json.Unmarshal(raw, &byLang) != nil {
		var s string
		_ = json.Unmarshal(raw, &s)
		return strings.TrimSpace(s)
	}
	for _, code := range []string{locale, "fa", "en"} {
		if s := strings.TrimSpace(byLang[code]); s != "" {
			return s
		}
	}
	codes := make([]string, 0, len(byLang))
	for code := range byLang {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	for _, code := range codes {
		if s := strings.TrimSpace(byLang[code]); s != "" {
			return s
		}
	}
	return ""
}

// noticeURL is the page a tap opens: the row's in-app path under the subscription's language. Anything that is
// not a plain in-app path ("/x", never "//host" or a scheme) falls back to the home.
func noticeURL(action sql.NullString, locale string) string {
	path := noticeFallbackPath
	if a := action.String; action.Valid && strings.HasPrefix(a, "/") && !strings.HasPrefix(a, "//") && !strings.ContainsAny(a, "\\\r\n") {
		path = a
	}
	return "/" + locale + path
}

// noticeMessage is what the push of row says in locale, before RenderNotice.
func noticeMessage(row store.ListUnpushedNotificationsRow, locale string) notifications.Message {
	m := notifications.Message{Title: noticeText(row.Title, locale), URL: noticeURL(row.ActionUrl, locale)}
	if row.Body.Valid {
		m.Body = noticeText(row.Body.V, locale)
	}
	return m
}

// TickNotices pushes the inbox rows that are not handled yet and returns how many went out.
func (p *Pusher) TickNotices(ctx context.Context, now time.Time) (int, error) {
	now = now.In(civildate.Tehran)
	type userState struct {
		subs     []store.PushSubscription
		prefs    notifications.Preferences
		hasPrefs bool
	}
	users := map[uint64]*userState{}
	sent := 0
	var after uint64
	for {
		rows, err := p.q.ListUnpushedNotifications(ctx, store.ListUnpushedNotificationsParams{AfterID: after, Limit: noticeBatch})
		if err != nil {
			return sent, fmt.Errorf("care: unpushed notifications: %w", err)
		}
		for _, row := range rows {
			after = row.ID
			u := users[row.UserID]
			// Stale, already seen in the app, or nowhere to send: handled without a push.
			silent := !row.CreatedAt.Valid || now.Sub(row.CreatedAt.Time) > NoticeLookback || row.ReadAt.Valid
			if !silent {
				if u == nil {
					u = &userState{}
					if u.subs, err = p.q.ListPushSubscriptions(ctx, row.UserID); err != nil {
						return sent, fmt.Errorf("care: push subscriptions: %w", err)
					}
					users[row.UserID] = u
				}
				silent = len(u.subs) == 0
			}
			if !silent {
				if !u.hasPrefs {
					if u.prefs, err = notifications.Load(ctx, p.prefs, row.UserID); err != nil {
						return sent, err
					}
					u.hasPrefs = true
				}
				if d := notifications.DecideNotice(u.prefs, row.Type, now); !d.Send {
					if d.Reason == notifications.ReasonQuietHours {
						continue // not claimed: a later tick sends it when the window is over
					}
					silent = true
				}
			}
			n, err := p.q.ClaimNotificationPush(ctx, store.ClaimNotificationPushParams{
				Now: sql.NullTime{Time: now, Valid: true}, ID: row.ID,
			})
			if err != nil {
				return sent, fmt.Errorf("care: claim notification push: %w", err)
			}
			if n == 0 || silent {
				continue
			}
			delivered, err := p.deliver(ctx, u.subs, fmt.Sprintf("n:%d", row.ID), func(locale string) notifications.Push {
				return notifications.RenderNotice(u.prefs, noticeMessage(row, locale), locale)
			})
			if err != nil {
				return sent, err
			}
			if delivered {
				sent++
			}
		}
		if len(rows) < noticeBatch {
			return sent, nil
		}
	}
}
