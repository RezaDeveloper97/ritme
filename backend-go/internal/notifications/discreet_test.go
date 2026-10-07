package notifications

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/profile/store"
)

// allCategories is every category a sender may use: the settings groups plus the extra (pill) ones.
func allCategories() []Category {
	out := append([]Category{}, extraCategories...)
	for _, g := range Groups {
		out = append(out, g.Categories...)
	}
	return out
}

// CB-PRIV-01: every notification template, on every channel (push / web push / SMS), is neutral while the discreet
// flag is on and is the sender's own copy while it is off.
func TestDiscreet_EveryCategoryAndChannel(t *testing.T) {
	on, off := Defaults(), Defaults()
	off.NeutralCopy = false
	require.True(t, on.Discreet(), "discreet is the default (B-N1-11 «متن خنثی» on)")
	require.False(t, off.Discreet())

	for _, c := range allCategories() {
		msg := Message{Category: c, Title: "پریودت ۲ روز دیگر است · " + string(c), Body: "قرص فولیک اسید · روز ۲۶ سیکل", URL: "/x"}
		for _, loc := range []string{"fa", "en"} {
			push := Render(on, msg, loc)
			sms := RenderSMS(on, msg, loc)
			for _, leak := range []string{"پریود", "سیکل", "فولیک", string(c)} {
				assert.NotContains(t, push.Title+push.Body, leak, "push %s %s", c, loc)
				assert.NotContains(t, sms, leak, "sms %s %s", c, loc)
			}
			assert.Equal(t, T("push.neutral_title", loc)+"\n"+T("push.neutral_body", loc), sms)
			assert.Equal(t, "/x", push.URL, "the deep link stays")

			assert.Equal(t, Push{Title: msg.Title, Body: msg.Body, URL: "/x"}, Render(off, msg, loc))
			assert.Equal(t, msg.Title+"\n"+msg.Body, RenderSMS(off, msg, loc))
		}
	}
	assert.Equal(t, "یادآور امروز\nیک یادآور برای امروز داری.", RenderSMS(on, Message{Title: "x"}, "fa"), "board copy")
	assert.Equal(t, "only a title", RenderSMS(off, Message{Title: "only a title"}, "fa"))
}

func TestSMSTemplate(t *testing.T) {
	tmpl, ok := SMSTemplate(true, "invite", "invite-neutral")
	assert.True(t, ok)
	assert.Equal(t, "invite-neutral", tmpl)
	tmpl, ok = SMSTemplate(false, "invite", "invite-neutral")
	assert.True(t, ok)
	assert.Equal(t, "invite", tmpl)
	_, ok = SMSTemplate(true, "invite", "")
	assert.False(t, ok, "discreet without a neutral variant: do not send (no fallback to the regular wording)")
	tmpl, ok = SMSTemplate(false, "invite", "")
	assert.True(t, ok)
	assert.Equal(t, "invite", tmpl)
}

type getter struct {
	row store.NotificationPreference
	err error
}

func (g getter) GetNotificationPreferences(context.Context, uint64) (store.NotificationPreference, error) {
	return g.row, g.err
}

func TestDiscreetLoader_FailsSafe(t *testing.T) {
	ctx := context.Background()
	d, err := Discreet(ctx, getter{err: sql.ErrNoRows}, 1)
	require.NoError(t, err)
	assert.True(t, d, "no row = defaults = discreet")

	d, err = Discreet(ctx, getter{err: errors.New("db down")}, 1)
	require.Error(t, err)
	assert.True(t, d, "a read error never turns neutral copy off")

	d, err = Discreet(ctx, getter{row: store.NotificationPreference{NeutralCopy: false}}, 1)
	require.NoError(t, err)
	assert.False(t, d)
	assert.False(t, strings.Contains(T("push.neutral_body", "fa"), "ریتمی"), "the neutral line does not name the app either")
}
