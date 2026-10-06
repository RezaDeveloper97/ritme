package labs

import (
	"context"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// RecordAttentionNames caps the out-of-range markers named on one health-record lab row («ویتامین D پایین»).
const RecordAttentionNames = 3

// RecordLabs is the health record's lab rows (bloom B-N6-03, internal/healthrecord): the user's newest `limit` ready
// labs, newest sheet first, with their counts and the names + states of the markers that need attention. Read-only;
// every query is scoped by userID.
func (s *Service) RecordLabs(ctx context.Context, userID uint64, locale, def string, limit int) ([]*jsonx.OrderedMap, error) {
	labs, err := s.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	cat, err := s.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.UserMarkers(ctx, userID)
	if err != nil {
		return nil, err
	}
	byLab := map[uint64][]Evaluated{}
	for _, r := range rows {
		byLab[r.LabID] = append(byLab[r.LabID], evaluate(rowMarker(r), cat))
	}
	l := loc{Locale: locale, Default: def}
	out := []*jsonx.OrderedMap{}
	for _, lab := range labs {
		if lab.Status != StatusReady {
			continue
		}
		if len(out) >= limit {
			break
		}
		evals := byLab[lab.ID]
		c := countStates(evals)
		attention := []*jsonx.OrderedMap{}
		for _, e := range evals {
			if !Attention(e.State) || len(attention) >= RecordAttentionNames {
				continue
			}
			attention = append(attention, jsonx.Obj("name", l.name(e), "state", e.State,
				"state_label", T("states."+e.State, l.Locale, nil)))
		}
		out = append(out, jsonx.Obj(
			"id", lab.ID,
			"category", lab.Category,
			"title", l.title(lab),
			"date", labDate(lab.TakenOn, lab.CreatedAt).String(),
			"marker_count", c.Total,
			"attention_count", c.Attention,
			"all_normal", c.Total > 0 && c.Attention == 0,
			"attention", attention,
		))
	}
	return out, nil
}
