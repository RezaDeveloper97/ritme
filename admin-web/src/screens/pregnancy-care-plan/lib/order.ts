/** Move `from` to `to` (a new array; out-of-range moves are no-ops). */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  const next = [...items];
  if (from < 0 || from >= next.length || to < 0 || to >= next.length) return next;
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item as T);
  return next;
}

/** Drop empty translations; `null` when nothing is left (the API clears the column). */
export function cleanTranslations(value: Record<string, string>): Record<string, string> | null {
  const out: Record<string, string> = {};
  for (const [code, text] of Object.entries(value)) if (text.trim()) out[code] = text.trim();
  return Object.keys(out).length ? out : null;
}
