package healthrecord

import (
	"sort"

	"github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// PregnancyEntry is one row of the pregnancies section.
type PregnancyEntry struct {
	ID        uint64 // manual entries only (0 = tracked)
	Source    string
	Outcome   string
	Date      civildate.Date // zero = unknown (always for an «ended» tracked pregnancy)
	BabyCount int            // 0 = not told
}

// Birth reports whether the entry is a birth.
func (e PregnancyEntry) Birth() bool {
	return e.Outcome == OutcomeVaginal || e.Outcome == OutcomeCesarean || e.Outcome == OutcomeBirth
}

// TrackedPregnancies is what the app itself knows: an active pregnancy, the postpartum birth and how many pregnancies
// ended in a loss (only the count: a loss is an «ended» pregnancy with no type, date or note).
type TrackedPregnancies struct {
	Active     bool
	Due        civildate.Date
	Birth      *store.GetRecordBirthRow
	LossCount  int
	ManualRows []store.HealthRecordPregnancy
}

// Entries are the section rows: an ongoing pregnancy first, then dated rows newest first, undated rows last.
func (t TrackedPregnancies) Entries() []PregnancyEntry {
	var out []PregnancyEntry
	if t.Active {
		out = append(out, PregnancyEntry{Source: SourceTracked, Outcome: OutcomeOngoing, Date: t.Due})
	}
	if b := t.Birth; b != nil {
		outcome := OutcomeBirth
		if b.DeliveryType.Valid && (b.DeliveryType.String == OutcomeVaginal || b.DeliveryType.String == OutcomeCesarean) {
			outcome = b.DeliveryType.String
		}
		out = append(out, PregnancyEntry{Source: SourceTracked, Outcome: outcome, Date: b.BirthDate, BabyCount: int(b.BabyCount)})
	}
	for range t.LossCount {
		out = append(out, PregnancyEntry{Source: SourceTracked, Outcome: OutcomeEnded})
	}
	for _, m := range t.ManualRows {
		e := PregnancyEntry{ID: m.ID, Source: SourceManual, Outcome: m.Outcome}
		if m.EndedOn.Valid {
			e.Date = m.EndedOn.Date
		}
		if m.BabyCount.Valid {
			e.BabyCount = int(m.BabyCount.Int16)
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.Outcome == OutcomeOngoing) != (b.Outcome == OutcomeOngoing) {
			return a.Outcome == OutcomeOngoing
		}
		if a.Date.IsZero() != b.Date.IsZero() {
			return !a.Date.IsZero()
		}
		return a.Date.After(b.Date)
	})
	return out
}

// JSON is the entry for audience (only a manual entry of the owner is editable and has an id).
func (e PregnancyEntry) JSON(aud Audience) *jsonx.OrderedMap {
	var id, date, babies any
	editable := e.Source == SourceManual && aud == AudienceOwner
	if editable {
		id = e.ID
	}
	if !e.Date.IsZero() {
		date = e.Date.String()
	}
	if e.BabyCount > 0 {
		babies = e.BabyCount
	}
	return jsonx.Obj("id", id, "source", e.Source, "outcome", e.Outcome, "date", date, "baby_count", babies,
		"editable", editable)
}

// ManualJSON is a manual entry as the write endpoints return it.
func ManualJSON(m store.HealthRecordPregnancy) *jsonx.OrderedMap {
	e := PregnancyEntry{ID: m.ID, Source: SourceManual, Outcome: m.Outcome}
	if m.EndedOn.Valid {
		e.Date = m.EndedOn.Date
	}
	if m.BabyCount.Valid {
		e.BabyCount = int(m.BabyCount.Int16)
	}
	return e.JSON(AudienceOwner)
}

// pregnanciesSection is {pregnancies_count, births_count, items}.
func pregnanciesSection(entries []PregnancyEntry, aud Audience) *jsonx.OrderedMap {
	births := 0
	items := make([]*jsonx.OrderedMap, 0, len(entries))
	for _, e := range entries {
		if e.Birth() {
			births++
		}
		items = append(items, e.JSON(aud))
	}
	return jsonx.Obj("pregnancies_count", len(entries), "births_count", births, "items", items)
}
