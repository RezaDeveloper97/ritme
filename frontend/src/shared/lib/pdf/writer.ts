/**
 * A minimal PDF 1.4 writer: one full-bleed JPEG per page (DCTDecode). Pure —
 * no DOM — so it is unit-tested in Node. The text is rendered by the browser
 * onto a canvas first (see `render.ts`), which is what gives Persian correct
 * shaping and bidi with the app's own Vazirmatn font: the glyphs are embedded
 * as pixels, so the PDF opens identically in every viewer.
 */

export interface PdfImagePage {
  /** Baseline JPEG bytes. */
  jpeg: Uint8Array;
  /** Pixel size of the JPEG. */
  pixelWidth: number;
  pixelHeight: number;
}

/** A4 in PDF points. */
export const A4_POINTS = { width: 595.28, height: 841.89 } as const;

const encoder = new TextEncoder();

export function buildImagePdf(
  pages: readonly PdfImagePage[],
  size: { width: number; height: number } = A4_POINTS,
): Uint8Array {
  if (pages.length === 0) throw new Error('buildImagePdf: no pages');
  const chunks: Uint8Array[] = [];
  const offsets: number[] = [];
  let length = 0;
  const push = (part: string | Uint8Array) => {
    const bytes = typeof part === 'string' ? encoder.encode(part) : part;
    chunks.push(bytes);
    length += bytes.length;
  };
  const object = (id: number, body: () => void) => {
    offsets[id] = length;
    push(`${id} 0 obj\n`);
    body();
    push('\nendobj\n');
  };

  const w = size.width.toFixed(2);
  const h = size.height.toFixed(2);
  // 1 catalog, 2 pages, then per page: page, content, image.
  const pageId = (i: number) => 3 + i * 3;
  const total = 2 + pages.length * 3;

  push('%PDF-1.4\n%âãÏÓ\n');
  object(1, () => push('<< /Type /Catalog /Pages 2 0 R >>'));
  object(2, () =>
    push(`<< /Type /Pages /Count ${pages.length} /Kids [${pages.map((_, i) => `${pageId(i)} 0 R`).join(' ')}] >>`),
  );
  pages.forEach((page, i) => {
    const id = pageId(i);
    const content = `q ${w} 0 0 ${h} 0 0 cm /Im0 Do Q`;
    object(id, () =>
      push(
        `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${w} ${h}] ` +
          `/Resources << /XObject << /Im0 ${id + 2} 0 R >> >> /Contents ${id + 1} 0 R >>`,
      ),
    );
    object(id + 1, () => push(`<< /Length ${content.length} >>\nstream\n${content}\nendstream`));
    object(id + 2, () => {
      push(
        `<< /Type /XObject /Subtype /Image /Width ${page.pixelWidth} /Height ${page.pixelHeight} ` +
          `/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ${page.jpeg.length} >>\nstream\n`,
      );
      push(page.jpeg);
      push('\nendstream');
    });
  });

  const xref = length;
  let table = `xref\n0 ${total + 1}\n0000000000 65535 f \n`;
  for (let id = 1; id <= total; id++) table += `${String(offsets[id]).padStart(10, '0')} 00000 n \n`;
  push(table);
  push(`trailer\n<< /Size ${total + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`);

  const out = new Uint8Array(length);
  let at = 0;
  for (const c of chunks) {
    out.set(c, at);
    at += c.length;
  }
  return out;
}
