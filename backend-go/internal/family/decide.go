package family

import "time"

// Action is what the actor wants to do on a domain.
type Action uint8

// Actions and the level each needs: read → view, nudge → nudge, write → write.
const (
	ActionRead Action = iota + 1
	ActionNudge
	ActionWrite
)

func (a Action) needs() (Level, bool) {
	switch a {
	case ActionRead:
		return LevelView, true
	case ActionNudge:
		return LevelNudge, true
	case ActionWrite:
		return LevelWrite, true
	}
	return LevelOff, false
}

// Outcome is the answer of check().
type Outcome uint8

// Outcomes. The zero value is Deny.
const (
	Deny Outcome = iota
	Allow
	// Propose: a `write` on the owner's own record becomes a proposal she approves (F-04, F-05; HTTP 202).
	Propose
)

var outcomeCodes = [...]string{Deny: "deny", Allow: "allow", Propose: "propose"}

func (o Outcome) String() string { return outcomeCodes[o%3] }

// Reason says why Decide denied (ReasonNone on allow / propose). Members never see it (F-21: only 404 vs 403).
type Reason uint8

// Reasons (family.md §5).
const (
	ReasonNone Reason = iota
	ReasonNoLink
	ReasonExpired
	ReasonBlocked
	ReasonNotAllowed
	ReasonModeInactive
	ReasonLevel
	ReasonNotGuardian
)

var reasonCodes = [...]string{ReasonNone: "", ReasonNoLink: "no_link", ReasonExpired: "expired",
	ReasonBlocked: "blocked", ReasonNotAllowed: "not_allowed", ReasonModeInactive: "mode_inactive", ReasonLevel: "level",
	ReasonNotGuardian: "not_guardian"}

func (r Reason) String() string {
	if int(r) < len(reasonCodes) {
		return reasonCodes[r]
	}
	return "unknown"
}

// NotFound reports whether the denial is answered 404 (no existence oracle, F-21): no link, expired, blocked owner,
// not a guardian. Every other denial is a 403 on an active link.
func (r Reason) NotFound() bool {
	return r == ReasonNoLink || r == ReasonExpired || r == ReasonBlocked || r == ReasonNotGuardian
}

// Decision is Decide's answer.
type Decision struct {
	Outcome Outcome
	Reason  Reason
}

// Allowed reports an Allow.
func (d Decision) Allowed() bool { return d.Outcome == Allow }

// SubjectKind is whose data the action touches.
type SubjectKind uint8

// Subject kinds.
const (
	SubjectOwner SubjectKind = iota // the owner's own records (the snapshot's OwnerID)
	SubjectHousehold
	SubjectChild
)

// SubScope is a child sub-scope; a limited (carer) guardian may act only inside the scopes of their row.
type SubScope uint8

// Child sub-scopes (family.md §2: carer feeding / diaper / sleep only).
const (
	ScopeOther SubScope = 1 << iota
	ScopeFeeding
	ScopeDiaper
	ScopeSleep
)

// ParseSubScope maps a sub-scope code ("" / unknown → ScopeOther, which no limited guardian holds).
func ParseSubScope(code string) SubScope {
	switch code {
	case "feeding":
		return ScopeFeeding
	case "diaper":
		return ScopeDiaper
	case "sleep":
		return ScopeSleep
	}
	return ScopeOther
}

// Subject is the target of the action.
type Subject struct {
	Kind    SubjectKind
	ChildID uint64   // SubjectChild
	Scope   SubScope // SubjectChild: the sub-scope the action touches
}

// Grant is one stored grant row as Decide reads it.
type Grant struct {
	Level   Level
	Epoch   uint32
	Stamped bool // mode_epoch IS NOT NULL
}

// Guardianship is one child_guardians row of the actor.
type Guardianship struct {
	ChildID   uint64
	Active    bool
	Limited   bool     // level = limited (carer): only SubScopes
	SubScopes SubScope // bit set
}

