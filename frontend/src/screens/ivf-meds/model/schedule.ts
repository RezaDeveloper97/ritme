import {
  IVF_INJECTED_ROUTES,
  type IvfDose,
  type IvfInjectionSite,
  type IvfInventory,
  type IvfRoute,
  type IvfSites,
} from '@/entities/ivf';
import { diffInDays, fromApiDate } from '@/shared/lib/date';

/*
 * Pure helpers of the injection schedule (nbl_IVF_Meds, CB-IVF-03). No React,
 * no locale: the UI turns the keys into copy.
 */

/** Site codes the bundled copy knows (catalog `ivf_injection_sites` seed); others show the catalog title. */
export const KNOWN_SITES = [
  'abdomen_upper_right',
  'abdomen_upper_left',
  'thigh_right',
  'thigh_left',
  'abdomen_lower_right',
  'abdomen_lower_left',
  'arm_right',
  'arm_left',
] as const;
export type KnownSite = (typeof KNOWN_SITES)[number];

export function isKnownSite(code: string): code is KnownSite {
  return (KNOWN_SITES as readonly string[]).includes(code);
}

/** Does a dose of this route take an injection site? */
export function takesSite(route: IvfRoute): boolean {
  return IVF_INJECTED_ROUTES.includes(route);
}

/**
 * The site today's next injection is logged with: what she picked (if it is
 * still an active code), else the API's least-recently-used suggestion, else
 * the first site in rotation order.
 */
export function chosenSite(sites: IvfSites, picked: string | null): string | null {
  if (picked && sites.codes.includes(picked)) return picked;
  if (sites.suggested && sites.codes.includes(sites.suggested)) return sites.suggested;
  return sites.codes[0] ?? null;
}

/** Catalog title of a site, or null → the bundled name / the raw code. */
export function siteTitle(code: string, catalog: readonly IvfInjectionSite[] | undefined): string | null {
  return catalog?.find((s) => s.code === code)?.title ?? null;
}

/** Catalog hint of a site («حداقل ۵ سانت دور از ناف…»), if any. */
export function siteHint(code: string | null, catalog: readonly IvfInjectionSite[] | undefined): string | null {
  if (!code) return null;
  return catalog?.find((s) => s.code === code)?.body ?? null;
}

/** The rotation line under the picker: last / today's suggestion vs her own pick. */
export function rotationNote(
  sites: IvfSites,
  chosen: string | null,
): { last: string | null; today: { site: string; kind: 'suggested' | 'picked' } | null } {
  return {
    last: sites.last?.site ?? null,
    today: chosen ? { site: chosen, kind: chosen === sites.suggested ? 'suggested' : 'picked' } : null,
  };
}

/** The site a log of this dose sends (null for a non-injected medicine — the API rejects it). */
export function siteForLog(dose: Pick<IvfDose, 'route'>, chosen: string | null): string | null {
  return takesSite(dose.route) ? chosen : null;
}

export type StockState = 'low' | 'ok' | 'none';

/** Inventory badge: «کم است» / «کافی», none when the stock is not tracked. */
export function stockState(inventory: IvfInventory | null): StockState {
  if (!inventory) return 'none';
  return inventory.low ? 'low' : 'ok';
}

/**
 * «کافی تا جمعه»: how the run-out day is named — a weekday within the coming
 * week, a date further out, nothing when it is unknown (not scheduled today).
 */
export function runsOutLabel(inventory: IvfInventory | null, today: string): { kind: 'weekday' | 'date'; on: string } | null {
  if (!inventory?.runsOutOn || inventory.daysLeft === null) return null;
  const days = diffInDays(fromApiDate(inventory.runsOutOn), fromApiDate(today));
  if (Number.isNaN(days) || days < 0) return null;
  return { kind: days <= 6 ? 'weekday' : 'date', on: inventory.runsOutOn };
}

/** Lowest-stock first, then by name — the low badges lead the list. */
export function sortForInventory<T extends { name: string; inventory: IvfInventory | null }>(meds: readonly T[]): T[] {
  const rank = (m: T) => (m.inventory?.low ? 0 : m.inventory ? 1 : 2);
  return [...meds].sort((a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name));
}
