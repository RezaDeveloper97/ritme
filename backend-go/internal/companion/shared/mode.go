package shared

import (
	"context"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// viewScope is what the cycle and symptoms views may reveal for one link: the owner's effective life mode
// (enums.ResolveLifeMode: an active pregnancy wins) and whether the link holds the pregnancy grant.
type viewScope struct {
	mode      enums.LifeMode
	pregnancy bool // the link may read the pregnancy section
}

// scope resolves the owner's life mode for the link.
func (r *Reader) scope(ctx context.Context, link companion.Link) (viewScope, error) {
	mode, err := r.healthlg.LifeMode(ctx, link.OwnerID)
	if err != nil {
		return viewScope{}, err
	}
	return viewScope{mode: enums.LifeMode(mode), pregnancy: link.Grants.Of(companion.SectionPregnancy).CanRead()}, nil
}

// pregnant reports whether the owner's effective mode is pregnancy — the only mode the pregnancy grant covers.
// Postpartum is not pregnancy (T-M9-02, FAM-M3): the grant never opens what a mother logs after the birth.
func (v viewScope) pregnant() bool { return v.mode == enums.LifeModePregnancy }

// pregnancyShared reports whether the link may see the owner's pregnancy: the grant and a pregnancy right now.
func (v viewScope) pregnancyShared() bool { return v.pregnancy && v.pregnant() }

// neutralCycle reports whether the cycle view must say nothing: a pregnant owner whose link has no pregnancy grant
// (her cycle numbers would reveal it), every postpartum owner (no grant covers that mode; FAM-M3) and every
// menopause owner (the cycle view means nothing there and would reveal the life stage).
func (v viewScope) neutralCycle() bool {
	switch v.mode {
	case enums.LifeModeMenopause, enums.LifeModePostpartum:
		return true
	case enums.LifeModePregnancy:
		return !v.pregnancy
	}
	return false
}

// symptomModes are the log modes whose options the symptoms view may pass on: always the cycle mode; with the
// pregnancy grant during a pregnancy also the pregnancy mode. Mode-only options of any other mode never leave
// (stitches or the weekly check-in after birth, the menopause items).
func (v viewScope) symptomModes() []string {
	out := []string{taxonomy.ModeCycle}
	if v.pregnancyShared() {
		out = append(out, taxonomy.ModePregnancy)
	}
	return out
}

// neutralCycle is the cycle view that tells nothing: no data, every number null.
func neutralCycle(today civildate.Date) *jsonx.OrderedMap {
	return jsonx.Obj(
		"date", today,
		"has_data", false,
		"cycle_day", nil,
		"cycle_length", nil,
		"main_phase", nil,
		"days_to_period", nil,
		"predicted_next_period_start", nil,
		"confidence", nil,
	)
}

// sharedPhase is the phase a companion sees: for an owner whose mode never gets fertility copy (teen; CMP-M1) the
// fertile phase becomes its neighbour, follicular, so neither the card nor the tips name a fertile window.
func sharedPhase(p enums.MainPhase, noFertility bool) enums.MainPhase {
	if noFertility && p == enums.MainPhaseFertile {
		return enums.MainPhaseFollicular
	}
	return p
}

// sharedEntry reports whether a day-log entry may reach the companion: its category and param must show in one of
// modes and, for option-valued params, its option (single: the value, multi / items: the item) must be offered in
// one of modes. Unknown params and options are dropped; the owner's custom items of a param that hosts them pass.
func sharedEntry(e taxonomy.Entry, modes []string) bool {
	c, ok := taxonomy.CategoryByCode(e.Category)
	if !ok {
		return false
	}
	p, ok := c.Param(e.Param)
	if !ok || !slices.ContainsFunc(modes, func(m string) bool { return c.Available(p, m) }) {
		return false
	}
	var code string
	switch p.Type {
	case taxonomy.Single:
		code = e.Code.String
	case taxonomy.Multi, taxonomy.Items, taxonomy.TextItems:
		code = e.Item
	default:
		return true
	}
	o, ok := p.Option(code)
	if !ok {
		return p.Dynamic || (p.Custom && strings.HasPrefix(code, taxonomy.CustomItemPrefix))
	}
	return o.Modes == nil || slices.ContainsFunc(modes, func(m string) bool { return slices.Contains(o.Modes, m) })
}
