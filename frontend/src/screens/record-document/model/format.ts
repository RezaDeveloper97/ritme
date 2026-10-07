import { type DocumentKind, EXTRACT_FIELDS, type Extraction, type RecordDocument } from '@/entities/health-record';

/* Pure display helpers of the document screen (CB-REC-04). */

/** A row of «اطلاعات خوانده‌شده از سند»: one schema key, or the merged gestational age. */
export type ReadRow = { key: string; value: string | number } | { key: 'ga'; weeks: number; days: number | null };

/**
 * The confirmed values in the kind's order, empty ones dropped, `ga_weeks` + `ga_days` merged into one «سن بارداری»
 * row (the board shows «۸ هفته و ۶ روز»).
 */
export function confirmedRows(kind: DocumentKind, e: Extraction | null): ReadRow[] {
  if (!e) return [];
  const values: Record<string, string | number | null | undefined> = e.reviewed ?? {};
  const rows: ReadRow[] = [];
  for (const key of EXTRACT_FIELDS[kind]) {
    if (key === 'ga_days') continue;
    const v = values[key];
    if (key === 'ga_weeks') {
      if (typeof v === 'number') rows.push({ key: 'ga', weeks: v, days: typeof values.ga_days === 'number' ? values.ga_days : null });
      continue;
    }
    if (v === null || v === undefined || v === '') continue;
    rows.push({ key, value: v });
  }
  return rows;
}

/** The detail rows the user filled in or confirmed (not the AI suggestions). */
export const DETAIL_KEYS = ['title', 'date', 'ended_on', 'centre', 'doctor', 'note'] as const;
export type DetailKey = (typeof DETAIL_KEYS)[number];

export function detailValue(doc: RecordDocument, key: DetailKey): string | null {
  switch (key) {
    case 'title':
      return doc.title;
    case 'date':
      return doc.date;
    case 'ended_on':
      return doc.kind === 'hospital' ? doc.endedOn : null;
    case 'centre':
      return doc.centre;
    case 'doctor':
      return doc.doctor;
    case 'note':
      return doc.note;
  }
}

/** Size in KB (< 1 MB) or MB with one decimal, as a number for the locale formatter. */
export function sizeOf(bytes: number): { unit: 'kb' | 'mb'; value: number } {
  if (bytes < 1024 * 1024) return { unit: 'kb', value: Math.max(1, Math.round(bytes / 1024)) };
  return { unit: 'mb', value: Math.round((bytes / 1024 / 1024) * 10) / 10 };
}
