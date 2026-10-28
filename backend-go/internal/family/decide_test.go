package family

import (
	"fmt"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var t0 = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

// allowRule is one row of the allow table: role may reach domain up to max, in the listed owner modes. Written by hand
// from docs/engine-v6/family.md §2 (independent of registry.go): if registry.go grants anything not listed here,
// TestDecide_Matrix fails. Everything absent from this table must deny.
type allowRule struct {
	role  Role
	dom   Domain
	max   Level
	modes []string
	epoch bool // epoch-bound (F-23)
	owner bool // owner data: write → propose
	male  bool // a male owner may grant it (F-20)
}

var adult = []string{"cycle", "ttc", "pregnancy", "postpartum", "menopause"}

var allowTable = []allowRule{
	{RolePartner, DomainCyclePhase, LevelView, []string{"cycle", "ttc"}, true, true, false},
	{RolePartner, DomainPregnancy, LevelView, []string{"pregnancy"}, true, true, false},
	{RolePartner, DomainAppointments, LevelWrite, adult, true, true, true},
	{RolePartner, DomainFetalActivity, LevelView, []string{"pregnancy"}, true, true, false},
	{RolePartner, DomainVitals, LevelView, adult, true, true, false},
	{RolePartner, DomainMeds, LevelWrite, adult, true, true, true},
	{RolePartner, DomainUrgentAlert, LevelView, adult, false, true, false},
	{RolePartner, DomainHousehold, LevelWrite, adult, false, false, true},
	{RolePartner, DomainTTCFertility, LevelView, []string{"ttc"}, true, true, false},
	{RoleCarer, DomainPregnancy, LevelView, []string{"pregnancy"}, true, true, false},
	{RoleCarer, DomainAppointments, LevelView, adult, true, true, true},
	{RoleFriend, DomainCyclePhase, LevelView, []string{"cycle", "ttc"}, true, true, false},
	{RoleTeenParent, DomainTeenPeriodWeek, LevelView, []string{"teen"}, false, false, false},
	{RoleTeenParent, DomainTeenKit, LevelView, []string{"teen"}, false, false, false},
	{RoleTeenParent, DomainTeenNotes, LevelView, []string{"teen"}, false, false, false},
}

func ruleFor(r Role, d Domain) (allowRule, bool) {
	for _, a := range allowTable {
		if a.role == r && a.dom == d {
			return a, true
		}
	}
	return allowRule{}, false
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func memberSnap(r Role, mode string, epoch uint32, d Domain, l Level, grantEpoch uint32, stamped bool) Snapshot {
	s := Snapshot{OwnerID: 1, ActorID: 2, OwnerMode: ModeOf(mode), OwnerEpoch: epoch, HasLink: true, LinkID: 9, Role: r}
	if d < numDomains {
		s.Grants[d] = Grant{Level: l, Epoch: grantEpoch, Stamped: stamped}
	}
	return s
}

// expected is the oracle from the allow table.
func expected(r Role, d Domain, l Level, a Action, mode string, epochMatch, expired, blocked, male bool) Decision {
	if blocked {
		return Decision{Deny, ReasonBlocked}
	}
	if expired {
		return Decision{Deny, ReasonExpired}
	}
	if d == DomainHousehold {
		// asked on an owner subject in the matrix: household needs the household subject
		return Decision{Deny, ReasonNotAllowed}
	}
	rule, ok := ruleFor(r, d)
	if !ok || (male && !rule.male) {
		return Decision{Deny, ReasonNotAllowed}
	}
	if !contains(rule.modes, mode) || (rule.epoch && !epochMatch) {
		return Decision{Deny, ReasonModeInactive}
	}
	need := map[Action]Level{ActionRead: LevelView, ActionNudge: LevelNudge, ActionWrite: LevelWrite}[a]
	if min(l, rule.max) < need {
		return Decision{Deny, ReasonLevel}
	}
	if a == ActionWrite && rule.owner {
		return Decision{Propose, ReasonNone}
	}
	return Decision{Allow, ReasonNone}
}

// TestDecide_Matrix: domain × level × action × role × mode × epoch × expiry × blocked × sex, every row checked against
// the hand-written allow table, with its reason.
func TestDecide_Matrix(t *testing.T) {
	modes := []string{"cycle", "ttc", "pregnancy", "postpartum", "menopause", "teen", ""}
	domains := append(append([]Domain{}, GrantDomains...), NeverShared...)
	domains = append(domains, DomainChildren, DomainNone)
	rows, allows := 0, 0
	for _, r := range []Role{RoleNone, RolePartner, RoleOtherParent, RoleCarer, RoleFriend, RoleTeenParent} {
		for _, d := range domains {
			for _, l := range []Level{LevelOff, LevelView, LevelNudge, LevelWrite} {
				for _, a := range []Action{ActionRead, ActionNudge, ActionWrite} {
					for _, mode := range modes {
						for _, epochMatch := range []bool{true, false} {
							for _, expired := range []bool{false, true} {
								for _, blocked := range []bool{false, true} {
									for _, male := range []bool{false, true} {
										ge := uint32(3)
										if !epochMatch {
											ge = 2
										}
										s := memberSnap(r, mode, 3, d, l, ge, true)
										s.OwnerBlocked, s.OwnerMale = blocked, male
										if expired {
											s.ExpiresAt = t0
										} else {
											s.ExpiresAt = t0.Add(time.Second)
										}
										got := Decide(s, Subject{}, d, a, t0)
										want := expected(r, d, l, a, mode, epochMatch, expired, blocked, male)
										rows++
										if got.Outcome != Deny {
											allows++
										}
										if got != want {
											t.Fatalf("role=%s dom=%d(%s) level=%s action=%d mode=%q epoch=%v expired=%v blocked=%v male=%v: got %s/%s want %s/%s",
												r, d, d, l, a, mode, epochMatch, expired, blocked, male, got.Outcome, got.Reason, want.Outcome, want.Reason)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	assert.Positive(t, allows)
	t.Logf("%d rows, %d allow/propose", rows, allows)
}

// TestDecide_Rows is the readable part of the matrix: one named row per rule and reason.
func TestDecide_Rows(t *testing.T) {
	live := func(r Role, mode string, d Domain, l Level) Snapshot { return memberSnap(r, mode, 4, d, l, 4, true) }
	cases := []struct {
		name string
		s    Snapshot
		subj Subject
		d    Domain
		a    Action
		want Decision
	}{
		{"owner reads her own data", Snapshot{OwnerID: 1, ActorID: 1}, Subject{}, DomainVitals, ActionRead, Decision{Allow, ReasonNone}},
		{"owner writes her own data", Snapshot{OwnerID: 1, ActorID: 1}, Subject{}, DomainMeds, ActionWrite, Decision{Allow, ReasonNone}},
		{"no link", Snapshot{OwnerID: 1, ActorID: 2}, Subject{}, DomainCyclePhase, ActionRead, Decision{Deny, ReasonNoLink}},
		{"blocked owner", func() Snapshot {
			s := live(RolePartner, "cycle", DomainCyclePhase, LevelView)
			s.OwnerBlocked = true
			return s
		}(), Subject{}, DomainCyclePhase, ActionRead, Decision{Deny, ReasonBlocked}},
		{"expired carer", func() Snapshot {
			s := live(RoleCarer, "pregnancy", DomainPregnancy, LevelView)
			s.ExpiresAt = t0.Add(-time.Minute)
			return s
		}(), Subject{}, DomainPregnancy, ActionRead, Decision{Deny, ReasonExpired}},
		{"carer before expiry", func() Snapshot {
			s := live(RoleCarer, "pregnancy", DomainPregnancy, LevelView)
			s.ExpiresAt = t0.Add(time.Minute)
			return s
		}(), Subject{}, DomainPregnancy, ActionRead, Decision{Allow, ReasonNone}},
		{"partner reads cycle phase", live(RolePartner, "cycle", DomainCyclePhase, LevelView), Subject{}, DomainCyclePhase, ActionRead, Decision{Allow, ReasonNone}},
		{"cycle phase paused in pregnancy", live(RolePartner, "pregnancy", DomainCyclePhase, LevelView), Subject{}, DomainCyclePhase, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"stale epoch pauses meds (F-23)", memberSnap(RolePartner, "cycle", 5, DomainMeds, LevelWrite, 4, true), Subject{}, DomainMeds, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"unstamped grant denies", memberSnap(RolePartner, "pregnancy", 1, DomainPregnancy, LevelView, 0, false), Subject{}, DomainPregnancy, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"unknown owner epoch denies", memberSnap(RolePartner, "cycle", 0, DomainMeds, LevelView, 0, true), Subject{}, DomainMeds, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"urgent alert survives a stale epoch", memberSnap(RolePartner, "cycle", 5, DomainUrgentAlert, LevelView, 4, true), Subject{}, DomainUrgentAlert, ActionRead, Decision{Allow, ReasonNone}},
		{"teen owner pauses partner meds", live(RolePartner, "teen", DomainMeds, LevelWrite), Subject{}, DomainMeds, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"teen parent reads the teen card", live(RoleTeenParent, "teen", DomainTeenKit, LevelView), Subject{}, DomainTeenKit, ActionRead, Decision{Allow, ReasonNone}},
		{"teen parent never writes", live(RoleTeenParent, "teen", DomainTeenKit, LevelWrite), Subject{}, DomainTeenKit, ActionWrite, Decision{Deny, ReasonLevel}},
		{"partner on teen card", live(RolePartner, "teen", DomainTeenKit, LevelView), Subject{}, DomainTeenKit, ActionRead, Decision{Deny, ReasonNotAllowed}},
		// H1: a teen owner with a pregnancy flag still resolves to teen (resolveMode) — adult sharing stays suspended.
		{"teen+pregnant owner pauses partner pregnancy", live(RolePartner, resolveMode("teen", true, ""), DomainPregnancy, LevelView), Subject{}, DomainPregnancy, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"teen+pregnant owner pauses partner meds", live(RolePartner, resolveMode("teen", true, ""), DomainMeds, LevelWrite), Subject{}, DomainMeds, ActionWrite, Decision{Deny, ReasonModeInactive}},
		{"teen+pregnant owner pauses partner appointments", live(RolePartner, resolveMode("teen", true, ""), DomainAppointments, LevelWrite), Subject{}, DomainAppointments, ActionRead, Decision{Deny, ReasonModeInactive}},
		// L1: teen domains are live only while the owner is a teen.
		{"teen card off once the owner is an adult", live(RoleTeenParent, "cycle", DomainTeenKit, LevelView), Subject{}, DomainTeenKit, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"teen week off in pregnancy", live(RoleTeenParent, "pregnancy", DomainTeenPeriodWeek, LevelView), Subject{}, DomainTeenPeriodWeek, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"friend on meds", live(RoleFriend, "cycle", DomainMeds, LevelWrite), Subject{}, DomainMeds, ActionRead, Decision{Deny, ReasonNotAllowed}},
		{"male owner cycle phase (F-20)", func() Snapshot {
			s := live(RolePartner, "cycle", DomainCyclePhase, LevelView)
			s.OwnerMale = true
			return s
		}(), Subject{}, DomainCyclePhase, ActionRead, Decision{Deny, ReasonNotAllowed}},
		{"male owner meds (F-20)", func() Snapshot { s := live(RolePartner, "cycle", DomainMeds, LevelView); s.OwnerMale = true; return s }(), Subject{}, DomainMeds, ActionRead, Decision{Allow, ReasonNone}},
		{"view cannot nudge", live(RolePartner, "cycle", DomainMeds, LevelView), Subject{}, DomainMeds, ActionNudge, Decision{Deny, ReasonLevel}},
		{"nudge level nudges", live(RolePartner, "cycle", DomainMeds, LevelNudge), Subject{}, DomainMeds, ActionNudge, Decision{Allow, ReasonNone}},
		{"nudge cannot write", live(RolePartner, "cycle", DomainMeds, LevelNudge), Subject{}, DomainMeds, ActionWrite, Decision{Deny, ReasonLevel}},
		{"write on owner meds proposes", live(RolePartner, "cycle", DomainMeds, LevelWrite), Subject{}, DomainMeds, ActionWrite, Decision{Propose, ReasonNone}},
		{"write on appointments proposes", live(RolePartner, "postpartum", DomainAppointments, LevelWrite), Subject{}, DomainAppointments, ActionWrite, Decision{Propose, ReasonNone}},
		{"write on household allows", live(RolePartner, "pregnancy", DomainHousehold, LevelWrite), Subject{Kind: SubjectHousehold}, DomainHousehold, ActionWrite, Decision{Allow, ReasonNone}},
		{"household on owner subject", live(RolePartner, "pregnancy", DomainHousehold, LevelWrite), Subject{}, DomainHousehold, ActionWrite, Decision{Deny, ReasonNotAllowed}},
		{"stored level above max is capped", live(RolePartner, "cycle", DomainCyclePhase, LevelWrite), Subject{}, DomainCyclePhase, ActionWrite, Decision{Deny, ReasonLevel}},
		{"ttc switch in ttc", live(RolePartner, "ttc", DomainTTCFertility, LevelView), Subject{}, DomainTTCFertility, ActionRead, Decision{Allow, ReasonNone}},
		{"ttc switch out of ttc", live(RolePartner, "cycle", DomainTTCFertility, LevelView), Subject{}, DomainTTCFertility, ActionRead, Decision{Deny, ReasonModeInactive}},
		{"never-shared symptoms", live(RolePartner, "cycle", DomainCyclePhase, LevelWrite), Subject{}, NeverSymptoms, ActionRead, Decision{Deny, ReasonNotAllowed}},
		{"unknown action", live(RolePartner, "cycle", DomainCyclePhase, LevelView), Subject{}, DomainCyclePhase, 0, Decision{Deny, ReasonLevel}},
		{"child: active write guardian", Snapshot{OwnerID: 1, ActorID: 2, Guardians: []Guardianship{{ChildID: 7, Active: true}}}, Subject{Kind: SubjectChild, ChildID: 7}, DomainChildren, ActionWrite, Decision{Allow, ReasonNone}},
		{"child: not a guardian of this child", Snapshot{OwnerID: 1, ActorID: 2, Guardians: []Guardianship{{ChildID: 8, Active: true}}}, Subject{Kind: SubjectChild, ChildID: 7}, DomainChildren, ActionRead, Decision{Deny, ReasonNotGuardian}},
		{"child: guardian left", Snapshot{OwnerID: 1, ActorID: 2, Guardians: []Guardianship{{ChildID: 7}}}, Subject{Kind: SubjectChild, ChildID: 7}, DomainChildren, ActionRead, Decision{Deny, ReasonNotGuardian}},
		{"child: carer in scope", Snapshot{ActorID: 2, Guardians: []Guardianship{{ChildID: 7, Active: true, Limited: true, SubScopes: ScopeFeeding | ScopeSleep}}}, Subject{Kind: SubjectChild, ChildID: 7, Scope: ScopeFeeding}, DomainChildren, ActionWrite, Decision{Allow, ReasonNone}},
		{"child: carer out of scope", Snapshot{ActorID: 2, Guardians: []Guardianship{{ChildID: 7, Active: true, Limited: true, SubScopes: ScopeFeeding}}}, Subject{Kind: SubjectChild, ChildID: 7, Scope: ScopeDiaper}, DomainChildren, ActionWrite, Decision{Deny, ReasonNotGuardian}},
		{"child: carer on vaccines", Snapshot{ActorID: 2, Guardians: []Guardianship{{ChildID: 7, Active: true, Limited: true, SubScopes: ScopeFeeding | ScopeDiaper | ScopeSleep}}}, Subject{Kind: SubjectChild, ChildID: 7, Scope: ScopeOther}, DomainChildren, ActionRead, Decision{Deny, ReasonNotGuardian}},
		{"child: an owner link is not guardianship", live(RolePartner, "cycle", DomainMeds, LevelWrite), Subject{Kind: SubjectChild, ChildID: 7}, DomainChildren, ActionRead, Decision{Deny, ReasonNotGuardian}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, Decide(c.s, c.subj, c.d, c.a, t0))
		})
	}
}

// TestDecide_Properties: random snapshots never reach a never-shared category, never `allow` a write on an owner
// record, and always deny without an active, unexpired link of an unblocked owner.
func TestDecide_Properties(t *testing.T) {
	gen := func(role, mode, epoch, dom, act, lvl, ge uint8, stamped, blocked, hasLink, expired, male bool) bool {
		s := Snapshot{OwnerID: 1, ActorID: 2, OwnerMode: Mode(1) << (mode % 7), OwnerEpoch: uint32(epoch % 3),
			OwnerBlocked: blocked, OwnerMale: male, HasLink: hasLink, Role: Role(role % uint8(numRoles+1))}
		if expired {
			s.ExpiresAt = t0
		}
		for _, d := range GrantDomains {
			s.Grants[d] = Grant{Level: Level(lvl % 5), Epoch: uint32(ge % 3), Stamped: stamped}
		}
		a := Action(act%3 + 1)
		// never-shared
		for _, d := range NeverShared {
			if Decide(s, Subject{}, d, a, t0).Outcome != Deny {
				return false
			}
		}
		d := GrantDomains[int(dom)%len(GrantDomains)]
		got := Decide(s, Subject{}, d, a, t0)
		spec, _ := Lookup(d)
		if a == ActionWrite && spec.OwnerData && got.Outcome == Allow {
			return false
		}
		if (!hasLink || blocked || expired) && got.Outcome != Deny {
			return false
		}
		return true
	}
	require.NoError(t, quick.Check(gen, &quick.Config{MaxCount: 20000}))
}

func TestResolveMode_TeenBeatsPregnancy(t *testing.T) {
	assert.Equal(t, "teen", resolveMode("teen", true, ""), "FAM-H1: a pregnancy flag never lifts teen suspension")
	assert.Equal(t, "teen", resolveMode("teen", false, "pregnancy"))
	assert.Equal(t, "pregnancy", resolveMode("cycle", true, ""))
}

func TestRegistry_TeenDomainsTeenOnly(t *testing.T) {
	for _, d := range []Domain{DomainTeenPeriodWeek, DomainTeenKit, DomainTeenNotes} {
		spec, ok := Lookup(d)
		require.True(t, ok, d.String())
		assert.Equal(t, mTeen, spec.Modes, "%s is live in teen mode only (L1)", d)
	}
}

func TestRegistry(t *testing.T) {
	for _, d := range GrantDomains {
		got, ok := ParseDomain(d.String())
		require.True(t, ok, d.String())
		assert.Equal(t, d, got)
	}
	_, ok := ParseDomain("symptoms")
	assert.False(t, ok, "symptoms is gone (F-03)")
	_, ok = ParseDomain("children")
	assert.False(t, ok, "children go through guardianship")
	assert.Equal(t, RolePartner, ParseRole("", "spouse"))
	assert.Equal(t, RoleTeenParent, ParseRole("", "parent"))
	assert.Equal(t, RoleNone, ParseRole("admin", "partner"))
	assert.Equal(t, map[Domain]Level{DomainCyclePhase: LevelView}, TemplateGrants(TemplateCycleBuddy, RoleFriend))
	assert.Equal(t, map[Domain]Level{DomainAppointments: LevelWrite, DomainMeds: LevelNudge}, TemplateGrants(TemplateMedsAppointments, RolePartner))
	assert.Equal(t, map[Domain]Level{DomainPregnancy: LevelView, DomainAppointments: LevelView}, TemplateGrants(TemplateFullFamily, RoleCarer))
	assert.Equal(t, LevelNudge, Defaults(RolePartner)[DomainMeds], "spouse default = nudge (F-04)")
	assert.NotContains(t, Defaults(RolePartner), DomainTTCFertility, "TTC switch default off (F-02)")
	edd := t0.AddDate(0, 2, 0)
	assert.Equal(t, edd.AddDate(0, 0, 42), CarerExpiry("pregnancy", edd, time.Time{}, t0))
	assert.Equal(t, t0.AddDate(0, 0, 30), CarerExpiry("cycle", edd, time.Time{}, t0))
	for _, r := range []Reason{ReasonNoLink, ReasonExpired, ReasonBlocked, ReasonNotGuardian} {
		assert.True(t, r.NotFound(), fmt.Sprint(r))
	}
	for _, r := range []Reason{ReasonNotAllowed, ReasonModeInactive, ReasonLevel} {
		assert.False(t, r.NotFound(), fmt.Sprint(r))
	}
}

func BenchmarkDecide(b *testing.B) {
	s := memberSnap(RolePartner, "cycle", 3, DomainMeds, LevelWrite, 3, true)
	s.ExpiresAt = t0.Add(time.Hour)
	b.ReportAllocs()
	var sink Decision
	for b.Loop() {
		sink = Decide(s, Subject{}, DomainMeds, ActionWrite, t0)
	}
	if sink.Outcome != Propose {
		b.Fatal(sink)
	}
}

func TestDecide_ZeroAllocs(t *testing.T) {
	s := memberSnap(RolePartner, "cycle", 3, DomainMeds, LevelWrite, 3, true)
	allocs := testing.AllocsPerRun(1000, func() { _ = Decide(s, Subject{}, DomainMeds, ActionWrite, t0) })
	assert.Zero(t, allocs)
}
