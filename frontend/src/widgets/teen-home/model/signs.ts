import type { TeenCatalogItem } from '@/entities/teen';

/*
 * The «signs» card of the teen home (nbl_Teen_Home). Before the first period
 * the card leads with the first approach sign and closes with the estimate
 * line; afterwards the estimate item («سال اول: …») is the whole card.
 */

/**
 * How far along the decorative «getting closer» bar is for an estimate code.
 * Only the two calm «before the first period» estimates draw it; any other
 * (or unknown, admin-added) code draws no bar rather than a made-up value.
 */
const ESTIMATE_PERCENT: Record<string, number> = {
  estimate_year_or_two: 35,
  estimate_coming_months: 65,
};

export function estimatePercent(readiness: TeenCatalogItem | null): number | null {
  if (!readiness) return null;
  return ESTIMATE_PERCENT[readiness.code] ?? null;
}

export interface SignsCard {
  title: string | null;
  body: string | null;
  /** The other approach signs (growth spurt …), one short row each. */
  extra: TeenCatalogItem[];
  /** «بر اساس جواب‌هایت: …» under the bar; null when the estimate is the card itself. */
  estimate: string | null;
  percent: number | null;
  caution: boolean;
}

export function signsCard(signs: TeenCatalogItem[], readiness: TeenCatalogItem | null): SignsCard | null {
  const [lead, ...extra] = signs;
  const caution = readiness?.severity === 'caution';
  if (lead) {
    return {
      title: lead.title,
      body: lead.body,
      extra,
      estimate: readiness?.title ?? null,
      percent: caution ? null : estimatePercent(readiness),
      caution,
    };
  }
  if (!readiness) return null;
  return { title: readiness.title, body: readiness.body, extra: [], estimate: null, percent: null, caution };
}
