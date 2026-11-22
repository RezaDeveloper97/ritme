/*
 * The companion calendar overlay (GET /companion/calendar, Go only): per partner link, the days her period is likely
 * to fall on — today onwards only, and only when she turned on «روزهای احتمالی پریود» for this link and shares the
 * cycle section. Predictions, never her logged history or cycle numbers.
 */

export interface CompanionCalendarPartner {
  linkId: number;
  partnerName: string | null;
  /** `Y-m-d` days; `null` = she does not share them (≠ `[]`: shared, nothing predicted in the window, or she is late). */
  periodDays: string[] | null;
}

export interface CompanionCalendar {
  from: string;
  to: string;
  partners: CompanionCalendarPartner[];
}
