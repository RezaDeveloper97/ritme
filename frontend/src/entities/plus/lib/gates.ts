import { entitlementKey } from './keys';
import type { PlusEntitlement, PlusStatus } from '../model/types';

export { PLUS_ANALYSIS_LONG_RANGE, PLUS_ASSISTANT, PLUS_DEEP_ANALYSIS, PLUS_LAB_AI } from './keys';

/** The analysis ranges that need Plus (A-03/D-178): a year or everything. 7d/2w/1m/3m/6m stay free. */
const LONG_RANGES: readonly string[] = ['1y', 'all'];

export function isLongRange(range: string): boolean {
  return LONG_RANGES.includes(range);
}

/** The entitlement row of `key` in a status, null while the status is unknown. */
export function findEntitlement(status: PlusStatus | undefined, key: string): PlusEntitlement | null {
  return status?.entitlements.find((e) => e.key === key) ?? null;
}

/** Locked only when the status says so; an unknown status or missing row never locks (the API gate decides). */
export function featureLocked(status: PlusStatus | undefined, key: string): boolean {
  const e = findEntitlement(status, key);
  return e !== null && !e.allowed;
}

/** Whether the 1-year / all range is locked for this user, so the request should not be made at all. */
export function rangeLocked(status: PlusStatus | undefined, range: string): boolean {
  return isLongRange(range) && featureLocked(status, entitlementKey.longRange);
}

/** A-11: the "why this date" breakdown, the BBT coverline/confirmation and the LH explanation. */
export function detailLocked(status: PlusStatus | undefined): boolean {
  return featureLocked(status, entitlementKey.deep);
}

export type LabQuota =
  | { state: 'unlimited' }
  | { state: 'available'; limit: number; remaining: number }
  | { state: 'spent'; limit: number }
  | { state: 'locked' };

/** Lab AI quota line input, from the entitlement's limit/remaining (never a hardcoded number). */
export function labQuotaOf(e: PlusEntitlement | null): LabQuota | null {
  if (!e) return null;
  if (!e.allowed) return { state: 'locked' };
  if (e.unlimited || e.limit === null) return { state: 'unlimited' };
  const remaining = Math.max(0, e.remaining ?? e.limit - e.used);
  return remaining > 0 ? { state: 'available', limit: e.limit, remaining } : { state: 'spent', limit: e.limit };
}

/**
 * A Plus user's assistant fair-use race (429 `plus_quota_exceeded` on
 * `plus.assistant_unlimited`): a polite message, never the paywall.
 */
export function isAssistantFairUse(denial: { kind: string; feature: string } | null | undefined): boolean {
  return !!denial && denial.kind === 'exhausted' && denial.feature === entitlementKey.assistant;
}

/** A 402 on a long analysis range: render the paywall card, not an error. */
export function isRangeDenial(denial: { kind: string; feature: string } | null | undefined): boolean {
  return !!denial && denial.kind === 'locked' && denial.feature === entitlementKey.longRange;
}
