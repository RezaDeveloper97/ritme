import { z } from 'zod';

import { roundBbt } from '../model/bbt';
import {
  BBT_PHASES,
  BBT_RANGES,
  type BbtCycle,
  type BbtRange,
  type BbtStats,
  CERVICAL_MUCUS,
  CHANCE_LEVELS,
  CHANCE_MAX_BARS,
  EVIDENCE_STRENGTHS,
  FERTILITY_SYMPTOMS,
  type FertilityBbt,
  type FertilityChance,
  type FertilityDay,
  type FertilityInsights,
  type FertilitySymptom,
  type FertilityTip,
  type FertilityToday,
  INSIGHT_CONFIDENCE,
  INTERCOURSE_TYPES,
  type InsightEvidence,
  LH_RESULTS,
  type LabeledValue,
  type OvulationHistoryRow,
} from '../model/types';

/**
 * Boundary parsers for `/api/v1/fertility/*` (CLAUDE.md §10 — zod at the
 * boundary). The frontend is written before the Go endpoints (T-M5-01…03), so
 * the parsers tolerate the shapes the README leaves open:
 *
 * - an enum value this bundle doesn't know parses to `null` («ثبت نشه»), an
 *   unknown symptom is dropped — a newer backend never crashes an older bundle;
 * - an enum may arrive bare (`"positive"`) or as `{value, label}`;
 * - numbers may arrive as numbers or numeric strings (`decimal(4,2)` → `"36.42"`);
 * - `GET /fertility/bbt` may answer one cycle at the top level or `{cycles: […]}`.
 */

/** Unknown / missing enum → null. */
const enumOrNull = <T extends readonly [string, ...string[]]>(values: T) =>
  z.enum(values).nullable().catch(null);

/** A number (or numeric string) → number; anything else → null. */
const numberOrNull = z.unknown().transform((v): number | null => {
  if (v === null || v === undefined || v === '') return null;
  const n = typeof v === 'number' ? v : typeof v === 'string' ? Number(v) : Number.NaN;
  return Number.isFinite(n) ? n : null;
});

const intOrNull = numberOrNull.transform((n) => (n === null ? null : Math.round(n)));

const countSchema = intOrNull.transform((n) => (n === null || n < 0 ? 0 : n));

const bbtValue = numberOrNull.transform((n) => (n === null ? null : roundBbt(n)));

const textOrNull = z
  .unknown()
  .transform((v) => (typeof v === 'string' && v.trim() !== '' ? v : null));

const dateSchema = z.string().transform((v) => v.slice(0, 10));

/** "07:30" or "07:30:00" → "07:30". */
const timeOrNull = textOrNull.transform((v) => (v ? v.slice(0, 5) : null));

/** Pull `value` out of `{value, label}`; pass a bare value through. */
function unwrapValue(raw: unknown): unknown {
  if (typeof raw === 'object' && raw !== null && 'value' in raw) {
    return (raw as { value: unknown }).value;
  }
  return raw;
}

function labeled<T extends readonly [string, ...string[]]>(values: T) {
  const valueSchema = enumOrNull(values);
  return z.unknown().transform((raw): LabeledValue<T[number]> => {
    const label =
      typeof raw === 'object' && raw !== null && 'label' in raw
        ? textOrNull.parse((raw as { label: unknown }).label)
        : null;
    return { value: valueSchema.parse(unwrapValue(raw) ?? null), label };
  });
}

const bareEnum = <T extends readonly [string, ...string[]]>(values: T) =>
  z.unknown().transform((raw): T[number] | null => enumOrNull(values).parse(unwrapValue(raw) ?? null));

const symptomsSchema = z
  .unknown()
  .transform((raw): FertilitySymptom[] => {
    if (!Array.isArray(raw)) return [];
    const got = new Set(raw.map(unwrapValue));
    return FERTILITY_SYMPTOMS.filter((s) => got.has(s));
  });

const chanceSchema = z
  .object({
    level: z.unknown().optional(),
    label: z.unknown().optional(),
    bars: z.unknown().optional(),
  })
  .transform(
    (r): FertilityChance => ({
      level: bareEnum(CHANCE_LEVELS).parse(r.level),
      label: textOrNull.parse(r.label),
      bars: Math.min(CHANCE_MAX_BARS, countSchema.parse(r.bars)),
    }),
  );

const chanceOrNull = z
  .unknown()
  .transform((raw): FertilityChance | null => {
    const parsed = chanceSchema.safeParse(raw);
    return parsed.success ? parsed.data : null;
  });

const EMPTY_CHANCE: FertilityChance = { level: null, label: null, bars: 0 };

