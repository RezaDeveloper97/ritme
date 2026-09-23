/**
 * Tags the backend sanitizer keeps (mirror of `allowed` in
 * backend-go/internal/content/sanitizer/clean.go). The editor must never emit
 * anything outside this list — rich-text.test.ts checks every node and mark of
 * `editorExtensions` (editor-extensions.ts) against it.
 */
export const SANITIZER_ALLOWED_TAGS: ReadonlySet<string> = new Set([
  'p', 'br', 'strong', 'b', 'em', 'i', 'u', 's', 'mark', 'sub', 'sup', 'span',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'blockquote', 'pre', 'code',
  'hr', 'figure', 'figcaption', 'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td',
  'a', 'img',
]);

/** Attributes the sanitizer keeps per tag (others are dropped on save). */
export const SANITIZER_ALLOWED_ATTRS: Readonly<Record<string, readonly string[]>> = {
  a: ['href', 'title', 'target', 'rel'],
  img: ['src', 'alt', 'width', 'height'],
  th: ['colspan', 'rowspan'],
  td: ['colspan', 'rowspan'],
};

/** '' for an empty document; trims the outer whitespace. */
export function normalizeEditorHtml(html: string): string {
  const trimmed = html.trim();
  return trimmed === '<p></p>' ? '' : trimmed;
}
