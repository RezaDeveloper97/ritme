package consent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/consent/store"
)

// Ledger actions (consent_events.action).
const (
	ActionGrant  = "grant"
	ActionRevoke = "revoke"
)

// Ledger sources (consent_events.source): where the act happened.
const (
	SourceSignup    = "signup"    // the signup screen (verify-otp `consents`)
	SourceSettings  = "settings"  // the Privacy toggles, the one-time pending sheet
	SourceFeature   = "feature"   // a feature's consent sheet (PUT /consents/{code}), the assistant item switches
	SourceFamily    = "family"    // family links (T-M11-08)
	SourceTelemed   = "telemed"   // a telemed booking's data share
	SourceShareLink = "sharelink" // a doctor report share link
	SourceBackfill  = "backfill"  // goose 00101: events reconstructed from user_consents
	SourceAdmin     = "admin"     // an admin action on the user's behalf
)

var sources = []string{SourceSignup, SourceSettings, SourceFeature, SourceFamily, SourceTelemed, SourceShareLink,
	SourceBackfill, SourceAdmin}

func validSource(s string) bool { return slices.Contains(sources, s) }

// HistoryLimit caps GET /consents/history (newest first).
const HistoryLimit = 200

// Event is one ledger row.
type Event struct {
	ID      uint64
	Consent string
	Version int
	Action  string
	Scope   json.RawMessage // nil for a whole-consent event
	Source  string
	At      sql.NullTime
}

// History is the user's own events, newest first (at most HistoryLimit).
func (s *Service) History(ctx context.Context, userID uint64) ([]Event, error) {
	rows, err := s.q.ListConsentEvents(ctx, store.ListConsentEventsParams{UserID: userID, Limit: HistoryLimit})
	if err != nil {
		return nil, fmt.Errorf("consent: history: %w", err)
	}
	out := make([]Event, 0, len(rows))
	for _, r := range rows {
		e := Event{ID: r.ID, Consent: r.Consent, Version: int(r.Version), Action: r.Action, Source: r.Source, At: r.CreatedAt}
		if r.Scope.Valid {
			e.Scope = r.Scope.V
		}
		out = append(out, e)
	}
	return out, nil
}

// Scoped is an evidence-only event: what the user shared, with whom, under which scope. It has no projection and
// gates nothing (T-M10-23 §6).
type Scoped struct {
	UserID  uint64
	Consent string // an evidence code (AssistantProfileItems, TelemedShare, ShareLink)
	Action  string // ActionGrant | ActionRevoke
	Scope   any    // marshalled to JSON
	Source  string
	At      time.Time
}

// ErrInvalidEvent is a Scoped event with an unknown code, action or source, or a scope that does not marshal.
var ErrInvalidEvent = errors.New("consent: invalid event")

// AppendScoped writes e through dbtx — the caller's transaction, so the share and its evidence commit together.
func AppendScoped(ctx context.Context, dbtx store.DBTX, e Scoped) error {
	d, ok := lookupEvidence(e.Consent)
	if !ok || (e.Action != ActionGrant && e.Action != ActionRevoke) || !validSource(e.Source) {
		return fmt.Errorf("%w: %s %s %s", ErrInvalidEvent, e.Consent, e.Action, e.Source)
	}
	var scope db.NullRawJSON
	if e.Scope != nil {
		raw, err := json.Marshal(e.Scope)
		if err != nil {
			return fmt.Errorf("%w: scope: %w", ErrInvalidEvent, err)
		}
		scope = db.NullRawJSON{V: raw, Valid: true}
	}
	err := store.New(dbtx).InsertConsentEvent(ctx, store.InsertConsentEventParams{
		UserID: e.UserID, Consent: d.Code, Version: uint32(d.Version), Action: e.Action, Scope: scope, Source: e.Source, //nolint:gosec // G115: small catalog version
		Now: sql.NullTime{Time: e.At, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("consent: append %s: %w", e.Consent, err)
	}
	return nil
}

// Signup is what verify-otp needs (auth.SignupConsents): the mandatory codes with the version in force, and
// recording the versions the signup screen showed. auth cannot import this package (import cycle), so the route
// wiring hands it over.
type Signup struct{ svc *Service }

// NewSignup wires the signup consents on svc.
func NewSignup(svc *Service) Signup { return Signup{svc: svc} }

// Requirement is one mandatory consent and the version that must be accepted (an alias of an unnamed struct, so it
// is auth.SignupConsent without either package importing the other).
type Requirement = struct {
	Code    string
	Version int
}

// Required are the mandatory consents in display order.
func (Signup) Required() []Requirement {
	ds := Mandatory()
	out := make([]Requirement, len(ds))
	for i, d := range ds {
		out[i] = Requirement{Code: d.Code, Version: d.Version}
	}
	return out
}

// Record grants every mandatory consent at the version in versions (source signup), in display order. A version that
// is not the one in force is ErrStaleVersion (the caller validated them first).
func (s Signup) Record(ctx context.Context, userID uint64, versions map[string]int, now time.Time) error {
	for _, d := range Mandatory() {
		if err := s.svc.AcceptFrom(ctx, userID, d.Code, versions[d.Code], SourceSignup, now); err != nil {
			return err
		}
	}
	return nil
}

// RequiredMessage is the 422 consent_required message of verify-otp in locale.
func (Signup) RequiredMessage(locale string) string { return T("signup_required", locale) }