// Snapshot is everything Decide needs about one (owner, actor) pair, loaded in one statement (Loader).
type Snapshot struct {
	OwnerID uint64
	ActorID uint64
	// Owner state: effective mode, mode epoch (0 = unknown → every epoch-bound domain denies), blocked / male.
	OwnerMode    Mode
	OwnerEpoch   uint32
	OwnerBlocked bool
	OwnerMale    bool
	// Link: the active link owner → actor (HasLink false = none).
	HasLink   bool
	LinkID    uint64
	Role      Role
	ExpiresAt time.Time // zero = no expiry
	Grants    [numDomains]Grant
	// Guardians: the actor's guardianships of the children asked about (SubjectChild).
	Guardians []Guardianship
}

// Decide is check(actor, subject, domain, action) (SPEC §1.8, family.md §5, F-23). Pure, allocation-free; order:
//
//  1. actor is the owner (owner / household subject) → allow;
//  2. child subject → an active guardian allow (limited: listed sub-scopes only), else deny not_guardian;
//  3. no active link → no_link; owner blocked → blocked;
//  4. now ≥ expires_at → expired;
//  5. domain not grantable for the role or for a male owner (or never-shared) → not_allowed;
//  6. owner mode out of the domain's scope, or an epoch-bound grant not stamped with the owner's epoch → mode_inactive;
//  7. level below the action → level;
//  8. nudge → allow (the caller checks the quota);
//  9. write on an owner record → propose; write on household → allow; read → allow.
func Decide(s Snapshot, subj Subject, d Domain, a Action, now time.Time) Decision {
	need, ok := a.needs()
	if !ok {
		return Decision{Deny, ReasonLevel}
	}
	switch subj.Kind {
	case SubjectChild:
		return decideChild(s, subj)
	case SubjectOwner, SubjectHousehold:
		if s.ActorID != 0 && s.ActorID == s.OwnerID {
			return Decision{Allow, ReasonNone}
		}
	default:
		return Decision{Deny, ReasonNotAllowed}
	}
	if !s.HasLink || s.OwnerID == 0 || s.ActorID == 0 {
		return Decision{Deny, ReasonNoLink}
	}
	if s.OwnerBlocked {
		return Decision{Deny, ReasonBlocked}
	}
	if !s.ExpiresAt.IsZero() && !now.Before(s.ExpiresAt) {
		return Decision{Deny, ReasonExpired}
	}
	if (subj.Kind == SubjectHousehold) != (d == DomainHousehold) {
		return Decision{Deny, ReasonNotAllowed}
	}
	spec, ok := Lookup(d)
	if !ok {
		return Decision{Deny, ReasonNotAllowed}
	}
	maxLevel := MaxFor(d, s.Role, s.OwnerMale)
	if maxLevel == LevelOff {
		return Decision{Deny, ReasonNotAllowed}
	}
	g := s.Grants[d]
	if spec.Modes&s.OwnerMode == 0 {
		return Decision{Deny, ReasonModeInactive}
	}
	if spec.EpochBound && (!g.Stamped || s.OwnerEpoch == 0 || g.Epoch != s.OwnerEpoch) {
		return Decision{Deny, ReasonModeInactive}
	}
	if min(g.Level, maxLevel) < need {
		return Decision{Deny, ReasonLevel}
	}
	if a == ActionWrite && spec.OwnerData {
		return Decision{Propose, ReasonNone}
	}
	return Decision{Allow, ReasonNone}
}

func decideChild(s Snapshot, subj Subject) Decision {
	if subj.ChildID == 0 || s.ActorID == 0 {
		return Decision{Deny, ReasonNotGuardian}
	}
	for i := range s.Guardians {
		g := &s.Guardians[i]
		if g.ChildID != subj.ChildID || !g.Active {
			continue
		}
		if g.Limited && (subj.Scope == 0 || subj.Scope == ScopeOther || g.SubScopes&subj.Scope != subj.Scope) {
			return Decision{Deny, ReasonNotGuardian}
		}
		return Decision{Allow, ReasonNone}
	}
	return Decision{Deny, ReasonNotGuardian}
}
