/** The learn subtitle age: «۳ ماهگی» under two years, whole years after. */
export function learnAge(months: number): { unit: 'months' | 'years'; n: number } {
  if (months >= 24) return { unit: 'years', n: Math.floor(months / 12) };
  return { unit: 'months', n: Math.max(0, months) };
}
