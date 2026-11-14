package companion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/companion/store"
)

// «ریتمی همراه» v2: what the owner wants from a link (goal), which coarse day signals she consents to share (one
// consent per signal, all off by default) and which kinds of suggestions the companion should see (action_prefs).
// Stored on the companions row (goose 00117). Nothing here is health data: the signals are scope flags; the day
// values are derived per request by the companion home and never stored.

// Goal is what the owner wants the link to help with.
type Goal string

// Goals (the picker order of the design). Unknown is the default and what an old row (NULL) reads as.
const (
	GoalCycle       Goal = "cycle"
	GoalSupport     Goal = "support"
	GoalLowEnergy   Goal = "low_energy"
	GoalQualityTime Goal = "quality_time"
	GoalInfo        Goal = "info"
	GoalUnknown     Goal = "unknown"
)

// Goals are the accepted goals, in picker order.
var Goals = []Goal{GoalCycle, GoalSupport, GoalLowEnergy, GoalQualityTime, GoalInfo, GoalUnknown}

// Valid reports whether g is a known goal.
func (g Goal) Valid() bool { return slices.Contains(Goals, g) }

// orUnknown is g, or unknown for an empty or unknown code (the wire never carries anything else).
func (g Goal) orUnknown() Goal {
	if g.Valid() {
		return g
	}
	return GoalUnknown
}

// Signal is one coarse day signal the owner may share.
type Signal string

// Signals, in card order.
const (
	SignalPain   Signal = "pain"
	SignalMood   Signal = "mood"
	SignalEnergy Signal = "energy"
	SignalSleep  Signal = "sleep"
)

// SignalOrder lists every signal, in card order (JSON key order).
var SignalOrder = []Signal{SignalPain, SignalMood, SignalEnergy, SignalSleep}

// Valid reports whether s is a known signal.
func (s Signal) Valid() bool { return slices.Contains(SignalOrder, s) }

// SignalScope is the owner's consent per signal (missing = off).
type SignalScope map[Signal]bool

// On reports whether the owner shares s.
func (sc SignalScope) On(s Signal) bool { return sc[s] }

// ActionPref is a kind of suggestion card the companion sees.
type ActionPref string

// Action kinds, in picker order.
const (
	ActionFood ActionPref = "food"
	ActionGift ActionPref = "gift"
	ActionTime ActionPref = "time"
	ActionCalm ActionPref = "calm"
	ActionHelp ActionPref = "help"
)

// ActionPrefs are the accepted action kinds, in picker order.
var ActionPrefs = []ActionPref{ActionFood, ActionGift, ActionTime, ActionCalm, ActionHelp}

// MaxActionPrefs caps the list (every kind once).
const MaxActionPrefs = 5

// Valid reports whether a is a known action kind.
func (a ActionPref) Valid() bool { return slices.Contains(ActionPrefs, a) }

// Preferences are the owner's per-link «ریتمی همراه» settings.
type Preferences struct {
	Goal    Goal
	Signals SignalScope
	Actions []ActionPref // ordered as chosen, distinct
}

// DefaultPreferences are what a new link (and an old row) has: goal unknown, every signal off, no action kinds.
func DefaultPreferences() Preferences {
	return Preferences{Goal: GoalUnknown, Signals: SignalScope{}, Actions: []ActionPref{}}
}

// PreferencesPatch is a partial update: nil fields stay as they are; a present Signals map changes only its keys.
type PreferencesPatch struct {
	Goal    *Goal
	Signals SignalScope
	Actions *[]ActionPref
}

// Preference validation errors (the handlers answer 422 on the field).
var (
	ErrInvalidGoal    = errors.New("companion: invalid goal")
	ErrInvalidSignal  = errors.New("companion: invalid signal")
	ErrInvalidActions = errors.New("companion: invalid action prefs")
)

