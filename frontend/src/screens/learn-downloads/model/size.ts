/** Bytes → a unit key + rounded value for «۲۱۴ مگابایت» / «۳٫۲ گیگ» (B-N8-04). */
export function sizeParts(bytes: number): { unit: 'kb' | 'mb' | 'gb'; n: number } {
  if (bytes <= 0) return { unit: 'mb', n: 0 };
  const kb = bytes / 1024;
  if (kb < 1024) return { unit: 'kb', n: Math.max(1, Math.round(kb)) };
  const mb = kb / 1024;
  if (mb < 1024) return { unit: 'mb', n: mb < 10 ? Math.round(mb * 10) / 10 : Math.round(mb) };
  return { unit: 'gb', n: Math.round((mb / 1024) * 10) / 10 };
}

/** Share of the quota in use, 0–100, at least 1 when anything is stored. */
export function meterPercent(used: number, quota: number | null): number {
  if (!quota || quota <= 0) return 0;
  const p = (used / quota) * 100;
  return used > 0 ? Math.min(100, Math.max(1, Math.round(p))) : 0;
}
