import type { LabQuota } from '@/entities/plus';

export type LabQuotaNote =
  | { key: 'available' | 'plusAvailable'; remaining: number; limit: number }
  | { key: 'spent'; limit: number };

/**
 * The quota line above the upload form: the free plan's monthly allowance
 * (read from the entitlement, admin-tunable) or the Plus one. Null for an
 * unlimited / locked / unknown quota (the Plus gate or nothing shows instead).
 */
export function labQuotaNote(quota: LabQuota | null, isPlus: boolean): LabQuotaNote | null {
  if (!quota || quota.state === 'unlimited' || quota.state === 'locked') return null;
  if (quota.state === 'spent') return { key: 'spent', limit: quota.limit };
  return { key: isPlus ? 'plusAvailable' : 'available', remaining: quota.remaining, limit: quota.limit };
}

/** The free quota is spent: the form is replaced by the paywall (a Plus user's spent quota is the server's 429 note). */
export function labPaywalled(note: LabQuotaNote | null, isPlus: boolean): boolean {
  return note?.key === 'spent' && !isPlus;
}
