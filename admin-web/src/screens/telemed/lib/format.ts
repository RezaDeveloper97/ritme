import type { BadgeTone } from '@/shared/ui';

/** Prices travel as integer rials; the panel shows and edits toman (1 toman = 10 rials). */
export const rialsToToman = (rials: number): number => Math.round(rials / 10);
export const tomanToRials = (toman: string): number => Math.max(0, Math.round(Number(toman) || 0)) * 10;

const BOOKING_TONE: Record<string, BadgeTone> = {
  confirmed: 'green',
  completed: 'data',
  held: 'amber',
  cancelled: 'neutral',
  expired: 'neutral',
  failed: 'red',
};
const PAYMENT_TONE: Record<string, BadgeTone> = {
  paid: 'green',
  free: 'neutral',
  refunded: 'data',
  refund_pending: 'red',
  pending: 'amber',
  declined: 'red',
  none: 'neutral',
};
export const bookingTone = (s: string): BadgeTone => BOOKING_TONE[s] ?? 'neutral';
export const paymentTone = (s: string): BadgeTone => PAYMENT_TONE[s] ?? 'neutral';

export const BOOKING_TABS = ['all', 'refunds'] as const;
export const PRESCRIPTION_FORMS = ['tablet', 'capsule', 'syrup', 'injection', 'drops'] as const;
