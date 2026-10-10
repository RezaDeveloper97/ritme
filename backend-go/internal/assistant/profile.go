package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// RecordProfile builds the shared profile from the health record (share audience: no free-text notes, no name):
// age and life_mode from the record's person, the other keys from the sections of the same name.
func RecordProfile(svc *healthrecord.Service) ProfileBuilder {
	return func(ctx context.Context, userID uint64, keys []string, today civildate.Date, locale string) (map[string]any, error) {
		sections := []string{} // non-nil: Build builds only these
		for _, k := range keys {
			if k != ProfileAge && k != ProfileLifeMode {
				sections = append(sections, k)
			}
		}
		rec, err := svc.Build(ctx, userID, healthrecord.AudienceShare, healthrecord.Options{Today: today, Locale: locale,
			DefaultLocale: locale, Sections: sections})
		if err != nil {
			return nil, fmt.Errorf("assistant: health record: %w", err)
		}
		out := map[string]any{}
		var person map[string]any
		if raw, err := json.Marshal(rec.Person); err == nil {
			_ = json.Unmarshal(raw, &person)
		}
		for _, k := range []string{ProfileAge, ProfileLifeMode} {
			if slices.Contains(keys, k) {
				out[k] = person[k]
			}
		}
		for _, sec := range rec.Sections {
			if !slices.Contains(sections, sec.Key) {
				continue
			}
			if sec.Empty {
				out[sec.Key] = nil
				continue
			}
			var v any
			if raw, err := json.Marshal(sec.Data); err == nil {
				_ = json.Unmarshal(raw, &v)
			}
			if sec.Key == ProfilePregnancies {
				v = withoutLosses(v)
			}
			out[sec.Key] = v
		}
		return out, nil
	}
}

// withoutLosses drops every pregnancy that did not end in a birth (tracked losses and manual «ended» rows) from the
// pregnancies section and recounts it: a loss is never sent to the model, not even as a count. nil when nothing is
// left.
func withoutLosses(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	items, _ := m["items"].([]any)
	kept, births := []any{}, 0
	for _, it := range items {
		e, ok := it.(map[string]any)
		if !ok {
			continue
		}
		switch e["outcome"] {
		case healthrecord.OutcomeVaginal, healthrecord.OutcomeCesarean, healthrecord.OutcomeBirth:
			births++
		case healthrecord.OutcomeOngoing:
		default:
			continue
		}
		kept = append(kept, map[string]any{"outcome": e["outcome"], "date": e["date"], "baby_count": e["baby_count"]})
	}
	if len(kept) == 0 {
		return nil
	}
	return map[string]any{"pregnancies_count": len(kept), "births_count": births, "items": kept}
}
