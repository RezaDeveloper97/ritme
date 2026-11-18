package companion

import (
	"time"

	"github.com/ritme/backend-go/internal/family"
)

// Legacy ↔ v6 mapping (T-M11-05, D-204). Grants are stored with the v6 codes (family.md §2, goose 00110) and
// decided by family.Decide; /companions/* keeps its request and response shape (sections cycle | symptoms | meds |
// appointments | pregnancy and the teen card, levels none | view | edit) through this mapping:
//
//	legacy in  → v6 stored          v6 stored → legacy out
//	cycle *    → cycle_phase view   cycle_phase view → cycle view
//	pregnancy *→ pregnancy view     pregnancy view   → pregnancy view
//	meds/appointments view → view   view | nudge     → view
//	meds/appointments edit → write  write            → edit
//	symptoms * → (dropped, F-03)    —                → symptoms none
//	teen_* view → teen_* view       teen_* view      → teen_* view
//
// What a member effectively has (Level, ListForCompanion) is never the stored row: it is Decide over the loaded
// snapshot (expiry, blocked owner, role, male owner, mode scope and epoch, teen-mode owner).

// sectionDomain is the v6 domain of a legacy section (false for symptoms, which no longer exists).
func sectionDomain(s Section) (family.Domain, bool) {
	switch s {
	case SectionCycle:
		return family.DomainCyclePhase, true
	case SectionPregnancy:
		return family.DomainPregnancy, true
	case SectionMeds:
		return family.DomainMeds, true
	case SectionAppointments:
		return family.DomainAppointments, true
	case SectionTeenPeriodWeek:
		return family.DomainTeenPeriodWeek, true
	case SectionTeenKit:
		return family.DomainTeenKit, true
	case SectionTeenNotes:
		return family.DomainTeenNotes, true
	}
	return family.DomainNone, false
}

// legacyScope are the v6 domains a legacy section can name (every section, teen card included): the only domains a
// /companions grant edit changes. The v6-only domains (urgent_alert, vitals, household, fetal_activity,
// ttc_fertility) cannot be expressed there and are kept as stored (T-M12-17b).
var legacyScope = func() []family.Domain {
	out := []family.Domain{}
	for _, s := range append(append([]Section{}, Sections...), TeenSections...) {
		if d, ok := sectionDomain(s); ok {
			out = append(out, d)
		}
	}
	return out
}()

// toV6 maps legacy grants to the v6 set (legacy levels capped to each domain's maximum, as the 00110 migration does).
func toV6(g Grants) map[family.Domain]family.Level {
	out := map[family.Domain]family.Level{}
	for s, l := range g {
		d, ok := sectionDomain(s)
		if !ok || l == LevelNone || !l.Valid() {
			continue
		}
		v := family.LevelView
		if l == LevelEdit {
			v = family.LevelWrite
		}
		if spec, ok := family.Lookup(d); ok && v > spec.Max {
			v = spec.Max
		}
		out[d] = v
	}
	return out
}

// legacyOf maps one stored grant row (v6 code, or a not yet migrated legacy one) to the legacy section and level.
func legacyOf(code, level string) (Section, Level, bool) {
	switch Section(code) {
	case SectionCycle, SectionMeds, SectionAppointments, SectionPregnancy, SectionTeenPeriodWeek, SectionTeenKit,
		SectionTeenNotes:
		if l := Level(level); l == LevelView || l == LevelEdit {
			sec := Section(code)
			if l == LevelEdit && (sec == SectionCycle || sec == SectionPregnancy) {
				l = LevelView
			}
			return sec, l, true
		}
	}
	d, ok := family.ParseDomain(code)
	if !ok {
		return "", LevelNone, false
	}
	v, ok := family.ParseLevel(level)
	if !ok || v == family.LevelOff {
		return "", LevelNone, false
	}
	var sec Section
	switch d {
	case family.DomainCyclePhase:
		sec = SectionCycle
	case family.DomainPregnancy, family.DomainMeds, family.DomainAppointments, family.DomainTeenPeriodWeek,
		family.DomainTeenKit, family.DomainTeenNotes:
		sec = Section(d.String())
	default:
		return "", LevelNone, false // v6-only domains have no legacy section
	}
	if v == family.LevelWrite {
		return sec, LevelEdit, true
	}
	return sec, LevelView, true
}

// effectiveLevel is the legacy level a snapshot really gives on a section right now.
func effectiveLevel(s family.Snapshot, sec Section, now time.Time) Level {
	d, ok := sectionDomain(sec)
	if !ok {
		return LevelNone
	}
	if family.Decide(s, family.Subject{}, d, family.ActionRead, now).Outcome == family.Deny {
		return LevelNone
	}
	// Until proposals ship (T-M11-09) a legacy «ثبت برای …» write on a proposable record stays a direct write with
	// the owner notice and audit row (the B-N4-02 behaviour); Decide's Propose counts as edit here.
	if family.Decide(s, family.Subject{}, d, family.ActionWrite, now).Outcome != family.Deny {
		return LevelEdit
	}
	return LevelView
}

// effectiveGrants is the legacy grant set a snapshot really gives a link of type t.
func effectiveGrants(s family.Snapshot, t Type, now time.Time) Grants {
	out := Grants{}
	for _, sec := range SectionsFor(t) {
		if l := effectiveLevel(s, sec, now); l != LevelNone {
			out[sec] = l
		}
	}
	return out
}
