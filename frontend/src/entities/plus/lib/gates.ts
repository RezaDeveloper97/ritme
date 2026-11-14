import { entitlementKey } from './keys';
import type { PlusEntitlement, PlusStatus } from '../model/types';

export {
  PLUS_ANALYSIS_LONG_RANGE,
  PLUS_COMPANION,
  PLUS_DEEP_ANALYSIS,
  PLUS_SMS_REMINDERS,
} from './keys';

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

/** A 402 on a long analysis range: render the paywall card, not an error. */
export function isRangeDenial(denial: { kind: string; feature: string } | null | undefined): boolean {
  return !!denial && denial.kind === 'locked' && denial.feature === entitlementKey.longRange;
}
