package profile

import (
	"context"

	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/profile/store"
)

// WithConsentLedger routes the B-N1-12 toggle writes of PUT /profile/consents through the consent ledger (T-M10-23,
// D-174): a grant or withdrawal there becomes a consent_events row (source settings) written with the user_consents
// projection in one transaction, exactly like PUT /consents/{code}. Reads and the support reports stay on q.
// UpdateConsents has already checked that an AI grant names the version in force, so svc accepts what it is given.
func WithConsentLedger(q PrivacyStore, svc *consent.Service) PrivacyStore {
	return ledgerStore{PrivacyStore: q, svc: svc}
}

type ledgerStore struct {
	PrivacyStore
	svc *consent.Service
}

func (l ledgerStore) GrantUserConsent(ctx context.Context, a store.GrantUserConsentParams) error {
	return l.svc.AcceptFrom(ctx, a.UserID, a.Consent, int(a.Version.Int16), consent.SourceSettings, a.Now.Time)
}

func (l ledgerStore) RevokeUserConsent(ctx context.Context, a store.RevokeUserConsentParams) error {
	return l.svc.WithdrawFrom(ctx, a.UserID, a.Consent, consent.SourceSettings, a.Now.Time)
}
