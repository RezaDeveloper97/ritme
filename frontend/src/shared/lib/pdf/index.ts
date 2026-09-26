/**
 * Client-side PDF generation (canvas-rendered pages → PDF). The renderer is
 * heavy-ish and only needed on demand, so it is exposed through a lazy loader:
 * `const { renderPdf } = await loadPdfGenerator()`.
 */
export type { PdfBlock, PdfBlockKind, PdfDocumentSpec } from './render';
export { A4_POINTS, buildImagePdf, type PdfImagePage } from './writer';
export { shareOrDownloadFile } from './share';

export function loadPdfGenerator(): Promise<typeof import('./render')> {
  return import('./render');
}
