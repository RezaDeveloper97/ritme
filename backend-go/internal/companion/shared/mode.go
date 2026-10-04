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

// MaxLateDays is the lateness the cycle view still shows (the cycle engine's own cap, resolver maxLateDays). Past it
// days_late and cycle_day are null for every companion: a long lateness says more about the owner (a pregnancy she
// keeps private) than about her cycle (CMP-H1).
const MaxLateDays = 14

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

// pregnancyMode reports whether the mode is pregnancy or postpartum (what the pregnancy grant covers).
func (v viewScope) pregnancyMode() bool {
	return v.mode == enums.LifeModePregnancy || v.mode == enums.LifeModePostpartum
}

// neutralCycle reports whether the cycle view must say nothing: a pregnancy / postpartum owner whose link has no
// pregnancy grant (her cycle numbers would reveal it), and every menopause owner (the cycle view means nothing there
// and would reveal the life stage).
func (v viewScope) neutralCycle() bool {
	return v.mode == enums.LifeModeMenopause || (v.pregnancyMode() && !v.pregnancy)
}

// symptomModes are the log modes whose options the symptoms view may pass on: always the cycle mode; with the
// pregnancy grant also the owner's pregnancy / postpartum mode. Mode-only options of any other mode never leave
// (leg cramps in pregnancy, stitches after birth, the menopause items).
func (v viewScope) symptomModes() []string {
	out := []string{taxonomy.ModeCycle}
	if v.pregnancy && v.pregnancyMode() {
		out = append(out, string(v.mode))
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
		"days_late", nil,
		"predicted_next_period_start", nil,
		"confidence", nil,
	)
}

// tooLate reports whether a lateness is past MaxLateDays.
func tooLate(daysLate *int) bool { return daysLate != nil && *daysLate > MaxLateDays }

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
