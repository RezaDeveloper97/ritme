package care

import "time"

// Delegated single-record access stays inside the companion's section view (B-N4-08b, CMP-M2): the meds section shows
// only active medications and the appointments section only upcoming, non-private ones (internal/companion/shared),
// so GET / PUT of one record with for_user_id answers the same 404 as an unknown id for anything else — a paused or
// ended medication, a past or cancelled visit, a private follow-up. An edit grant cannot reopen history either.

// outsideMedicationView reports whether a companion (actorID ≠ subjectID) may not open or edit the medication.
func outsideMedicationView(m Medication, subjectID, actorID uint64) bool {
	return subjectID != actorID && !m.Row.IsActive
}

// outsideAppointmentView reports whether a companion (actorID ≠ subjectID) may not open or edit the appointment at
// now.
func outsideAppointmentView(a Appointment, subjectID, actorID uint64, now time.Time) bool {
	return subjectID != actorID && (a.HiddenFrom(subjectID, actorID) || !a.Upcoming(now))
}
