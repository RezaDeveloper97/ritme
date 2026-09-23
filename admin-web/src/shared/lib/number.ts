/** Locale-aware integer formatting (Persian digits and separators in fa). */
export function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : locale).format(value);
}
