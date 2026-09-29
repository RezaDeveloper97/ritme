/**
 * «۱ روز قبل یادآوری» is shown only while the appointment's reminder bell is on. `GET /care/today`
 * does not carry the switch, so the row reads it from the appointment itself; until that answer
 * arrives (or if it fails) the chip keeps today's behaviour (QA 2026-09-29-c L4).
 */
export function showRemindChip(appointment: { isActive: boolean } | undefined): boolean {
  return appointment?.isActive !== false;
}
