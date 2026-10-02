import { z } from 'zod';

/**
 * Boundary parser of `GET /api/v1/analysis/pregnancy` (B-N3-12; shapes:
 * backend-go/internal/analysis/pregnancy.go, goldens under
 * testdata/golden/preg_*). snake_case → camelCase; an unknown enum value
 * degrades to `null` instead of failing the hub. Thresholds come from the
 * server — this bundle never repeats a number.
 */

const num = z.number();
const nullableNum = z.number().nullable().catch(null);
const softEnum = <const T extends [string, ...string[]]>(values: T) => z.enum(values).nullable().catch(null);

/** `{plus, locked, ready, data}` — a locked section carries no data. */
function section<T extends z.ZodTypeAny>(data: T) {
  return z.object({
    plus: z.boolean(),
    locked: z.boolean(),
    ready: z.boolean(),
    data: data.nullable(),
  });
}

const iomRow = z
  .object({ category: z.string(), bmi_min: nullableNum, bmi_max: nullableNum, gain_min: num, gain_max: num })
  .transform((r) => ({ category: r.category, bmiMin: r.bmi_min, bmiMax: r.bmi_max, gainMin: r.gain_min, gainMax: r.gain_max }));

const weightPoint = z
  .object({ date: z.string(), ga_days: num, week: num, weight: num, gain: num })
  .transform((p) => ({ date: p.date, gaDays: p.ga_days, week: p.week, weight: p.weight, gain: p.gain }));

const minMax = z.object({ min: num, max: num });

const weightGain = z
  .object({
    missing: softEnum(['baseline', 'height', 'weights']),
    baseline: z
      .object({ weight: num, source: softEnum(['before_pregnancy', 'first_trimester', 'profile']), date: z.string().nullable() })
      .nullable(),
    height_cm: nullableNum,
    bmi: nullableNum,
    bmi_category: z.string().nullable(),
    target: minMax.nullable(),
    status: softEnum(['below', 'within', 'above']),
    current: z
      .object({
        date: z.string(),
        ga_days: num,
        week: num,
        weight: num,
        gain: num,
        recommended: minMax.nullable(),
      })
      .nullable(),
    recent_4w: nullableNum,
    points: z.array(weightPoint).catch([]),
    band: z
      .array(z.object({ ga_weeks: num, min: num, max: num }))
      .nullable()
      .transform((b) => (b ?? []).map((x) => ({ gaWeeks: x.ga_weeks, min: x.min, max: x.max }))),
    iom_table: z.array(iomRow).catch([]),
  })
  .transform((w) => ({
    missing: w.missing,
    baseline: w.baseline,
    heightCm: w.height_cm,
    bmi: w.bmi,
    bmiCategory: w.bmi_category,
    target: w.target,
    status: w.status,
    current: w.current && {
      date: w.current.date,
      gaDays: w.current.ga_days,
      week: w.current.week,
      weight: w.current.weight,
      gain: w.current.gain,
      recommended: w.current.recommended,
    },
    recent4w: w.recent_4w,
    points: w.points,
    band: w.band,
    iomTable: w.iom_table,
  }));

const bpPair = z.object({ systolic: num, diastolic: num });

const bloodPressure = z
  .object({
    readings_count: num,
    high_count: num,
    severe_count: num,
    status: softEnum(['below_threshold', 'high', 'severe']),
    latest: bpPair.extend({ date: z.string() }).nullable(),
    readings: z.array(bpPair.extend({ date: z.string(), high: z.boolean() })).catch([]),
    threshold: bpPair,
    severe_threshold: bpPair,
  })
  .transform((b) => ({
    readingsCount: b.readings_count,
    highCount: b.high_count,
    severeCount: b.severe_count,
    status: b.status,
    latest: b.latest,
    readings: b.readings,
    threshold: b.threshold,
    severeThreshold: b.severe_threshold,
  }));

export const GLUCOSE_SLOTS = ['fasting', 'one_hour', 'two_hour'] as const;
export type GlucoseSlotKey = (typeof GLUCOSE_SLOTS)[number];

