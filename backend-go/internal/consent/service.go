package consent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/consent/store"
)

// Errors of the service.
var (
	ErrUnknown      = errors.New("consent: unknown code")
	ErrStaleVersion = errors.New("consent: version is not the one in force")
)

// Reasons a consent is required.
const (
	ReasonMissing  = "missing"  // never granted, or withdrawn
	ReasonOutdated = "outdated" // granted for an older text
)

// RequiredError is what a feature gets when the user has not accepted the text in force of Code.
type RequiredError struct {
	Code    string
	Version int // the version the user must accept
	Reason  string
}

func (e *RequiredError) Error() string {
	return fmt.Sprintf("consent: %s v%d required (%s)", e.Code, e.Version, e.Reason)
}

// Store is the persistence of the service (internal/consent/store; user_id in every query).
type Store interface {
	GetConsent(ctx context.Context, arg store.GetConsentParams) (store.GetConsentRow, error)
	ListConsents(ctx context.Context, userID uint64) ([]store.ListConsentsRow, error)
	GrantConsent(ctx context.Context, arg store.GrantConsentParams) error
	RevokeConsent(ctx context.Context, arg store.RevokeConsentParams) error
	InsertConsentEvent(ctx context.Context, arg store.InsertConsentEventParams) error
	ListConsentEvents(ctx context.Context, arg store.ListConsentEventsParams) ([]store.ListConsentEventsRow, error)
}

// Service reads and writes the user's consents: the ledger (consent_events) and its projection (user_consents).
type Service struct {
	q  Store
	db *sql.DB // nil with NewServiceWith: writes then run on q without a transaction (unit tests)
}

// NewService wires the service on db.
func NewService(db *sql.DB) *Service { return &Service{q: store.New(db), db: db} }

// NewServiceWith wires the service on a store (tests).
func NewServiceWith(q Store) *Service { return &Service{q: q} }

