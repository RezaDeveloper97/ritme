/** A month chip: «۳ ماه» under two years, whole years after (24 → «۲ سال»), else months. */
export function chipUnit(months: number): { unit: 'months' | 'years'; n: number } {
  if (months >= 24 && months % 12 === 0) return { unit: 'years', n: months / 12 };
  return { unit: 'months', n: months };
}