// NormalizeActions validates action kinds and dedupes them keeping the first occurrence's order. More than
// MaxActionPrefs distinct kinds cannot happen with a valid set; the raw cap is the caller's (validation max:5).
func NormalizeActions(in []ActionPref) ([]ActionPref, error) {
	out := make([]ActionPref, 0, len(in))
	for _, a := range in {
		if !a.Valid() {
			return nil, ErrInvalidActions
		}
		if !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	if len(out) > MaxActionPrefs {
		return nil, ErrInvalidActions
	}
	return out, nil
}

// Validate checks and normalises the preferences in place (empty goal → unknown, actions deduped).
func (p *Preferences) Validate() error {
	if p.Goal == "" {
		p.Goal = GoalUnknown
	}
	if !p.Goal.Valid() {
		return ErrInvalidGoal
	}
	for s := range p.Signals {
		if !s.Valid() {
			return ErrInvalidSignal
		}
	}
	if p.Signals == nil {
		p.Signals = SignalScope{}
	}
	acts, err := NormalizeActions(p.Actions)
	if err != nil {
		return err
	}
	p.Actions = acts
	return nil
}

// Apply merges a patch onto p and validates the result.
func (p Preferences) Apply(patch PreferencesPatch) (Preferences, error) {
	out := Preferences{Goal: p.Goal, Signals: SignalScope{}, Actions: slices.Clone(p.Actions)}
	for s, on := range p.Signals {
		out.Signals[s] = on
	}
	if patch.Goal != nil {
		out.Goal = *patch.Goal
	}
	for s, on := range patch.Signals {
		out.Signals[s] = on
	}
	if patch.Actions != nil {
		out.Actions = *patch.Actions
	}
	return out, out.Validate()
}

// Empty reports whether the patch changes nothing.
func (patch PreferencesPatch) Empty() bool {
	return patch.Goal == nil && len(patch.Signals) == 0 && patch.Actions == nil
}

// preferencesOf reads the stored columns (NULL / unknown codes read as the defaults: a code dropped from the enum
// never breaks a read).
func preferencesOf(c store.Companion) Preferences {
	p := DefaultPreferences()
	if g := Goal(c.Goal.String); c.Goal.Valid && g.Valid() {
		p.Goal = g
	}
	p.Signals = SignalScope{SignalPain: c.SharePain, SignalMood: c.ShareMood, SignalEnergy: c.ShareEnergy,
		SignalSleep: c.ShareSleep}
	for _, code := range strings.Split(c.ActionPrefs.String, ",") {
		if a := ActionPref(strings.TrimSpace(code)); a.Valid() && !slices.Contains(p.Actions, a) {
			p.Actions = append(p.Actions, a)
		}
	}
	return p
}

// actionsColumn is the stored spelling of the action kinds (comma-separated, ordered; NULL when none).
func actionsColumn(acts []ActionPref) sql.NullString {
	codes := make([]string, len(acts))
	for i, a := range acts {
		codes[i] = string(a)
	}
	return nstr(strings.Join(codes, ","))
}

// goalColumn stores unknown as NULL, so the default and an old row look the same.
func goalColumn(g Goal) sql.NullString {
	if g == GoalUnknown {
		return sql.NullString{}
	}
	return nstr(string(g))
}

// UpdatePreferences merges patch into the preferences of one of ownerID's invited or active links (ErrNotFound for
// anyone else's, a revoked one or a bad id — no existence oracle) and records a grants_changed audit row without values
// (the signal flags are sharing consents, so the owner's trail shows the change under the label she already knows; a
// new action code would be skipped by the clients' audit list and need new copy in every language).
func (s *Service) UpdatePreferences(ctx context.Context, ownerID, companionID uint64, patch PreferencesPatch) (Link, error) {
	if patch.Empty() {
		return s.OwnerLink(ctx, ownerID, companionID) // nothing to write, nothing to audit
	}
	now := s.now()
	var link Link
	err := s.inTx(ctx, func(q *store.Queries) error {
		if err := lockOwner(ctx, q, ownerID); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, companionID)
		if err != nil || c.OwnerID != ownerID || Status(c.Status) == StatusRevoked {
			return notFound(err)
		}
		next, err := preferencesOf(c).Apply(patch)
		if err != nil {
			return err
		}
		n, err := q.UpdateCompanionPreferences(ctx, store.UpdateCompanionPreferencesParams{
			Goal: goalColumn(next.Goal), SharePain: next.Signals.On(SignalPain), ShareMood: next.Signals.On(SignalMood),
			ShareEnergy: next.Signals.On(SignalEnergy), ShareSleep: next.Signals.On(SignalSleep),
			ActionPrefs: actionsColumn(next.Actions), Now: nt(now), ID: c.ID, OwnerID: ownerID,
		})
		if err != nil {
			return fmt.Errorf("companion: preferences: %w", err)
		}
		if n == 0 {
			return ErrNotFound
		}
		if err := audit(ctx, q, ownerID, ownerID, c.ID, "", ActionGrantsChanged, now); err != nil {
			return err
		}
		link, err = loadLink(ctx, q, c.ID)
		return err
	})
	return link, err
}
