package menopause

import (
	"context"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Log taxonomy slots the API reads (bloom B-N3-01 + the CB-MENO-01 items).
const (
	SlotHotFlashes  = "symptoms.general.hot_flashes"
	SlotNightSweats = "symptoms.general.night_sweats"
	SlotFatigue     = "symptoms.general.fatigue"

	paramBleedingPresence = "bleeding.presence"
	paramBleedingFlow     = "bleeding.flow"
	paramBleedingSpotting = "bleeding.spotting"
	paramTriggers         = "menopause.triggers"
	presenceNone          = "none"
)

// LogDay is one logged day: the raw entries and bloom's analysis view of them.
type LogDay struct {
	Entries []taxonomy.Entry
	View    *analysis.Day
}

// Bleeding: any bleeding or spotting logged (bleeding.presence ≠ none, a bleeding.flow, or bleeding.spotting = yes)
// — the postmenopausal_bleeding rule (docs/canvas-build/menopause.md §2).
func (d LogDay) Bleeding() bool {
	for _, e := range d.Entries {
		switch e.ParamKey() {
		case paramBleedingPresence:
			if e.Code.Valid && e.Code.String != "" && e.Code.String != presenceNone {
				return true
			}
		case paramBleedingFlow:
			if e.Code.Valid && e.Code.String != "" {
				return true
			}
		case paramBleedingSpotting:
			if e.Code.String == taxonomy.Yes {
				return true
			}
		}
	}
	return false
}

// Triggers are the menopause.triggers logged that day.
func (d LogDay) Triggers() []string {
	var out []string
	for _, e := range d.Entries {
		if e.ParamKey() == paramTriggers && e.Item != "" {
			out = append(out, e.Item)
		}
	}
	return out
}

// Has reports whether a symptom slot was logged above «no» (bloom's symptom weight > 0).
func (d LogDay) Has(slot string) bool { return d.View != nil && d.View.Symptoms[slot] > 0 }

// Level is the level code of a symptom slot ("" when not logged).
func (d LogDay) Level(slot string) string {
	for _, e := range d.Entries {
		if e.Slot() == slot && e.Code.Valid {
			return e.Code.String
		}
	}
	return ""
}

// logDays are the user's logged days in [from, to] (days without entries are absent).
func (s *Service) logDays(ctx context.Context, userID uint64, from, to civildate.Date) (map[civildate.Date]LogDay, error) {
	rows, err := s.logs.Range(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	in := make([]analysis.DayEntries, len(rows))
	for i, r := range rows {
		in[i] = analysis.DayEntries{Date: r.Date, Entries: r.Entries}
	}
	views := analysis.BuildDays(in)
	out := make(map[civildate.Date]LogDay, len(rows))
	for _, r := range rows {
		out[r.Date] = LogDay{Entries: r.Entries, View: views[r.Date]}
	}
	return out, nil
}
