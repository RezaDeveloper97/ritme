// Package companion is «همدم» & family (bloom B-N4-01, canvas boards nbl_Hamdam_*): a user (the owner) invites a
// partner or spouse account with a one-time 6-character code (optionally bound to a mobile number), decides per
// section what that account may see or edit, and with a spouse forms a family whose children are shared.
//
// This package owns the schema (goose 00026_companions.sql), the sqlc store and the domain service. HTTP routes,
// the SMS adapter, owner notifications and the access-filtered reads/writes of other domains are wired by B-N4-02
// through Service: CreateInvite / RenewInvite / Accept / Revoke / SetGrants / SetSharedChildren / ListForOwner /
// ListForCompanion, the access checks CanRead / CanWrite / Level, and Audit for the read/write trail.
//
// Privacy rules:
//   - Grants are explicit. A section without a grant is none; a new link starts with exactly the grants the owner
//     picked (none by default). Grants only take effect while the link is active; revoking deletes them.
//   - The plain invite code exists only in the CreateInvite / RenewInvite result; the database keeps its HMAC.
//   - The audit trail records who touched which section and when, never a health payload.
//
// Codes are strings validated here (not DB ENUMs) so the canvas queue can add companion type `parent` and teen-only
// sections (CB-TEEN-01) and IVF / loss / record-sharing scopes (CB-IVF-01, CB-LOSS-01, CB-REC-03) by extending
// Types / Sections.
package companion

import (
	"errors"
	"slices"
)

// Type is the relation of the companion to the owner.
type Type string

// Companion types (Hamdam_Type). A spouse also forms a family with shared children. A parent (CB-TEEN-01,
// nbl_Teen_Parent) is the mother / parent a teen shares a read-only card with: only a teen-mode owner may invite one,
// it carries only TeenSections at level view, and it never writes or reads the regular sections.
const (
	TypePartner Type = "partner"
	TypeSpouse  Type = "spouse"
	TypeParent  Type = "parent"
)

// Types are the accepted companion types, in picker order.
var Types = []Type{TypePartner, TypeSpouse, TypeParent}

// Valid reports whether t is a known type.
func (t Type) Valid() bool { return slices.Contains(Types, t) }

// Status is the lifecycle state of a link.
type Status string

// Link statuses.
const (
	StatusInvited Status = "invited"
	StatusActive  Status = "active"
	StatusRevoked Status = "revoked"
)

// Section is a part of the owner's data a grant covers.
type Section string

// Sections (Hamdam_Access rows).
const (
	SectionCycle        Section = "cycle"
	SectionSymptoms     Section = "symptoms"
	SectionMeds         Section = "meds"
	SectionAppointments Section = "appointments"
	SectionPregnancy    Section = "pregnancy"
)

// Sections are the grantable sections of partner and spouse links, in Hamdam_Access order.
var Sections = []Section{SectionCycle, SectionSymptoms, SectionMeds, SectionAppointments, SectionPregnancy}

// Teen-only sections (CB-TEEN-01, nbl_Teen_Parent «چه چیزی را با مادرت در میان بگذاری؟»): what a teen may share with a
// parent link, and nothing else. Never a log, a date, a symptom or fertility data:
//   - teen_period_week: the next period as a week bucket only («این هفته» / «هفته بعد»);
//   - teen_kit: whether her school emergency kit is ready;
//   - teen_notes: the note she wrote for her parent (teen_profiles.parent_note), nothing she logged.
const (
	SectionTeenPeriodWeek Section = "teen_period_week"
	SectionTeenKit        Section = "teen_kit"
	SectionTeenNotes      Section = "teen_notes"
)

// TeenSections are the grantable sections of a parent link, in board order. View only.
var TeenSections = []Section{SectionTeenPeriodWeek, SectionTeenKit, SectionTeenNotes}

// Valid reports whether s is a known section (of any link type).
func (s Section) Valid() bool {
	return slices.Contains(Sections, s) || slices.Contains(TeenSections, s)
}

// IsTeen reports whether s is a teen-only section.
func (s Section) IsTeen() bool { return slices.Contains(TeenSections, s) }

// SectionsFor are the sections a link of type t may be granted, in display order.
func SectionsFor(t Type) []Section {
	if t == TypeParent {
		return TeenSections
	}
	return Sections
}

// AllowedFor reports whether a link of type t may hold a grant on s: parent links only the teen sections, every
// other type only the regular ones.
func (s Section) AllowedFor(t Type) bool { return slices.Contains(SectionsFor(t), s) }

// MaxLevel is the highest level a section can be granted: teen sections are read-only (a parent never writes).
func (s Section) MaxLevel() Level {
	if s.IsTeen() {
		return LevelView
	}
	return LevelEdit
}

// Permits reports whether level l may be granted on s.
func (s Section) Permits(l Level) bool { return l != LevelEdit || s.MaxLevel() == LevelEdit }

// Level is what a grant allows on a section.
type Level string

