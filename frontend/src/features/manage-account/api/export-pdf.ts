import type { Locale } from '@/shared/i18n';
import type { PdfBlock, PdfDocumentSpec } from '@/shared/lib/pdf';

/** Copy the PDF needs; owned by the screen that offers the export. */
export interface PdfExportLabels {
  title: string;
  subtitle: string;
  /** Raw footer template with `{page}` / `{pages}` (pass `t.raw`, not `t`). */
  footer: string;
  dir: 'rtl' | 'ltr';
  locale: Locale;
}

/** Longest value printed per field — a stray blob must not produce 200 pages. */
const MAX_VALUE = 600;

function scalar(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

function clip(text: string): string {
  return text.length > MAX_VALUE ? `${text.slice(0, MAX_VALUE)}…` : text;
}

function recordLines(record: Record<string, unknown>): string {
  return Object.entries(record)
    .map(([k, v]) => `${k}: ${clip(scalar(v))}`)
    .join('\n');
}

/**
 * The export payload (GET /profile/export, opaque here) as PDF blocks: one
 * heading per top-level key, one paragraph per record. Keys stay as the API
 * names them — the PDF is the same data as the JSON, only printable.
 */
export function exportPdfSpec(payload: Record<string, unknown>, labels: PdfExportLabels): PdfDocumentSpec {
  const blocks: PdfBlock[] = [
    { kind: 'title', text: labels.title },
    { kind: 'subtitle', text: labels.subtitle },
  ];
  for (const [key, value] of Object.entries(payload)) {
    blocks.push({ kind: 'heading', text: key });
    const items = Array.isArray(value) ? value : [value];
    if (items.length === 0) blocks.push({ kind: 'muted', text: '—' });
    for (const item of items) {
      const text =
        item && typeof item === 'object' && !Array.isArray(item)
          ? recordLines(item as Record<string, unknown>)
          : clip(scalar(item));
      blocks.push({ kind: 'text', text });
      blocks.push({ kind: 'rule' });
    }
  }
  return { blocks, dir: labels.dir, footer: labels.footer, locale: labels.locale };
}
