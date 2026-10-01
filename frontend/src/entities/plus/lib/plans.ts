import type { PlusPlan, PlusStatus } from '../model/types';

/** The plan pre-selected on «انتخاب اشتراک»: the highlighted one, else the first. */
export function defaultPlan(plans: readonly PlusPlan[]): PlusPlan | null {
  return plans.find((p) => p.isHighlighted) ?? plans[0] ?? null;
}

/** `?plan=<id>` → that plan when it exists in the catalogue, else the default. */
export function planFromParam(plans: readonly PlusPlan[], param: string | null): PlusPlan | null {
  const id = Number(param);
  return (Number.isInteger(id) && plans.find((p) => p.id === id)) || defaultPlan(plans);
}

/**
 * «ارتقا به ۶ ماهه» on the manage screen: the longest plan longer than the
 * current one (null when the user already has the longest, or has no plan).
 */
export function upgradePlan(plans: readonly PlusPlan[], currentMonths: number | null): PlusPlan | null {
  if (currentMonths === null) return null;
  const longer = plans.filter((p) => p.durationMonths > currentMonths);
  return longer.reduce<PlusPlan | null>((best, p) => (!best || p.durationMonths > best.durationMonths ? p : best), null);
}

/** Share of the current period still ahead (0–1), for the «روز مانده» ring. */
export function remainingShare(daysLeft: number, startsAt: string, endsAt: string): number {
  const total = (Date.parse(endsAt) - Date.parse(startsAt)) / 86_400_000;
  if (!Number.isFinite(total) || total <= 0) return 0;
  return Math.max(0, Math.min(1, daysLeft / Math.ceil(total)));
}

/** What the manage screen is about: a paid subscription, a running trial, or nothing to manage. */
export type PlusMembership =
  | { kind: 'subscription'; status: PlusStatus }
  | { kind: 'trial'; status: PlusStatus }
  | { kind: 'none'; status: PlusStatus };

export function membershipOf(status: PlusStatus): PlusMembership {
  if (status.subscription && status.subscription.daysLeft > 0) return { kind: 'subscription', status };
  if (status.trial?.isActive) return { kind: 'trial', status };
  return { kind: 'none', status };
}

export interface GatewayReturn {
  reference: string;
  authority: string;
  /** `OK` or `NOK`, as normalized by the API's return hop. */
  status: string;
}

/**
 * The query the API's return hop (`/api/v1/payments/{provider}/return`) adds
 * to `PLUS_CALLBACK_URL`. Null when a parameter is missing (opened by hand).
 */
export function parseGatewayReturn(params: { get(name: string): string | null }): GatewayReturn | null {
  const reference = params.get('reference')?.trim() ?? '';
  const authority = params.get('authority')?.trim() ?? '';
  if (!reference || !authority || reference.length > 64 || authority.length > 191) return null;
  return { reference, authority, status: params.get('status') === 'OK' ? 'OK' : 'NOK' };
}
