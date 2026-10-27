package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// UserStates is what the small engine consumers (companion, teen, checkups, cycle settings, challenges; T-M9-36,
// D-136) read: the prepared engine state of any user at the engine parameters of the user-local day today. A nil
// UserStates means CYCLE_ENGINE=v1 and the consumer keeps its legacy path.
type UserStates interface {
	ForUser(ctx context.Context, uid uint64, today civildate.Date) (*engine.State, error)
}

// RevisionReader is the part of the cycle store Source reads (*store.Queries satisfies it).
type RevisionReader interface {
	GetUserRevisions(ctx context.Context, userID uint64) (store.GetUserRevisionsRow, error)
}

// Source is the UserStates over a Provider. A consumer has no authenticated-request revision for the user it reads
// (a companion reads the owner, a parent the teen), so Source reads engine_rev from user_revisions (one indexed
// query); a user without a row is revision 0, the same as the auth query.
type Source struct {
	P    *Provider
	Revs RevisionReader
}

// ForUser returns uid's state prepared with the parameters valid on today.
func (s Source) ForUser(ctx context.Context, uid uint64, today civildate.Date) (*engine.State, error) {
	var rev uint64
	r, err := s.Revs.GetUserRevisions(ctx, uid)
	switch {
	case err == nil:
		rev = r.EngineRev
	case !errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("state: revisions: %w", err)
	}
	return s.P.For(ctx, uid, rev, Params{Engine: engineparams.For(today), FP: engineparams.Fingerprint(today)})
}
