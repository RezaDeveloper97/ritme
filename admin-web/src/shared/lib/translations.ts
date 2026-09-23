/**
 * The text to show for a translatable value in a list or heading: the first
 * non-empty of the preferred codes (UI locale, then the default content
 * language), else any non-empty translation, else ''. Mirrors the backend's
 * `localized()` fallback (frontend/CLAUDE.md §6.3).
 */
export function pickTranslation(
  value: Record<string, string> | null | undefined,
  preferred: ReadonlyArray<string | null | undefined>,
): string {
  if (!value) return '';
  for (const code of preferred) {
    const text = code ? value[code]?.trim() : '';
    if (text) return text;
  }
  for (const text of Object.values(value)) {
    if (text?.trim()) return text.trim();
  }
  return '';
}

/** Plain-text preview of (possibly HTML) content, cut at `max` characters. */
export function excerpt(text: string, max = 90): string {
  const plain = text
    .replace(/<[^>]*>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return plain.length > max ? `${plain.slice(0, max).trimEnd()}…` : plain;
}
