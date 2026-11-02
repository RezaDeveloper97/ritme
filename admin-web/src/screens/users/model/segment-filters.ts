/** Directory filters <-> saved segment filters <-> URL query (admin-api.md §6c). A segment never carries free-text `q`. */

/** The filter names a segment may hold, in display order (server: segments.Keys without `q`). */
export const SEGMENT_KEYS = [
  "status",
  "plan",
  "activity",
  "cohort",
  "churn",
  "mode",
  "data_quality",
  "main_warning",
] as const;
export type SegmentKey = (typeof SEGMENT_KEYS)[number];
export type SegmentFilters = Partial<Record<SegmentKey, string>>;

/** URL defaults of the users list: every directory filter plus the optional segment id. */
export const LIST_DEFAULTS: Record<string, string> = {
  status: "all",
  plan: "",
  activity: "",
  cohort: "",
  churn: "",
  mode: "",
  data_quality: "",
  main_warning: "",
  segment: "",
};

/** The active list filters as a segment payload: empty values and the default `status=all` are dropped. */
export function filtersToSegment(
  filters: Record<string, string>,
): SegmentFilters {
  const out: SegmentFilters = {};
  for (const k of SEGMENT_KEYS) {
    const v = filters[k];
    if (!v || (k === "status" && v === "all")) continue;
    out[k] = v;
  }
  return out;
}

/** Opening a saved segment: its filters as a query string (no leading `?`). */
export function segmentToSearch(filters: SegmentFilters): string {
  const sp = new URLSearchParams();
  for (const k of SEGMENT_KEYS) {
    const v = filters[k];
    if (v) sp.set(k, v);
  }
  return sp.toString();
}

/** Whether anything narrows the list (search text, any filter, a segment). */
export function isFiltered(
  q: string,
  filters: Record<string, string>,
): boolean {
  return (
    Boolean(q) ||
    Object.entries(filters).some(([k, v]) =>
      k === "status" ? v !== "all" : Boolean(v),
    )
  );
}
