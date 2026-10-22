import type { ReportState } from '@/entities/vital';

/**
 * How many more readings the hub card asks for (SPEC v6 report minimum: ≥ 2
 * readings on ≥ 2 days), or null when the report has enough data or there is
 * no reading yet (the card then invites the first one). Pure.
 */
export function readingsNeeded(state: ReportState | null, hasReading: boolean): number | null {
  if (!hasReading || !state || state.status !== 'not_enough_data' || state.readingsNeeded <= 0) return null;
  return state.readingsNeeded;
}
