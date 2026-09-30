import { z } from 'zod';

/** Response of `GET /cycle/history` (backend-go/api/openapi.yaml → getCycleHistory). */
const cycleRowSchema = z
  .object({
    start: z.string(),
    end: z.string().nullable(),
    period_end: z.string().nullable(),
    period_days: z.number().int(),
    period_ongoing: z.boolean(),
    length: z.number().int(),
    is_current: z.boolean(),
    in_range: z.boolean().nullable(),
  })
  .transform((r) => ({
    start: r.start,
    end: r.end,
    periodDays: r.period_days,
    periodOngoing: r.period_ongoing,
    length: r.length,
    isCurrent: r.is_current,
    inRange: r.in_range,
  }));

export const REGULARITY = ['regular', 'irregular', 'not_enough_data'] as const;
export type Regularity = (typeof REGULARITY)[number];

export const cycleHistorySchema = z
  .object({
    date: z.string(),
    based_on_cycles: z.number().int(),
    cycle_length: z.object({
      median: z.number().int().nullable(),
      spread_days: z.number().int().nullable(),
      range: z.object({ min: z.number().int(), max: z.number().int() }).nullable(),
      predicted: z.number().int(),
    }),
    period_length: z.object({ median: z.number().int().nullable() }),
    regularity: z.object({
      // An unknown future status degrades to "not enough data" rather than failing the screen.
      status: z.string().transform((s): Regularity => ((REGULARITY as readonly string[]).includes(s) ? (s as Regularity) : 'not_enough_data')),
      in_range: z.number().int(),
      total: z.number().int(),
    }),
    cycles: z.array(cycleRowSchema),
  })
  .transform((h) => ({
    date: h.date,
    basedOn: h.based_on_cycles,
    medianCycle: h.cycle_length.median,
    spread: h.cycle_length.spread_days,
    range: h.cycle_length.range,
    predicted: h.cycle_length.predicted,
    medianPeriod: h.period_length.median,
    regularity: h.regularity.status,
    inRange: h.regularity.in_range,
    total: h.regularity.total,
    cycles: h.cycles,
  }));

export type CycleHistory = z.infer<typeof cycleHistorySchema>;
export type CycleHistoryRow = CycleHistory['cycles'][number];

/** Response of `GET /cycle/symptom-pattern` (getCycleSymptomPattern). */
export const PATTERN_GROUPS = ['symptoms', 'mood', 'pain'] as const;
export type PatternGroup = (typeof PATTERN_GROUPS)[number];

const windowSchema = z
  .object({
    start_day: z.number().int(),
    end_day: z.number().int(),
    relation: z.enum(['before_period', 'early', 'mid']),
    days: z.number().int(),
  })
  .transform((w) => ({ startDay: w.start_day, endDay: w.end_day, relation: w.relation, days: w.days }));

const symptomSchema = z.object({
  key: z.string(),
  cycles: z.number().int(),
  strip: z.array(z.number()),
  window: windowSchema.nullable(),
});

export const symptomPatternSchema = z
  .object({
    ready: z.boolean(),
    cycles_counted: z.number().int(),
    cycles_needed: z.number().int(),
    typical: z.object({
      cycle_length: z.number().int(),
      period_length: z.number().int(),
      ovulation_day: z.number().int(),
    }),
    groups: z.array(z.object({ key: z.string(), items: z.array(symptomSchema) })),
  })
  .transform((p) => {
    const groups: Record<PatternGroup, z.infer<typeof symptomSchema>[]> = { symptoms: [], mood: [], pain: [] };
    for (const g of p.groups) {
      if ((PATTERN_GROUPS as readonly string[]).includes(g.key)) groups[g.key as PatternGroup] = g.items;
    }
    return {
      ready: p.ready,
      cyclesCounted: p.cycles_counted,
      cyclesNeeded: p.cycles_needed,
      cycleLength: p.typical.cycle_length,
      periodLength: p.typical.period_length,
      ovulationDay: p.typical.ovulation_day,
      groups,
    };
  });

export type SymptomPattern = z.infer<typeof symptomPatternSchema>;
export type SymptomPatternItem = SymptomPattern['groups'][PatternGroup][number];
export type SymptomWindow = NonNullable<SymptomPatternItem['window']>;
