/** Tile classes per `checkup_types.tone` — admin tokens only (no rose/teal ramps here). */
const TONES: Record<string, string> = {
  rose: 'bg-[var(--danger-soft)] text-[var(--danger-deep)]',
  violet: 'bg-[var(--pink-bg)] text-[var(--brand)]',
  amber: 'bg-[var(--amber-soft)] text-[var(--amber-deep)]',
  teal: 'bg-[var(--data-soft)] text-[var(--data-deep)]',
  green: 'bg-[var(--green-soft)] text-[var(--green-deep)]',
  neutral: 'bg-[var(--surface-2)] text-[var(--ink-3)]',
};

export function toneClass(tone: string | null | undefined): string {
  return TONES[tone ?? 'neutral'] ?? TONES.neutral ?? '';
}
