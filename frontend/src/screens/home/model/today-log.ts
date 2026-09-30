import { LOG_CATEGORIES } from '@/entities/health-log';

/**
 * What the «ثبت امروز» card says about today's saved daily log (B-N1-06):
 * the bleeding level (or spotting), the symptom fields that are set, and the
 * moods. Keys only — the card labels them through the `log` messages.
 */
export interface TodayLogSummary {
  /** `bleeding_intensity` value, `'spotting'`, or null when no bleeding was logged. */
  bleeding: string | null;
  /** Field keys of the logged symptoms (pain + appetite/energy categories), in form order. */
  symptoms: string[];
  moods: string[];
}

const SYMPTOM_CATEGORIES = new Set(['pain', 'digestion']);
const SYMPTOM_FIELDS = LOG_CATEGORIES.filter((c) => SYMPTOM_CATEGORIES.has(c.key)).flatMap((c) =>
  c.fields.map((f) => f.key as string),
);

function isSet(value: unknown): boolean {
  if (value === null || value === undefined || value === false || value === '') return false;
  if (Array.isArray(value)) return value.length > 0;
  return true;
}

export function summarizeTodayLog(log: object | null | undefined): TodayLogSummary {
  const row = (log ?? {}) as Record<string, unknown>;
  const intensity = row.bleeding_intensity;
  const bleeding =
    typeof intensity === 'string' && intensity !== '' ? intensity : row.spotting === true ? 'spotting' : null;
  const moods = Array.isArray(row.moods) ? row.moods.filter((m): m is string => typeof m === 'string') : [];
  return { bleeding, symptoms: SYMPTOM_FIELDS.filter((k) => isSet(row[k])), moods };
}