/** `GET /fertility/today`. */
export const fertilityTodaySchema = z
  .object({
    date: dateSchema,
    cycle_day: intOrNull.optional(),
    chance: chanceOrNull.optional(),
    lh: labeled(LH_RESULTS).optional(),
    bbt: z.unknown().optional(),
    intercourse: labeled(INTERCOURSE_TYPES).optional(),
  })
  .transform(
    (r): FertilityToday => ({
      date: r.date,
      cycleDay: r.cycle_day ?? null,
      chance: r.chance ?? EMPTY_CHANCE,
      lh: r.lh ?? { value: null, label: null },
      bbt: bbtValue.parse(unwrapValue(r.bbt) ?? null),
      intercourse: r.intercourse ?? { value: null, label: null },
    }),
  );

/** `GET|PUT /fertility/days/{date}` — the merged day. */
export const fertilityDaySchema = z
  .object({
    date: dateSchema,
    cycle_day: intOrNull.optional(),
    lh: bareEnum(LH_RESULTS).optional(),
    mucus: bareEnum(CERVICAL_MUCUS).optional(),
    bbt: z.unknown().optional(),
    bbt_time: timeOrNull.optional(),
    intercourse: bareEnum(INTERCOURSE_TYPES).optional(),
    symptoms: symptomsSchema.optional(),
    note: textOrNull.optional(),
    chance: chanceOrNull.optional(),
  })
  .transform(
    (r): FertilityDay => ({
      date: r.date,
      cycleDay: r.cycle_day ?? null,
      lh: r.lh ?? null,
      mucus: r.mucus ?? null,
      bbt: bbtValue.parse(unwrapValue(r.bbt) ?? null),
      bbtTime: r.bbt_time ?? null,
      intercourse: r.intercourse ?? null,
      symptoms: r.symptoms ?? [],
      note: r.note ?? null,
      chance: r.chance ?? null,
    }),
  );

// ── BBT ────────────────────────────────────────────────────────

const bbtPointSchema = z.object({
  cycle_day: intOrNull,
  date: dateSchema,
  value: bbtValue,
});

/** Drops points without a day or a reading; sorted by cycle day. */
const bbtPointsSchema = z
  .unknown()
  .transform((raw) => {
    if (!Array.isArray(raw)) return [];
    const points = [];
    for (const item of raw) {
      const parsed = bbtPointSchema.safeParse(item);
      if (!parsed.success) continue;
      const { cycle_day: cycleDay, date, value } = parsed.data;
      if (cycleDay === null || value === null) continue;
      points.push({ cycleDay, date, value });
    }
    return points.sort((a, b) => a.cycleDay - b.cycleDay);
  });

const windowDaysSchema = z
  .unknown()
  .transform((raw): BbtCycle['fertileWindow'] => {
    if (typeof raw !== 'object' || raw === null) return null;
    const r = raw as Record<string, unknown>;
    const fromDay = intOrNull.parse(r.from_day ?? null);
    const toDay = intOrNull.parse(r.to_day ?? null);
    return fromDay !== null && toDay !== null && fromDay <= toDay ? { fromDay, toDay } : null;
  });

const bbtCycleSchema = z
  .object({
    start_date: textOrNull.optional(),
    points: bbtPointsSchema.optional(),
    coverline: bbtValue.optional(),
    fertile_window: windowDaysSchema.optional(),
    shift_day: intOrNull.optional(),
    phase: bareEnum(BBT_PHASES).optional(),
  })
  .transform(
    (r): BbtCycle => ({
      startDate: r.start_date ? r.start_date.slice(0, 10) : null,
      points: r.points ?? [],
      coverline: r.coverline ?? null,
      fertileWindow: r.fertile_window ?? null,
      shiftDay: r.shift_day ?? null,
      phase: r.phase ?? null,
    }),
  );

const bbtStatsSchema = z
  .object({
    pre_ovulation_avg: bbtValue.optional(),
    logged_days: countSchema.optional(),
    cycle_days_so_far: countSchema.optional(),
    gaps: countSchema.optional(),
  })
  .transform(
    (r): BbtStats => ({
      preOvulationAvg: r.pre_ovulation_avg ?? null,
      loggedDays: r.logged_days ?? 0,
      cycleDaysSoFar: r.cycle_days_so_far ?? 0,
      gaps: r.gaps ?? 0,
    }),
  );

const EMPTY_STATS: BbtStats = { preOvulationAvg: null, loggedDays: 0, cycleDaysSoFar: 0, gaps: 0 };

/** `"…"` or `{title, body}` → tip; anything else → null. */
const tipSchema = z
  .unknown()
  .transform((raw): FertilityTip | null => {
    if (typeof raw === 'string') return raw.trim() ? { title: null, body: raw } : null;
    if (typeof raw !== 'object' || raw === null) return null;
    const r = raw as Record<string, unknown>;
    const body = textOrNull.parse(r.body ?? r.text ?? null);
    const title = textOrNull.parse(r.title ?? null);
    if (!body && !title) return null;
    return { title: body ? title : null, body: body ?? (title as string) };
  });

