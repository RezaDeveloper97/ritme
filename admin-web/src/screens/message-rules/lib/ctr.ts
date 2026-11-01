import type { Stat, VersionStat } from '../api/rules';

export interface VersionRow {
  version: number;
  variants: { variant: string; impressions: number; clicks: number; ctr: number | null }[];
  impressions: number;
  clicks: number;
  ctr: number | null;
}

const rate = (clicks: number, impressions: number) => (impressions > 0 ? clicks / impressions : null);

/** `stats_by_version` grouped per version (newest first), each with its variants and a version total. */
export function groupByVersion(rows: readonly VersionStat[]): VersionRow[] {
  const map = new Map<number, VersionRow>();
  for (const r of rows) {
    const row = map.get(r.version) ?? { version: r.version, variants: [], impressions: 0, clicks: 0, ctr: null };
    row.variants.push({ variant: r.variant, impressions: r.impressions, clicks: r.clicks, ctr: rate(r.clicks, r.impressions) });
    row.impressions += r.impressions;
    row.clicks += r.clicks;
    map.set(r.version, row);
  }
  return [...map.values()]
    .map((r) => ({ ...r, ctr: rate(r.clicks, r.impressions), variants: [...r.variants].sort((a, b) => a.variant.localeCompare(b.variant)) }))
    .sort((a, b) => b.version - a.version);
}

export const ctrOf = (s: Pick<Stat, 'impressions' | 'clicks'>) => rate(s.clicks, s.impressions);
