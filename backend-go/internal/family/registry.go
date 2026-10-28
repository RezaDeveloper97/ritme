// Package family is the v6 family access core (engine v6 M11, T-M11-05; SPEC §1.7–§1.8; docs/engine-v6/family.md;
// DECISIONS.md F-01 … F-23, D-193, D-203, D-204): the domain registry, the pure Decide (check(actor, subject,
// domain, action) → allow | propose | deny), the batched snapshot loader with the lazy mode epoch (SyncMode), the
// grant-change service (Grants.Apply), the in-transaction recheck (RecheckTx) and the coalesced read audit.
//
// Deny by default: an unknown domain, role, level or mode, a missing link or owner state, a grant without the
// owner's current epoch — every gap answers deny. Only the rows of the registry below can ever allow.
package family

import (
	"time"

	"github.com/ritme/backend-go/internal/enums"
)

// Level is what a grant allows on a domain, ordered off < view < nudge < write (family.md §1).
type Level uint8

// Levels.
const (
	LevelOff Level = iota
	LevelView
	LevelNudge
	LevelWrite
)

var levelCodes = [...]string{LevelOff: "off", LevelView: "view", LevelNudge: "nudge", LevelWrite: "write"}

// String is the stored code.
func (l Level) String() string {
	if int(l) < len(levelCodes) {
		return levelCodes[l]
	}
	return "off"
}

// ParseLevel reads a stored level code. Unknown codes (and the legacy `none`) are off.
func ParseLevel(code string) (Level, bool) {
	for i, c := range levelCodes {
		if c == code {
			return Level(i), true
		}
	}
	return LevelOff, false
}

// Domain is a part of the owner's (or the household's) data a grant covers.
type Domain uint8

// Grantable domains (family.md §2). Children are not a link grant: they go through guardianship (child_guardians).
// The teen sections are the legacy `parent` (teen_parent) card, outside the v6 matrix (F-17), kept unchanged.
const (
	DomainNone Domain = iota
	DomainCyclePhase
	DomainPregnancy
	DomainAppointments
	DomainFetalActivity
	DomainVitals
	DomainMeds
	DomainUrgentAlert
	DomainHousehold
	DomainTTCFertility
	DomainTeenPeriodWeek
	DomainTeenKit
	DomainTeenNotes
	// DomainChildren is the child subject's domain (guardianship only, never a link grant).
	DomainChildren
	numDomains
)

// Never-shared categories (SPEC §1.7, family.md §4): valid Domain values with no registry row, so Decide can be asked
// about them (and the property test proves it never allows). The legacy `symptoms` section is one of them (F-03).
const (
	NeverPrivateNotes Domain = iota + 64
	NeverMood
	NeverSex
	NeverLabResults
	NeverChats
	NeverCycleHistory
	NeverFertilityRaw
	NeverSymptoms
	neverEnd
)

// NeverShared lists the never-shared categories.
var NeverShared = []Domain{NeverPrivateNotes, NeverMood, NeverSex, NeverLabResults, NeverChats, NeverCycleHistory,
	NeverFertilityRaw, NeverSymptoms}

var domainCodes = [numDomains]string{
	DomainCyclePhase: "cycle_phase", DomainPregnancy: "pregnancy", DomainAppointments: "appointments",
	DomainFetalActivity: "fetal_activity", DomainVitals: "vitals", DomainMeds: "meds", DomainUrgentAlert: "urgent_alert",
	DomainHousehold: "household", DomainTTCFertility: "ttc_fertility", DomainTeenPeriodWeek: "teen_period_week",
	DomainTeenKit: "teen_kit", DomainTeenNotes: "teen_notes", DomainChildren: "children",
}

// String is the stored code ("" for none / never-shared).
func (d Domain) String() string {
	if d < numDomains {
		return domainCodes[d]
	}
	return ""
}

// ParseDomain reads a stored grant domain code (children and never-shared categories are not grant codes).
func ParseDomain(code string) (Domain, bool) {
	for i, c := range domainCodes {
		if c != "" && c == code && Domain(i) != DomainChildren {
			return Domain(i), true
		}
	}
	return DomainNone, false
}

// GrantDomains are the link-grantable domains in display order (family.md §2; teen sections last).
var GrantDomains = []Domain{DomainCyclePhase, DomainPregnancy, DomainAppointments, DomainFetalActivity, DomainVitals,
	DomainMeds, DomainUrgentAlert, DomainHousehold, DomainTTCFertility, DomainTeenPeriodWeek, DomainTeenKit, DomainTeenNotes}

// Role is the member's relation; it only seeds defaults and caps (family.md §2 Roles).
type Role uint8

// Roles. RoleNone is an unknown / missing role: nothing is allowed.
const (
	RoleNone Role = iota
	RolePartner
	RoleOtherParent
	RoleCarer
	RoleFriend
	RoleTeenParent
	numRoles
)

