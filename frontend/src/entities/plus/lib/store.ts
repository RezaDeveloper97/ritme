import type { PlusCatalog, PlusPlan } from '../model/types';

type Store = 'bazaar' | 'myket';

/** Whether this store build can sell Plus: the server has the store configured. */
export function storeAvailable(catalog: PlusCatalog, store: Store): boolean {
  return catalog.storeChannels.includes(store);
}

/** The plans buyable through `store` (those with a SKU for it). */
export function plansForChannel(plans: readonly PlusPlan[], store: Store): PlusPlan[] {
  return plans.filter((p) => !!p.storeSkus[store]);
}

export type PlanDuration = { unit: 'minutes'; value: number } | { unit: 'months'; value: number };

/** A plan's length: test plans are minutes (no per-month / savings lines), the rest months. */
export function planDuration(plan: Pick<PlusPlan, 'durationMinutes' | 'durationMonths'>): PlanDuration {
  return plan.durationMinutes != null
    ? { unit: 'minutes', value: plan.durationMinutes }
    : { unit: 'months', value: plan.durationMonths };
}