const glucose = z
  .object({
    slots: z
      .array(
        z.object({
          slot: z.string(),
          target_max: num,
          avg: nullableNum,
          within_target: z.boolean().nullable().catch(null),
          readings: num,
          in_target: num,
        }),
      )
      .catch([]),
  })
  .transform((g) => ({
    slots: g.slots
      .filter((s): s is typeof s & { slot: GlucoseSlotKey } => (GLUCOSE_SLOTS as readonly string[]).includes(s.slot))
      .map((s) => ({
        slot: s.slot,
        targetMax: s.target_max,
        avg: s.avg,
        withinTarget: s.within_target,
        readings: s.readings,
        inTarget: s.in_target,
      })),
  }));

const kicks = z
  .object({
    days: z.array(z.object({ date: z.string(), count: nullableNum, minutes_to_target: nullableNum })).catch([]),
    sessions: num,
    timed_sessions: num,
    low_count_days: num.catch(0),
    all_within_window: z.boolean().nullable().catch(null),
    target: num,
    window_minutes: num,
    from_week: num,
    current_week: num,
  })
  .transform((k) => ({
    days: k.days.map((d) => ({ date: d.date, count: d.count, minutes: d.minutes_to_target })),
    sessions: k.sessions,
    timedSessions: k.timed_sessions,
    lowCountDays: k.low_count_days,
    allWithinWindow: k.all_within_window,
    target: k.target,
    windowMinutes: k.window_minutes,
    fromWeek: k.from_week,
    currentWeek: k.current_week,
  }));

const symptoms = z
  .object({
    trimesters: z
      .array(
        z.object({
          trimester: num,
          is_current: z.boolean(),
          is_future: z.boolean(),
          logged_days: num,
          items: z.array(z.object({ key: z.string(), label: z.string(), days: num })).catch([]),
        }),
      )
      .catch([]),
  })
  .transform((s) => ({
    trimesters: s.trimesters.map((t) => ({
      trimester: t.trimester,
      isCurrent: t.is_current,
      isFuture: t.is_future,
      loggedDays: t.logged_days,
      items: t.items,
    })),
  }));

const visits = z
  .object({
    done: num,
    upcoming: num,
    next: z.object({ title: z.string(), date: z.string(), week: num, days_until: num }).nullable(),
    latest_result: z.object({ title: z.string(), date: z.string(), note: z.string() }).nullable(),
  })
  .transform((v) => ({
    done: v.done,
    upcoming: v.upcoming,
    next: v.next && { title: v.next.title, date: v.next.date, week: v.next.week, daysUntil: v.next.days_until },
    latestResult: v.latest_result,
  }));

export const pregnancyAnalysisSchema = z
  .object({
    pregnancy: z.object({
      week: num,
      weeks: num,
      days: num,
      trimester: num,
      due_date: z.string(),
      start_date: z.string(),
    }),
    sections: z.object({
      weight_gain: section(weightGain),
      blood_pressure: section(bloodPressure),
      glucose: section(glucose),
      kicks: section(kicks),
      symptoms: section(symptoms),
      visits: section(visits),
    }),
  })
  .transform((r) => ({
    pregnancy: {
      week: r.pregnancy.week,
      weeks: r.pregnancy.weeks,
      days: r.pregnancy.days,
      trimester: r.pregnancy.trimester,
      dueDate: r.pregnancy.due_date,
      startDate: r.pregnancy.start_date,
    },
    sections: {
      weightGain: r.sections.weight_gain,
      bloodPressure: r.sections.blood_pressure,
      glucose: r.sections.glucose,
      kicks: r.sections.kicks,
      symptoms: r.sections.symptoms,
      visits: r.sections.visits,
    },
  }));

export type PregnancyAnalysis = z.output<typeof pregnancyAnalysisSchema>;
export type PregnancySections = PregnancyAnalysis['sections'];
export type WeightGain = NonNullable<PregnancySections['weightGain']['data']>;
export type BloodPressure = NonNullable<PregnancySections['bloodPressure']['data']>;
export type GlucoseTargets = NonNullable<PregnancySections['glucose']['data']>;
export type Kicks = NonNullable<PregnancySections['kicks']['data']>;
export type TrimesterSymptoms = NonNullable<PregnancySections['symptoms']['data']>;
export type Visits = NonNullable<PregnancySections['visits']['data']>;
