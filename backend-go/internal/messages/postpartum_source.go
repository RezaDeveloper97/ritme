package messages

import (
	"context"

	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum"
)

// PostpartumSignals reads the postpartum facts of a user on a day (*postpartum.Service).
type PostpartumSignals interface {
	SignalsOn(ctx context.Context, userID uint64, date civildate.Date) (postpartum.Signals, error)
}

// postpartumSource is a StoreSource that also implements manager.PostpartumSource (bloom B-N5-01): the birth, the
// day's recovery alerts and the due EPDS check. Read only when the manager runs the postpartum engine.
type postpartumSource struct {
	*StoreSource
	pp PostpartumSignals
}

// WithPostpartum returns src extended with the postpartum facts (src itself when pp is nil).
func WithPostpartum(src *StoreSource, pp PostpartumSignals) manager.Source {
	if pp == nil {
		return src
	}
	return &postpartumSource{StoreSource: src, pp: pp}
}

// Postpartum implements manager.PostpartumSource.
func (s *postpartumSource) Postpartum(ctx context.Context, date civildate.Date) (*manager.PostpartumState, error) {
	sig, err := s.pp.SignalsOn(ctx, s.userID, date)
	if err != nil {
		return nil, err
	}
	st := &manager.PostpartumState{BirthDate: sig.BirthDate, Alerts: sig.Recovery.Alerts()}
	if sig.Schedule != nil {
		st.CheckinDue = sig.Schedule.Due
	}
	return st, nil
}
