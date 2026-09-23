/**
 * Records whose translatable columns are listed by the API at run time
 * (pregnancy weeks: 10 modules, phase contents: 9 sections). Picks each listed
 * field as a `{code: text}` object ({} when empty or malformed).
 */
export function sectionsOf(
  row: Record<string, unknown> | null | undefined,
  fields: readonly string[],
): Record<string, Record<string, string>> {
  const out: Record<string, Record<string, string>> = {};
  for (const field of fields) {
    const value = row?.[field];
    const clean: Record<string, string> = {};
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      for (const [code, text] of Object.entries(value)) if (typeof text === 'string') clean[code] = text;
    }
    out[field] = clean;
  }
  return out;
}

/** How many of `fields` have text in at least one language. */
export function filledCount(sections: Record<string, Record<string, string>>): number {
  return Object.values(sections).filter((t) => Object.values(t).some((s) => s.trim() !== '')).length;
}
