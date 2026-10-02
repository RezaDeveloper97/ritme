// Package teen is teen mode (CB-TEEN-01, roadmap/E09-teen; boards nbl_Teen_Onb, nbl_Teen_Home, nbl_Teen_Parent), built
// on top of bloom's teen life-stage mode (B-N2-03: `user_life_profiles.life_mode = teen`, no fertility copy) and
// bloom's companion system (B-N4-02: invite / accept / revoke, grants, audit).
//
//   - Profile: the Teen_Onb answers (age band, first period) and the note she writes for her parent (teen_profiles).
//   - Kit: her school emergency-kit checklist; the items are admin catalog content (`teen_kit_items`), the ticks are
//     hers (teen_kit_checks).
//   - Content: readiness text, signs, «when to talk» and the FAQ are admin catalog content (`teen_signs`, `teen_faq`,
//     fa + en, needs_review) — age-appropriate, no fertility or sexual-health copy.
//   - Parent card: a parent link (companion type `parent`, only a teen-mode owner may invite one) carries only the
//     teen sections at level view. The parent's single read is GET /teen/linked: per active parent link a card with
//     the granted parts only (next period as a week bucket, kit ready, her note), each read audited before it is
//     built. Nothing is shared by default and a parent can never write.
//   - Commercial: teen accounts get no shop, ads / banners, Plus upsell or commercial recommendations (AllowsFor;
//     enforced server-side on GET /banners and the home's Plus trial banner, and exposed as `allows` for clients).
package teen

import (
	"errors"
	"slices"

	"github.com/ritme/backend-go/internal/enums"
)

// AgeBand is «چند سالته؟».
type AgeBand string

// Age bands, in board order.
const (
	Age10to12 AgeBand = "10_12"
	Age13to15 AgeBand = "13_15"
	Age16to17 AgeBand = "16_17"
)

// AgeBands are the accepted age bands, in board order.
var AgeBands = []AgeBand{Age10to12, Age13to15, Age16to17}

// Valid reports whether a is a known age band.
func (a AgeBand) Valid() bool { return slices.Contains(AgeBands, a) }

// Menarche is «پریود شده‌ای؟».
type Menarche string

// First-period answers, in board order.
const (
	MenarcheNotYet  Menarche = "not_yet"
	MenarcheUnder1y Menarche = "under_1y"
	MenarcheOver1y  Menarche = "over_1y"
)

// Menarches are the accepted answers, in board order.
var Menarches = []Menarche{MenarcheNotYet, MenarcheUnder1y, MenarcheOver1y}

// Valid reports whether m is a known answer.
func (m Menarche) Valid() bool { return slices.Contains(Menarches, m) }

// PeriodWeek is the only cycle datum a parent may see: the next period as a week bucket (Saturday-start weeks,
// Tehran). Never a date, a cycle day, a phase or lateness.
type PeriodWeek string

// Week buckets.
const (
	WeekThis    PeriodWeek = "this_week"
	WeekNext    PeriodWeek = "next_week"
	WeekLater   PeriodWeek = "later"
	WeekUnknown PeriodWeek = "unknown"
)

// Catalog groups (admin content, docs/canvas-build/catalog.md).
const (
	GroupSigns = "teen_signs"
	GroupFAQ   = "teen_faq"
	GroupKit   = "teen_kit_items"
	// Audience is the catalog audience code of teen items.
	Audience = "teen"
)

// Kinds of `teen_signs` items (meta.kind).
const (
	KindSign     = "sign"
	KindEstimate = "estimate"
	KindTalk     = "talk"
)

// MaxParentNote caps the note a teen writes for her parent (teen_profiles.parent_note).
const MaxParentNote = 280

// Errors.
var (
	// ErrProfileRequired: the parent note needs the onboarding answers first.
	ErrProfileRequired = errors.New("teen: profile required")
	// ErrUnknownKitItem: not an active `teen_kit_items` code.
	ErrUnknownKitItem = errors.New("teen: unknown kit item")
)

// Allows are the commercial surfaces a user may see. Teen accounts (minors) get none of them.
type Allows struct {
	Shop                      bool
	Banners                   bool
	Ads                       bool
	PlusUpsell                bool
	CommercialRecommendations bool
}

// AllowsCommercial reports whether a stored life-stage mode may see commercial content (false for teen).
func AllowsCommercial(mode enums.LifeMode) bool { return mode != enums.LifeModeTeen }

// AllowsFor is the commercial flag set of a stored life-stage mode.
func AllowsFor(mode enums.LifeMode) Allows {
	ok := AllowsCommercial(mode)
	return Allows{Shop: ok, Banners: ok, Ads: ok, PlusUpsell: ok, CommercialRecommendations: ok}
}