// write runs fn in one transaction that holds the user's row lock, so a grant or withdrawal and its event commit
// together or not at all, and one user's concurrent writes are serialised (no duplicate or lost events).
func (s *Service) write(ctx context.Context, userID uint64, fn func(q Store) error) error {
	if s.db == nil {
		return fn(s.q)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("consent: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := store.New(tx)
	if _, err := q.LockConsentOwner(ctx, userID); err != nil {
		return fmt.Errorf("consent: lock user: %w", err)
	}
	if err := fn(q); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("consent: commit: %w", err)
	}
	return nil
}

// State is the user's answer for one consent.
type State struct {
	Code      string
	Current   int  // version in force
	Granted   bool // the row says granted (whatever its version)
	Version   int  // version accepted last; 0 = none
	GrantedAt sql.NullTime
	RevokedAt sql.NullTime
}

// Valid reports whether the text in force is accepted.
func (s State) Valid() bool { return s.Granted && s.Version >= s.Current }

// Reason is why the state is not valid ("" when valid).
func (s State) Reason() string {
	switch {
	case s.Valid():
		return ""
	case s.Granted:
		return ReasonOutdated
	default:
		return ReasonMissing
	}
}

func stateOf(d Definition, granted bool, version sql.NullInt16, grantedAt, revokedAt sql.NullTime) State {
	st := State{Code: d.Code, Current: d.Version, Granted: granted, GrantedAt: grantedAt, RevokedAt: revokedAt}
	if version.Valid {
		st.Version = int(version.Int16)
	}
	return st
}

// Get is the user's state for code.
func (s *Service) Get(ctx context.Context, userID uint64, code string) (State, error) {
	d, ok := Lookup(code)
	if !ok {
		return State{}, ErrUnknown
	}
	row, err := s.q.GetConsent(ctx, store.GetConsentParams{UserID: userID, Consent: code})
	if errors.Is(err, sql.ErrNoRows) {
		return State{Code: code, Current: d.Version}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("consent: get %s: %w", code, err)
	}
	return stateOf(d, row.Granted, row.Version, row.GrantedAt, row.RevokedAt), nil
}

// List is the user's state for every feature consent (not the mandatory signup ones), in display order.
func (s *Service) List(ctx context.Context, userID uint64) ([]State, error) {
	rows, err := s.q.ListConsents(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("consent: list: %w", err)
	}
	by := make(map[string]store.ListConsentsRow, len(rows))
	for _, r := range rows {
		by[r.Consent] = r
	}
	out := make([]State, 0, len(catalog))
	for _, d := range catalog {
		if d.Mandatory {
			continue // GET /consents/pending lists the signup consents
		}
		r, ok := by[d.Code]
		if !ok {
			out = append(out, State{Code: d.Code, Current: d.Version})
			continue
		}
		out = append(out, stateOf(d, r.Granted, r.Version, r.GrantedAt, r.RevokedAt))
	}
	return out, nil
}

// Require is nil when the user accepted the text in force of code, a *RequiredError otherwise. A store error is
// returned as is (callers fail closed: no provider call without a positive answer).
func (s *Service) Require(ctx context.Context, userID uint64, code string) error {
	st, err := s.Get(ctx, userID, code)
	if err != nil {
		return err
	}
	if st.Valid() {
		return nil
	}
	return &RequiredError{Code: code, Version: st.Current, Reason: st.Reason()}
}

// Accept is AcceptFrom a feature's consent sheet (source feature).
func (s *Service) Accept(ctx context.Context, userID uint64, code string, version int, now time.Time) error {
	return s.AcceptFrom(ctx, userID, code, version, SourceFeature, now)
}

// Withdraw is WithdrawFrom a feature's consent sheet (source feature).
func (s *Service) Withdraw(ctx context.Context, userID uint64, code string, now time.Time) error {
	return s.WithdrawFrom(ctx, userID, code, SourceFeature, now)
}

// AcceptFrom records the user's acceptance of version of code at now, from source (Source*). Only the version in
// force can be accepted (ErrStaleVersion otherwise: the client showed an old text). The grant event and the projection
// are written in one transaction; accepting the version already accepted is a no-op (no second event).
func (s *Service) AcceptFrom(ctx context.Context, userID uint64, code string, version int, source string, now time.Time) error {
	d, ok := Lookup(code)
	if !ok {
		return ErrUnknown
	}
	if version != d.Version {
		return ErrStaleVersion
	}
	if !validSource(source) {
		return fmt.Errorf("consent: accept %s: unknown source %q", code, source)
	}
	at := sql.NullTime{Time: now, Valid: true}
	return s.write(ctx, userID, func(q Store) error {
		cur, err := current(ctx, q, userID, d)
		if err != nil {
			return err
		}
		if cur.Granted && cur.Version == version {
			return nil
		}
		if err := q.InsertConsentEvent(ctx, store.InsertConsentEventParams{
			UserID: userID, Consent: code, Version: uint32(version), Action: ActionGrant, Source: source, Now: at, //nolint:gosec // G115: small catalog version
		}); err != nil {
			return fmt.Errorf("consent: accept %s: event: %w", code, err)
		}
		if err := q.GrantConsent(ctx, store.GrantConsentParams{
			UserID: userID, Consent: code, Version: sql.NullInt16{Int16: int16(version), Valid: true}, Now: at, //nolint:gosec // G115: small catalog version
		}); err != nil {
			return fmt.Errorf("consent: accept %s: %w", code, err)
		}
		return nil
	})
}

// WithdrawFrom records the user's withdrawal of code at now, from source. A withdrawal of a granted consent writes a
// revoke event (the version last accepted; 0 for a grant from before versioned texts) with the projection; answering
// «no» to what was never granted only stores the explicit «no» (no event: nothing was withdrawn).
func (s *Service) WithdrawFrom(ctx context.Context, userID uint64, code string, source string, now time.Time) error {
	d, ok := Lookup(code)
	if !ok {
		return ErrUnknown
	}
	if !validSource(source) {
		return fmt.Errorf("consent: withdraw %s: unknown source %q", code, source)
	}
	at := sql.NullTime{Time: now, Valid: true}
	return s.write(ctx, userID, func(q Store) error {
		cur, err := current(ctx, q, userID, d)
		if err != nil {
			return err
		}
		if cur.Granted {
			if err := q.InsertConsentEvent(ctx, store.InsertConsentEventParams{
				UserID: userID, Consent: code, Version: uint32(cur.Version), Action: ActionRevoke, Source: source, Now: at, //nolint:gosec // G115: small catalog version
			}); err != nil {
				return fmt.Errorf("consent: withdraw %s: event: %w", code, err)
			}
		}
		if err := q.RevokeConsent(ctx, store.RevokeConsentParams{UserID: userID, Consent: code, Now: at}); err != nil {
			return fmt.Errorf("consent: withdraw %s: %w", code, err)
		}
		return nil
	})
}

// current is the projection row of d read inside a write (under the user lock).
func current(ctx context.Context, q Store, userID uint64, d Definition) (State, error) {
	row, err := q.GetConsent(ctx, store.GetConsentParams{UserID: userID, Consent: d.Code})
	if errors.Is(err, sql.ErrNoRows) {
		return State{Code: d.Code, Current: d.Version}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("consent: get %s: %w", d.Code, err)
	}
	return stateOf(d, row.Granted, row.Version, row.GrantedAt, row.RevokedAt), nil
}

// Pending are the mandatory consents the user has not accepted at the version in force (display order). Nothing is
// blocked while they are pending (H-18); the client shows the one-time acceptance sheet.
func (s *Service) Pending(ctx context.Context, userID uint64) ([]State, error) {
	rows, err := s.q.ListConsents(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("consent: pending: %w", err)
	}
	by := make(map[string]store.ListConsentsRow, len(rows))
	for _, r := range rows {
		by[r.Consent] = r
	}
	out := []State{}
	for _, d := range Mandatory() {
		st := State{Code: d.Code, Current: d.Version}
		if r, ok := by[d.Code]; ok {
			st = stateOf(d, r.Granted, r.Version, r.GrantedAt, r.RevokedAt)
		}
		if !st.Valid() {
			out = append(out, st)
		}
	}
	return out, nil
}
