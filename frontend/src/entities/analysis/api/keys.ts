import type { AnalysisRangeKey } from '../model/types';

/**
 * Query-key factory for `/analysis/*` (CLAUDE.md §8). Every report is keyed by
 * its range, so switching «۳ ماه / ۶ ماه / ۱ سال» caches each window. A day log
 * or a Plus change can refresh the whole tab through {@link analysisKeys.all}.
 * B-N3-09…12 add their reports as `report(name, range)`.
 */
export const analysisKeys = {
  all: ['analysis'] as const,
  summary: (range: AnalysisRangeKey) => [...analysisKeys.all, 'summary', range] as const,
  /** Detail reports (`cycle`, `period`, `symptoms`, `correlations`, `body`, …). */
  report: (name: string, range: AnalysisRangeKey) => [...analysisKeys.all, name, range] as const,
  monthly: (ym: string, calendar: string) => [...analysisKeys.all, 'monthly', calendar, ym] as const,
};
