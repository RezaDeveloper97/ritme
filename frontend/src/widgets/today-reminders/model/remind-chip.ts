/**
 * «۱ روز قبل یادآوری» is shown only while the appointment's reminder bell is on
 * (QA 2026-09-29-c L4). `GET /care/today` carries the switch as `is_active`, so the
 * row needs no request of its own (T-M7-23).
 */
export function showRemindChip(appointment: { isActive: boolean }): boolean {
  return appointment.isActive;
}
