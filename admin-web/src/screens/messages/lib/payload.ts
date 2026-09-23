/**
 * The smart-message payload editor (messages/edit.blade.php): list fields are
 * edited as one item per line, short texts as a single-line input, long ones
 * (> 60 characters) as a textarea. The API keeps the stored keys and shape.
 */
export type PayloadValue = string | string[];
export type FieldKind = 'list' | 'line' | 'text';

export function fieldKind(value: PayloadValue): FieldKind {
  if (Array.isArray(value)) return 'list';
  return Array.from(value).length > 60 ? 'text' : 'line';
}

/** Editor draft: every field as a string (lists joined by newlines). */
export function toDraft(payload: Record<string, PayloadValue>): Record<string, string> {
  return Object.fromEntries(Object.entries(payload).map(([k, v]) => [k, Array.isArray(v) ? v.join('\n') : v]));
}

/** Draft back to the PUT body: list fields as trimmed, non-empty lines. */
export function fromDraft(
  payload: Record<string, PayloadValue>,
  draft: Record<string, string>,
): Record<string, PayloadValue> {
  const out: Record<string, PayloadValue> = {};
  for (const [key, original] of Object.entries(payload)) {
    const text = draft[key] ?? '';
    out[key] = Array.isArray(original)
      ? text
          .split(/\r\n|\r|\n/)
          .map((l) => l.trim())
          .filter(Boolean)
      : text;
  }
  return out;
}

/** First payload value as a one-line preview (lists joined). */
export function previewOf(payload: Record<string, PayloadValue>, joiner: string): string {
  const first = Object.values(payload)[0];
  if (first === undefined) return '';
  return Array.isArray(first) ? first.join(joiner) : first;
}
