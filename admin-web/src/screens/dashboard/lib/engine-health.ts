/** View-models of the engine health card (T-M10-34, SPEC v6 §5.1). */

export interface BucketInput {
  key: string;
  pct: number | null;
  count: number | null;
  suppressed: boolean;
}
export interface DistributionInput {
  of: number | null;
  suppressed: boolean;
  buckets: BucketInput[];
}

export interface BucketRow {
  key: string;
  pct: number | null;
  count: number | null;
  /** Hidden bucket: shown as «کمتر از حد نمایش», never as 0. */
  hidden: boolean;
}

/** Buckets in a stable display order; unknown keys go last. A suppressed distribution hides every bucket. */
export function bucketRows(d: DistributionInput | null | undefined, order: readonly string[]): BucketRow[] {
  if (!d) return [];
  const rank = (k: string) => {
    const i = order.indexOf(k);
    return i < 0 ? order.length : i;
  };
  return [...d.buckets]
    .sort((a, b) => rank(a.key) - rank(b.key))
    .map((b) => {
      const hidden = d.suppressed || b.suppressed || b.pct == null;
      return { key: b.key, pct: hidden ? null : b.pct, count: hidden ? null : b.count, hidden };
    });
}

/** The «MAE» accuracy line is only meaningful from real predictions; a suppressed engine shows nothing. */
export function engineReadable(e: { suppressed: boolean; mean_error_days: number | null } | null | undefined): boolean {
  return !!e && !e.suppressed && e.mean_error_days != null;
}