var roleCodes = [numRoles]string{RolePartner: "partner", RoleOtherParent: "other_parent", RoleCarer: "carer",
	RoleFriend: "friend", RoleTeenParent: "teen_parent"}

// String is the stored code.
func (r Role) String() string {
	if r < numRoles {
		return roleCodes[r]
	}
	return ""
}

// ParseRole reads companions.role; a NULL role falls back to the legacy type the way the 00106 backfill does
// (partner / spouse → partner, parent → teen_parent; F-17). Anything else is RoleNone.
func ParseRole(role, legacyType string) Role {
	if role != "" {
		for i, c := range roleCodes {
			if c != "" && c == role {
				return Role(i)
			}
		}
		return RoleNone
	}
	switch legacyType {
	case "partner", "spouse":
		return RolePartner
	case "parent":
		return RoleTeenParent
	}
	return RoleNone
}

// modeSet is a bit set of owner effective modes.
type modeSet uint16

const (
	mCycle modeSet = 1 << iota
	mTTC
	mPregnancy
	mPostpartum
	mMenopause
	mTeen
)

// modeAny is every adult mode. A teen-mode owner (a minor) shares nothing outside the teen card (B-N4-08b, CMP-M1).
const modeAny = mCycle | mTTC | mPregnancy | mPostpartum | mMenopause

// Mode is the owner's effective life mode as a bit (0 = unknown → every mode-scoped check denies).
type Mode = modeSet

// ModeOf maps an effective life mode (enums.ResolveLifeMode) to its bit; unknown → 0.
func ModeOf(m string) Mode {
	switch enums.LifeMode(m) {
	case enums.LifeModeCycle:
		return mCycle
	case enums.LifeModeTTC:
		return mTTC
	case enums.LifeModePregnancy:
		return mPregnancy
	case enums.LifeModePostpartum:
		return mPostpartum
	case enums.LifeModeMenopause:
		return mMenopause
	case enums.LifeModeTeen:
		return mTeen
	}
	return 0
}

// Spec is one registry row.
type Spec struct {
	Domain Domain
	// Max is the highest level the domain can be granted (a grant above it is refused, never clamped, on write).
	Max Level
	// Modes are the owner modes the domain is visible in.
	Modes modeSet
	// OwnerData marks the owner's own health data: paused by an epoch mismatch (F-23) and `write` is a proposal (F-05).
	OwnerData bool
	// EpochBound: live only while grant.mode_epoch == owner.mode_epoch (F-23: every owner-data domain except the
	// urgent alert).
	EpochBound bool
	// Male: a male owner may grant it (F-20).
	Male bool
	// RoleMax caps each role (off = the role may not hold it); RoleDefault seeds a new link of that role.
	RoleMax     [numRoles]Level
	RoleDefault [numRoles]Level
}

// registry is indexed by Domain. The role caps follow family.md §2: the partner may hold every domain up to its max
// (default column = the seed), a carer only pregnancy and appointments at view (time-boxed, F-13), a friend only the
// cycle phase (F-01), the other parent nothing on the owner (shared children through guardianship), the teen parent
// only the teen card at view (F-17).
var registry = func() [numDomains]Spec {
	var r [numDomains]Spec
	set := func(s Spec) { r[s.Domain] = s }
	set(Spec{Domain: DomainCyclePhase, Max: LevelView, Modes: mCycle | mTTC, OwnerData: true, EpochBound: true,
		RoleMax:     roles(RolePartner, LevelView, RoleFriend, LevelView),
		RoleDefault: roles(RolePartner, LevelView, RoleFriend, LevelView)})
	set(Spec{Domain: DomainPregnancy, Max: LevelView, Modes: mPregnancy, OwnerData: true, EpochBound: true,
		RoleMax:     roles(RolePartner, LevelView, RoleCarer, LevelView),
		RoleDefault: roles(RolePartner, LevelView, RoleCarer, LevelView)})
	set(Spec{Domain: DomainAppointments, Max: LevelWrite, Modes: modeAny, OwnerData: true, EpochBound: true, Male: true,
		RoleMax:     roles(RolePartner, LevelWrite, RoleCarer, LevelView),
		RoleDefault: roles(RolePartner, LevelWrite, RoleCarer, LevelView)})
	set(Spec{Domain: DomainFetalActivity, Max: LevelView, Modes: mPregnancy, OwnerData: true, EpochBound: true,
		RoleMax:     roles(RolePartner, LevelView),
		RoleDefault: roles(RolePartner, LevelView)})
	set(Spec{Domain: DomainVitals, Max: LevelView, Modes: modeAny, OwnerData: true, EpochBound: true,
		RoleMax: roles(RolePartner, LevelView)})
	set(Spec{Domain: DomainMeds, Max: LevelWrite, Modes: modeAny, OwnerData: true, EpochBound: true, Male: true,
		RoleMax:     roles(RolePartner, LevelWrite),
		RoleDefault: roles(RolePartner, LevelNudge)})
	set(Spec{Domain: DomainUrgentAlert, Max: LevelView, Modes: modeAny, OwnerData: true,
		RoleMax:     roles(RolePartner, LevelView),
		RoleDefault: roles(RolePartner, LevelView)})
	set(Spec{Domain: DomainHousehold, Max: LevelWrite, Modes: modeAny, Male: true,
		RoleMax:     roles(RolePartner, LevelWrite),
		RoleDefault: roles(RolePartner, LevelWrite)})
	set(Spec{Domain: DomainTTCFertility, Max: LevelView, Modes: mTTC, OwnerData: true, EpochBound: true,
		RoleMax: roles(RolePartner, LevelView)}) // F-02: partner-only switch, default off
	for _, d := range []Domain{DomainTeenPeriodWeek, DomainTeenKit, DomainTeenNotes} {
		set(Spec{Domain: d, Max: LevelView, Modes: mTeen, RoleMax: roles(RoleTeenParent, LevelView)})
	}
	return r
}()

