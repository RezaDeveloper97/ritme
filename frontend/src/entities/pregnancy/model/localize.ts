import type { Bilingual, PregnancyFlags } from './types';

/**
 * The backend hands back some strings pre-localized both ways (`{ en, fa }`) —
 * status flags and formatted dates. The UI resolves them here with a graceful
 * fallback (requested locale → fa → en → empty).
 */
export function pickBilingual(value: Bilingual | string | null | undefined, locale: string): string {
  if (value == null) return '';
  if (typeof value === 'string') return value;
  const key = locale === 'en' ? 'en' : 'fa';
  return value[key as keyof Bilingual] || value.fa || value.en || '';
}

/** Flatten the status `flags` map into ordered, locale-resolved message lines. */
export function resolveFlags(
  flags: PregnancyFlags,
  locale: string,
): { key: string; text: string }[] {
  return Object.entries(flags)
    .map(([key, value]) => ({ key, text: pickBilingual(value, locale) }))
    .filter((f) => f.text.length > 0);
}
