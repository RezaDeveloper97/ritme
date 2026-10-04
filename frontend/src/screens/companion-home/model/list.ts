/**
 * «A, B and C» in the UI locale. Persian lists put «،» between items but never before «و» (stage B-4:
 * CLDR's `fa` conjunction gives «الف،‏ ب، و ج»), so `fa` is joined by hand; other locales use
 * `Intl.ListFormat`.
 */
export function joinList(items: readonly string[], locale: string): string {
  if (items.length === 0) return '';
  if (locale === 'fa') {
    if (items.length === 1) return items[0];
    return `${items.slice(0, -1).join('، ')} و ${items[items.length - 1]}`;
  }
  return new Intl.ListFormat(locale, { type: 'conjunction' }).format(items);
}