const intList = z
  .unknown()
  .transform((raw): number[] =>
    Array.isArray(raw)
      ? raw.map((v) => intOrNull.parse(v)).filter((n): n is number => n !== null)
      : [],
  );

const rangeSchema = z
  .unknown()
  .transform((raw): BbtRange => {
    const n = intOrNull.parse(raw ?? null);
    return (BBT_RANGES as readonly number[]).includes(n ?? -1) ? (n as BbtRange) : 1;
  });

function asRecord(raw: unknown): Record<string, unknown> {
  return typeof raw === 'object' && raw !== null ? (raw as Record<string, unknown>) : {};
}

/**
 * `GET /fertility/bbt?range=`. The README says "per cycle" without fixing the
 * envelope, so both `{cycles: [current, …], stats, …}` and a single cycle at
 * the top level are read; `stats` / `past_shift_days` / `tip` are taken from the
 * top level first, then from the current cycle.
 */
export const fertilityBbtSchema = z
  .unknown()
  .transform((raw): FertilityBbt => {
    const top = asRecord(raw);
    const rawCycles: unknown[] = Array.isArray(top.cycles) ? top.cycles : 'points' in top ? [top] : [];
    const cycles = rawCycles
      .map((c) => bbtCycleSchema.safeParse(asRecord(c)))
      .filter((p) => p.success)
      .map((p) => p.data as BbtCycle);
    const current = asRecord(rawCycles[0]);
    const pick = (key: string): unknown => top[key] ?? current[key] ?? null;
    const stats = bbtStatsSchema.safeParse(asRecord(pick('stats')));
    return {
      range: rangeSchema.parse(top.range),
      cycles,
      stats: stats.success ? stats.data : EMPTY_STATS,
      pastShiftDays: intList.parse(pick('past_shift_days')),
      tip: tipSchema.parse(pick('tip')),
    };
  });

// ── Insights ───────────────────────────────────────────────────

const evidenceSchema = z
  .object({
    key: z.unknown().optional(),
    title: z.string(),
    detail: textOrNull.optional(),
    strength: bareEnum(EVIDENCE_STRENGTHS).optional(),
  })
  .transform(
    (r): InsightEvidence => ({
      key: typeof r.key === 'string' ? r.key : '',
      title: r.title,
      detail: r.detail ?? null,
      strength: r.strength ?? null,
    }),
  );

const historyRowSchema = z
  .object({ month_label: z.string(), ovulation_day: intOrNull.optional(), cycle_start: textOrNull.optional() })
  .transform(
    (r): OvulationHistoryRow => ({
      monthLabel: r.month_label,
      ovulationDay: r.ovulation_day ?? null,
      cycleStart: r.cycle_start ? r.cycle_start.slice(0, 10) : null,
    }),
  );

/** Parse each row on its own and drop the ones that fail. */
const listOf = <T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) =>
  z.unknown().transform((raw): T[] => {
    if (!Array.isArray(raw)) return [];
    const out: T[] = [];
    for (const item of raw) {
      const parsed = schema.safeParse(item);
      if (parsed.success) out.push(parsed.data);
    }
    return out;
  });

const insightWindowSchema = z
  .unknown()
  .transform((raw): FertilityInsights['window'] => {
    const r = asRecord(raw);
    const start = textOrNull.parse(r.start ?? null);
    const end = textOrNull.parse(r.end ?? null);
    if (!start || !end) return null;
    const ovulation = textOrNull.parse(r.ovulation ?? null);
    return { start: start.slice(0, 10), end: end.slice(0, 10), ovulation: ovulation ? ovulation.slice(0, 10) : null };
  });

const tipTextList = z
  .unknown()
  .transform((raw): string[] =>
    Array.isArray(raw)
      ? raw
          .map((t) => tipSchema.parse(t))
          .filter((t): t is FertilityTip => t !== null)
          .map((t) => t.body)
      : [],
  );

/** `GET /fertility/insights`. */
export const fertilityInsightsSchema = z
  .object({
    cycles_used: countSchema.optional(),
    window: insightWindowSchema.optional(),
    confidence: bareEnum(INSIGHT_CONFIDENCE).optional(),
    evidence: listOf(evidenceSchema).optional(),
    history: listOf(historyRowSchema).optional(),
    tips: tipTextList.optional(),
  })
  .transform(
    (r): FertilityInsights => ({
      cyclesUsed: r.cycles_used ?? 0,
      window: r.window ?? null,
      confidence: r.confidence ?? null,
      evidence: r.evidence ?? [],
      history: r.history ?? [],
      tips: r.tips ?? [],
    }),
  );
