import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

import { A4_POINTS, type PdfImagePage, buildImagePdf } from './writer';

/**
 * Canvas → PDF text document. Loaded lazily through `loadPdfGenerator()` so the
 * code never ships with a screen's first paint. Everything happens on the
 * device; nothing is uploaded.
 */

export type PdfBlockKind = 'title' | 'subtitle' | 'heading' | 'text' | 'muted' | 'rule';

export interface PdfBlock {
  kind: PdfBlockKind;
  text?: string;
}

export interface PdfDocumentSpec {
  blocks: readonly PdfBlock[];
  dir: 'rtl' | 'ltr';
  /** CSS font-family; defaults to the page's body font (Vazirmatn in the app). */
  fontFamily?: string;
  /**
   * Footer on every page, e.g. the app name + date. `{page}`/`{pages}` are
   * replaced, so pass the raw message template (`t.raw(...)`), not `t(...)`.
   */
  footer?: string;
  /** Locale for the footer's page numbers (fa → Persian digits). */
  locale: Locale;
}

/** Fill a footer template's `{page}`/`{pages}` with locale digits. */
export function formatPdfFooter(template: string, page: number, pages: number, locale: Locale): string {
  return template
    .replace('{page}', formatNumber(page, locale))
    .replace('{pages}', formatNumber(pages, locale));
}

// A4 at ~144 dpi: sharp enough to read on a phone and to print.
const SCALE = 2;
const PAGE_W = Math.round(A4_POINTS.width * SCALE);
const PAGE_H = Math.round(A4_POINTS.height * SCALE);
const MARGIN = 48 * SCALE;
const FOOTER_H = 28 * SCALE;

// Paper, not UI: the PDF is always black-on-white whatever the app theme, so
// these are fixed CSS named colours rather than theme tokens (which flip).
const STYLE: Record<Exclude<PdfBlockKind, 'rule'>, { size: number; weight: number; color: string; gap: number }> = {
  title: { size: 20, weight: 800, color: 'black', gap: 6 },
  subtitle: { size: 11, weight: 500, color: 'dimgray', gap: 14 },
  heading: { size: 13, weight: 800, color: 'black', gap: 4 },
  text: { size: 11, weight: 500, color: 'darkslategray', gap: 3 },
  muted: { size: 9.5, weight: 500, color: 'gray', gap: 8 },
};

interface Line {
  text: string;
  kind: PdfBlockKind;
}

function wrap(ctx: CanvasRenderingContext2D, text: string, width: number): string[] {
  const out: string[] = [];
  for (const paragraph of text.split('\n')) {
    let line = '';
    for (const word of paragraph.split(/\s+/).filter(Boolean)) {
      const next = line ? `${line} ${word}` : word;
      if (line && ctx.measureText(next).width > width) {
        out.push(line);
        line = word;
      } else {
        line = next;
      }
    }
    out.push(line);
  }
  return out;
}

function toJpeg(canvas: HTMLCanvasElement): Promise<Uint8Array> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) return reject(new Error('canvas.toBlob failed'));
        blob.arrayBuffer().then((b) => resolve(new Uint8Array(b)), reject);
      },
      'image/jpeg',
      0.9,
    );
  });
}

export async function renderPdf(spec: PdfDocumentSpec): Promise<Blob> {
  const family = spec.fontFamily ?? getComputedStyle(document.body).fontFamily;
  const font = (kind: Exclude<PdfBlockKind, 'rule'>) =>
    `${STYLE[kind].weight} ${STYLE[kind].size * SCALE}px ${family}`;
  // Make sure every weight is loaded before measuring, or the fallback font is drawn.
  await Promise.all((['title', 'text', 'heading'] as const).map((k) => document.fonts.load(font(k), 'آب')));

  const canvas = document.createElement('canvas');
  canvas.width = PAGE_W;
  canvas.height = PAGE_H;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('2d canvas unavailable');
  const contentW = PAGE_W - MARGIN * 2;

  // Lay out: split blocks into lines and pages.
  const pagesLines: Array<Array<Line & { y: number }>> = [[]];
  let y = MARGIN;
  for (const block of spec.blocks) {
    if (block.kind === 'rule') {
      const h = 14 * SCALE;
      if (y + h > PAGE_H - MARGIN - FOOTER_H) {
        pagesLines.push([]);
        y = MARGIN;
      }
      pagesLines[pagesLines.length - 1].push({ text: '', kind: 'rule', y: y + h / 2 });
      y += h;
      continue;
    }
    const st = STYLE[block.kind];
    ctx.font = font(block.kind);
    const lineH = st.size * SCALE * 1.7;
    for (const text of wrap(ctx, block.text ?? '', contentW)) {
      if (y + lineH > PAGE_H - MARGIN - FOOTER_H) {
        pagesLines.push([]);
        y = MARGIN;
      }
      pagesLines[pagesLines.length - 1].push({ text, kind: block.kind, y: y + st.size * SCALE * 1.2 });
      y += lineH;
    }
    y += st.gap * SCALE;
  }

  const rtl = spec.dir === 'rtl';
  const pages: PdfImagePage[] = [];
  for (let p = 0; p < pagesLines.length; p++) {
    ctx.fillStyle = 'white';
    ctx.fillRect(0, 0, PAGE_W, PAGE_H);
    ctx.direction = spec.dir;
    ctx.textAlign = rtl ? 'right' : 'left';
    const x = rtl ? PAGE_W - MARGIN : MARGIN;
    for (const line of pagesLines[p]) {
      if (line.kind === 'rule') {
        ctx.fillStyle = 'gainsboro';
        ctx.fillRect(MARGIN, line.y, contentW, SCALE);
        continue;
      }
      ctx.font = font(line.kind);
      ctx.fillStyle = STYLE[line.kind].color;
      ctx.fillText(line.text, x, line.y);
    }
    if (spec.footer) {
      ctx.font = font('muted');
      ctx.fillStyle = STYLE.muted.color;
      ctx.textAlign = 'center';
      const footer = formatPdfFooter(spec.footer, p + 1, pagesLines.length, spec.locale);
      ctx.fillText(footer, PAGE_W / 2, PAGE_H - MARGIN / 2);
    }
    pages.push({ jpeg: await toJpeg(canvas), pixelWidth: PAGE_W, pixelHeight: PAGE_H });
  }
  const bytes = buildImagePdf(pages);
  return new Blob([bytes as BlobPart], { type: 'application/pdf' });
}
