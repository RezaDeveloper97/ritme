package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/assistant/store"
	"github.com/ritme/backend-go/internal/consent"
)

// SaveShared saves the user's per-item profile switches («دستیار چه چیزهایی را ببیند») together with their
// consent-ledger evidence (T-M10-23, D-174): consent assistant_profile_items, source feature, scope {items} = the
// items shared after the change (a grant), or {items} = the items no longer shared when none is left (a revoke).
// An unchanged choice writes no event. Evidence only: the assistant_profile consent stays the gate. SetShared calls
// it; the choice and its event commit together.
func (s *Service) SaveShared(ctx context.Context, userID uint64, keys []string, now time.Time) ([]string, error) {
	keys = orderedKeys(keys)
	before, err := s.Shared(ctx, userID)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(keys)
	tx, err := s.o.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("assistant: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.q.WithTx(tx).UpsertProfileSettings(ctx, store.UpsertProfileSettingsParams{UserID: userID, Shared: string(raw), Now: nt(now)}); err != nil {
		return nil, fmt.Errorf("assistant: save profile settings: %w", err)
	}
	if !slices.Equal(before, keys) {
		ev := consent.Scoped{UserID: userID, Consent: consent.AssistantProfileItems, Action: consent.ActionGrant,
			Source: consent.SourceFeature, Scope: map[string]any{"items": keys}, At: now}
		if len(keys) == 0 {
			ev.Action, ev.Scope = consent.ActionRevoke, map[string]any{"items": before}
		}
		if err := consent.AppendScoped(ctx, tx, ev); err != nil {
			return nil, fmt.Errorf("assistant: consent evidence: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("assistant: commit: %w", err)
	}
	return keys, nil
}
