import { MAX_ALLERGIES, MAX_ALLERGY_LENGTH } from './types';

/** Adds a trimmed allergy unless it is blank, too long, a case-insensitive duplicate or the list is full. */
export function addAllergy(list: readonly string[], raw: string): string[] {
  const value = raw.trim().replace(/\s+/g, ' ');
  if (!value || value.length > MAX_ALLERGY_LENGTH || list.length >= MAX_ALLERGIES) return [...list];
  if (list.some((a) => a.toLocaleLowerCase() === value.toLocaleLowerCase())) return [...list];
  return [...list, value];
}

export function removeAllergy(list: readonly string[], value: string): string[] {
  return list.filter((a) => a !== value);
}

/** Toggles code in list, keeping the canonical order of `all`. */
export function toggleCode(list: readonly string[], code: string, all: readonly string[]): string[] {
  const next = new Set(list);
  if (next.has(code)) next.delete(code);
  else next.add(code);
  return all.filter((c) => next.has(c));
}
