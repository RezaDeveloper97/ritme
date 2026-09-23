/**
 * Multipart bodies in the PHP bracket syntax the admin API parses (admin-api.md §7):
 *   { title: {fa: 'x'} }        → title[fa]=x
 *   { cycle_phases: ['a','b'] } → cycle_phases[]=a, cycle_phases[]=b
 *   { is_active: true }         → is_active=1   (false → 0)
 *   { category: null }          → category=     (empty = cleared, like an empty form input)
 *   { image: File }             → the file part
 * `undefined` leaves the key out (absent = unchanged on update).
 */
export type FormValue =
  | string
  | number
  | boolean
  | null
  | undefined
  | Blob
  | readonly FormValue[]
  | { readonly [key: string]: FormValue };

export function toFormData(values: Record<string, FormValue>): FormData {
  const fd = new FormData();
  const append = (name: string, value: FormValue): void => {
    if (value === undefined) return;
    if (value === null) fd.append(name, '');
    else if (typeof Blob !== 'undefined' && value instanceof Blob) fd.append(name, value);
    else if (typeof value === 'boolean') fd.append(name, value ? '1' : '0');
    else if (typeof value === 'string' || typeof value === 'number') fd.append(name, String(value));
    else if (Array.isArray(value)) for (const item of value) append(`${name}[]`, item);
    else for (const [key, item] of Object.entries(value)) append(`${name}[${key}]`, item);
  };
  for (const [key, value] of Object.entries(values)) append(key, value);
  return fd;
}

/** `''` → null, anything else trimmed: optional text inputs. */
export function blankToNull(value: string): string | null {
  const trimmed = value.trim();
  return trimmed === '' ? null : trimmed;
}

/** Number input text → integer, or null when empty / not a number. */
export function toIntOrNull(value: string): number | null {
  if (value.trim() === '') return null;
  const n = Number(value);
  return Number.isFinite(n) ? Math.trunc(n) : null;
}

/**
 * `<input type="datetime-local">` ↔ the API. The API sends Tehran ISO
 * (`2026-09-23T13:00:00+03:30`); the input shows its wall-clock part. Sending
 * back `2026-09-23 13:00` is read in the app time zone (Asia/Tehran), as the
 * Blade form did.
 */
export function isoToLocalInput(value: string | null | undefined): string {
  if (!value) return '';
  const match = /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})/.exec(value);
  return match ? `${match[1]}T${match[2]}` : '';
}

export function localInputToApi(value: string): string | null {
  if (!value) return null;
  return value.replace('T', ' ');
}