func roles(pairs ...any) [numRoles]Level {
	var out [numRoles]Level
	for i := 0; i+1 < len(pairs); i += 2 {
		out[pairs[i].(Role)] = pairs[i+1].(Level) //nolint:forcetypeassert // package-init literal pairs
	}
	return out
}

// Lookup returns the registry row of a grantable domain.
func Lookup(d Domain) (Spec, bool) {
	if d == DomainNone || d >= numDomains || d == DomainChildren {
		return Spec{}, false
	}
	return registry[d], true
}

// MaxFor is the highest level role may hold on d on an owner of the given sex (off = not allowed).
func MaxFor(d Domain, r Role, male bool) Level {
	s, ok := Lookup(d)
	if !ok || r >= numRoles {
		return LevelOff
	}
	if male && !s.Male {
		return LevelOff
	}
	return min(s.RoleMax[r], s.Max)
}

// Defaults are the seed grants of a new link of role r (custom template).
func Defaults(r Role) map[Domain]Level {
	out := map[Domain]Level{}
	if r >= numRoles {
		return out
	}
	for _, d := range GrantDomains {
		if l := registry[d].RoleDefault[r]; l > LevelOff {
			out[d] = l
		}
	}
	return out
}

// Template is a named grant preset (SPEC §1.7).
type Template string

// Templates. `legacy` marks links made before v6 (00106 backfill).
const (
	TemplateCycleBuddy       Template = "cycle_buddy"
	TemplateMedsAppointments Template = "meds_appointments"
	TemplateFullFamily       Template = "full_family"
	TemplateCustom           Template = "custom"
	TemplateLegacy           Template = "legacy"
)

// TemplateGrants are the grants a template seeds for role r, capped to what the role may hold (a row the role may
// not hold is left out). Children in full_family go through guardianship (T-M11-11), not a grant. custom and legacy
// seed nothing.
func TemplateGrants(t Template, r Role) map[Domain]Level {
	var want map[Domain]Level
	switch t {
	case TemplateCycleBuddy:
		want = map[Domain]Level{DomainCyclePhase: LevelView}
	case TemplateMedsAppointments:
		want = map[Domain]Level{DomainAppointments: LevelWrite, DomainMeds: LevelNudge}
	case TemplateFullFamily:
		want = map[Domain]Level{DomainPregnancy: LevelView, DomainAppointments: LevelWrite, DomainMeds: LevelNudge,
			DomainUrgentAlert: LevelView}
		if r == RolePartner {
			want[DomainFetalActivity] = LevelView
		}
	default:
		return map[Domain]Level{}
	}
	out := map[Domain]Level{}
	for d, l := range want {
		if c := MaxFor(d, r, false); c > LevelOff {
			out[d] = min(l, c)
		}
	}
	return out
}

// CarerTimeBoxDays is the carer default when the owner is neither pregnant nor postpartum (F-13).
const CarerTimeBoxDays = 30

// carerAfterDays is added to the EDD (pregnant) or the delivery date (postpartum) (F-13).
const carerAfterDays = 42

// CarerExpiry is a carer link's default expires_at (F-13): pregnant → EDD + 42 days, postpartum → delivery + 42 days,
// otherwise now + 30 days. A zero edd / delivery falls back to now + 30 days. The owner can extend it.
func CarerExpiry(mode string, edd, delivery, now time.Time) time.Time {
	day := 24 * time.Hour
	switch {
	case enums.LifeMode(mode) == enums.LifeModePregnancy && !edd.IsZero():
		return edd.Add(carerAfterDays * day)
	case enums.LifeMode(mode) == enums.LifeModePostpartum && !delivery.IsZero():
		return delivery.Add(carerAfterDays * day)
	}
	return now.Add(CarerTimeBoxDays * day)
}
