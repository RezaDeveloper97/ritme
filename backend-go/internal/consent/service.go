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
}

// Service reads and writes the user's consents.
type Service struct{ q Store }

// NewService wires the service on db.
func NewService(db *sql.DB) *Service { return &Service{q: store.New(db)} }

// NewServiceWith wires the service on a store (tests).
func NewServiceWith(q Store) *Service { return &Service{q: q} }

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

// List is the user's state for every catalog consent, in display order.
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

// Accept records the user's acceptance of version of code at now. Only the version in force can be accepted
// (ErrStaleVersion otherwise: the client showed an old text).
func (s *Service) Accept(ctx context.Context, userID uint64, code string, version int, now time.Time) error {
	d, ok := Lookup(code)
	if !ok {
		return ErrUnknown
	}
	if version != d.Version {
		return ErrStaleVersion
	}
	err := s.q.GrantConsent(ctx, store.GrantConsentParams{
		UserID: userID, Consent: code, Version: sql.NullInt16{Int16: int16(version), Valid: true}, //nolint:gosec // G115: small catalog version
		Now: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("consent: accept %s: %w", code, err)
	}
	return nil
}

// Withdraw records the user's withdrawal of code at now.
func (s *Service) Withdraw(ctx context.Context, userID uint64, code string, now time.Time) error {
	if _, ok := Lookup(code); !ok {
		return ErrUnknown
	}
	if err := s.q.RevokeConsent(ctx, store.RevokeConsentParams{UserID: userID, Consent: code, Now: sql.NullTime{Time: now, Valid: true}}); err != nil {
		return fmt.Errorf("consent: withdraw %s: %w", code, err)
	}
	return nil
}
