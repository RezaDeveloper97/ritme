/** A `[id]` route segment → positive integer id, or null (the page then 404s). */
export function parseRouteId(raw: string): number | null {
  if (!/^\d{1,15}$/.test(raw)) return null;
  const id = Number(raw);
  return id >= 1 ? id : null;
}
