/** Locale-aware integer formatting (Persian digits and separators in fa). */
export function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : locale).format(value);
}

// Persian (U+06F0…) and Arabic-Indic (U+0660…) digits, both typed by fa/ar keyboards.
const NON_LATIN_DIGIT = /[۰-۹٠-٩]/g;

/** Any digit script → ASCII digits; everything else untouched. */
export function toLatinDigits(text: string): string {
  return text.replace(NON_LATIN_DIGIT, (d) => String(d.charCodeAt(0) & 0xf));
}

/**
 * What a number field stores for typed text: ASCII digits, one leading minus and one decimal
 * point (the Persian «٫» counts as one); anything else is dropped. So `"۱۲"`, `"12"` and
 * `"١٢"` all store `"12"`, and the value reaches the API as it did with `type="number"`.
 */
export function toNumberText(text: string): string {
  const ascii = toLatinDigits(text).replace(/[٫،]/g, '.').replace(/−/g, '-');
  const negative = ascii.trimStart().startsWith('-');
  const [whole = '', ...rest] = ascii.replace(/[^0-9.]/g, '').split('.');
  const fraction = rest.length ? `.${rest.join('')}` : '';
  return `${negative ? '-' : ''}${whole}${fraction}`;
}

/** Stored number text → the locale's digits for display (`"12"` → `"۱۲"` in fa). */
export function toLocaleDigits(text: string, locale: string): string {
  return text.replace(/[0-9]/g, (d) => formatNumber(Number(d), locale));
}
