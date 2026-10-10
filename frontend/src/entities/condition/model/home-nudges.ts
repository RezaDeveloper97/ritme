import type { ConditionNudge, ConditionsOverview } from './types';

/**
 * The nudges the home shows (CB-COND-07b): the API's hits minus any whose program she is already in. The server
 * already skips enrolled programs; this keeps a stale cached answer from showing the card for a moment after joining.
 */
export function homeNudges(
  nudges: readonly ConditionNudge[] | undefined,
  overview: Pick<ConditionsOverview, 'programs'> | undefined,
): ConditionNudge[] {
  if (!nudges?.length) return [];
  const enrolled = new Set((overview?.programs ?? []).filter((p) => p.enrolled).map((p) => p.code));
  return nudges.filter((n) => !enrolled.has(n.program));
}
