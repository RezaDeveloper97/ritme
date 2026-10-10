const rtf = new Intl.RelativeTimeFormat('fa', { numeric: 'auto' });

/** «۲ ساعت پیش» / «دیروز» — coarse relative time of an ISO instant. */
export function relativeTime(iso: string | null, now = Date.now()): string {
  if (!iso) return '';
  const diff = (new Date(iso).getTime() - now) / 1000;
  const abs = Math.abs(diff);
  if (abs < 3600) return rtf.format(Math.round(diff / 60), 'minute');
  if (abs < 86400) return rtf.format(Math.round(diff / 3600), 'hour');
  return rtf.format(Math.round(diff / 86400), 'day');
}
