package companion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"time"
)

// Invite SMS limits (security review M-1). A refused send is not an error: the code is still returned and
// sms_sent=false, so the owner shares it herself.
const (
	// SMSPerRecipientPerDay caps invite SMS to one mobile number from all owners per SMSDay.
	SMSPerRecipientPerDay = 3
	// SMSPerOwnerPerDay caps the invite SMS one owner triggers per SMSDay.
	SMSPerOwnerPerDay = 5
	// SMSDay is the window of the two daily caps.
	SMSDay = 24 * time.Hour
	// SMSResendAfter is the minimum gap before a renew re-sends to the same number of the same link.
	SMSResendAfter = 2 * time.Minute
)

// SendGate is a fixed-window counter (internal/http adapts the Redis rate limiter): Allow counts one hit on key and
// reports whether it is within max per window.
type SendGate interface {
	Allow(ctx context.Context, key string, maxHits int, window time.Duration) (bool, error)
}

// smsAllowed applies, in order, the per-link resend gap (a changed number is a new key), the per-recipient and the
// per-owner daily caps. Keys hold a hash of the number, never the number. A gate error fails closed (no SMS).
func (h *Handlers) smsAllowed(ctx context.Context, inv CreatedInvite) bool {
	if h.opt.SendGate == nil {
		return true
	}
	sum := sha256.Sum256([]byte("companion-sms:" + inv.Phone))
	phone := hex.EncodeToString(sum[:])
	checks := []struct {
		key     string
		maxHits int
		window  time.Duration
	}{
		{"companion-sms-link:" + strconv.FormatUint(inv.Link.ID, 10) + ":" + phone, 1, SMSResendAfter},
		{"companion-sms-to:" + phone, SMSPerRecipientPerDay, SMSDay},
		{"companion-sms-owner:" + strconv.FormatUint(inv.Link.OwnerID, 10), SMSPerOwnerPerDay, SMSDay},
	}
	for _, chk := range checks {
		ok, err := h.opt.SendGate.Allow(ctx, chk.key, chk.maxHits, chk.window)
		if err != nil {
			h.logger.ErrorContext(ctx, "companion invite SMS gate failed", slog.String("error", err.Error()))
			return false
		}
		if !ok {
			return false
		}
	}
	return true
}
