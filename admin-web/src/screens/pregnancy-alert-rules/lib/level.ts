import { formatNumber } from '@/shared/lib';

/**
 * Alert card colours per level — the app's v2 alert tones (info = data teal,
 * suggestion = brand, follow_up = amber, urgent = danger), admin tokens only.
 */
const LEVELS: Record<string, { card: string; chip: string }> = {
  info: {
    card: 'border-[var(--data-deep)] bg-[var(--data-soft)]',
    chip: 'bg-[var(--data-soft)] text-[var(--data-deep)]',
  },
  suggestion: {
    card: 'border-[var(--brand)] bg-[var(--pink-bg)]',
    chip: 'bg-[var(--pink-bg)] text-[var(--brand)]',
  },
  follow_up: {
    card: 'border-[var(--amber-deep)] bg-[var(--amber-soft)]',
    chip: 'bg-[var(--amber-soft)] text-[var(--amber-deep)]',
  },
  urgent: {
    card: 'border-[var(--danger-deep)] bg-[var(--danger-soft)]',
    chip: 'bg-[var(--danger-soft)] text-[var(--danger-deep)]',
  },
};

export function levelClass(level: string | null | undefined, part: 'card' | 'chip'): string {
  return (LEVELS[level ?? 'info'] ?? LEVELS.info)?.[part] ?? '';
}

/** Sample values for the preview of a rule's texts (`{days}` → 3, …); numbers take the text's digits. */
const SAMPLES: Record<string, string | number> = {
  days: 3,
  severe_count: 2,
  count: 4,
  symptom: '—',
  week: 12,
  basis: '—',
  systolic: 145,
  diastolic: 95,
  fasting: 105,
  post_meal: 160,
  status: '—',
};

/** Fills the placeholders of a text written in `locale` (QA 2026-09-29-c L3: fa text → fa digits). */
export function fillSample(text: string, locale: string, samples: Record<string, string> = {}): string {
  return text.replace(/\{([a-z_]+)\}/g, (all, name: string) => {
    const v = samples[name] ?? SAMPLES[name];
    if (v === undefined) return all;
    return typeof v === 'number' ? formatNumber(v, locale) : v;
  });
}