// Access levels (نبیند / فقط دیدن / دیدن و ویرایش). None is never stored: no grant row means none.
const (
	LevelNone Level = "none"
	LevelView Level = "view"
	LevelEdit Level = "edit"
)

// Valid reports whether l is a known level.
func (l Level) Valid() bool { return l == LevelNone || l == LevelView || l == LevelEdit }

// CanRead reports whether the level allows reading.
func (l Level) CanRead() bool { return l == LevelView || l == LevelEdit }

// CanWrite reports whether the level allows writing.
func (l Level) CanWrite() bool { return l == LevelEdit }

// Grants maps a section to its level. Missing sections are none.
type Grants map[Section]Level

// Of returns the level of a section (none when absent).
func (g Grants) Of(s Section) Level {
	if l, ok := g[s]; ok && l.Valid() {
		return l
	}
	return LevelNone
}

// Validate checks every key and value (any link type).
func (g Grants) Validate() error {
	for s, l := range g {
		if !s.Valid() {
			return ErrInvalidSection
		}
		if !l.Valid() {
			return ErrInvalidLevel
		}
	}
	return nil
}

// ValidateFor checks the grants of a link of type t: every section must be allowed for the type and every level
// permitted on its section (teen sections: view only).
func (g Grants) ValidateFor(t Type) error {
	if err := g.Validate(); err != nil {
		return err
	}
	for s, l := range g {
		if !s.AllowedFor(t) {
			return ErrInvalidSection
		}
		if !s.Permits(l) {
			return ErrInvalidLevel
		}
	}
	return nil
}

// Action is an audit-trail event.
type Action string

// Audit actions. Read / Write carry a section; the lifecycle ones do not.
const (
	ActionRead          Action = "read"
	ActionWrite         Action = "write"
	ActionInvited       Action = "invited"
	ActionAccepted      Action = "accepted"
	ActionRevoked       Action = "revoked"
	ActionGrantsChanged Action = "grants_changed"
)

// Who ended a link (companions.revoked_by).
const (
	RevokedByOwner     = "owner"
	RevokedByCompanion = "companion"
)

// Validation errors (B-N4-02 maps them to 422).
var (
	ErrInvalidType      = errors.New("companion: invalid type")
	ErrInvalidSection   = errors.New("companion: invalid section")
	ErrInvalidLevel     = errors.New("companion: invalid level")
	ErrInvalidPhone     = errors.New("companion: invalid phone")
	ErrInvalidName      = errors.New("companion: invalid display name")
	ErrChildrenNoSpouse = errors.New("companion: shared children need a spouse link")
	// ErrParentNeedsTeen: only an owner in teen mode may invite a parent (CB-TEEN-01).
	ErrParentNeedsTeen = errors.New("companion: a parent link needs a teen-mode owner")
	// ErrTeenParentOnly: a teen-mode owner (a minor) may only invite a parent — no partner or spouse link.
	ErrTeenParentOnly = errors.New("companion: a teen may only invite a parent")
	// ErrPhoneRequired: a parent invite must be bound to the parent's mobile number (CB-TEEN-01).
	ErrPhoneRequired = errors.New("companion: a parent invite needs a phone")
)

// Lifecycle errors.
var (
	// ErrNotFound: no such link, or the caller is not a party to it (existence is not revealed).
	ErrNotFound = errors.New("companion: not found")
	// ErrInviteInvalid: malformed or unknown code.
	ErrInviteInvalid = errors.New("companion: invalid invite code")
	// ErrInviteExpired: the 24 h window has passed.
	ErrInviteExpired = errors.New("companion: invite expired")
	// ErrInviteUsed: already accepted, renewed or revoked (one-time).
	ErrInviteUsed = errors.New("companion: invite already used")
	// ErrInviteLocked: too many failed attempts on this invite.
	ErrInviteLocked = errors.New("companion: invite locked")
	// ErrInvitePhoneMismatch: the invite is bound to another mobile number (counts as a failed attempt).
	ErrInvitePhoneMismatch = errors.New("companion: invite is for another number")
	// ErrSelfInvite: the owner cannot be her own companion.
	ErrSelfInvite = errors.New("companion: cannot invite yourself")
	// ErrAlreadyLinked: the account already has a link with this owner.
	ErrAlreadyLinked = errors.New("companion: already linked")
	// ErrSpouseExists: the owner already has an invited or active spouse.
	ErrSpouseExists = errors.New("companion: spouse already linked")
	// ErrTooManyCompanions: the owner reached MaxOpenCompanions.
	ErrTooManyCompanions = errors.New("companion: too many companions")
	// ErrInviteNotAllowed: the owner's life mode no longer allows this link type (CB-TEEN-01: a partner / spouse
	// invite of an owner now in teen mode, or a parent invite of an owner who left teen mode).
	ErrInviteNotAllowed = errors.New("companion: invite not allowed for the owner's mode")
	// ErrNotPending: the operation needs an invited (not yet accepted) link.
	ErrNotPending = errors.New("companion: link is not pending")
)
