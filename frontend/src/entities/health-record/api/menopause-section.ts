import { z } from 'zod';

/*
 * The `menopause` provider section of the doctor report (CB-MENO-03, nbl_Meno_Report): the same object is the
 * `menopause` section of GET /health-record/report (bloom's builder → PDF + share link) and the `report` of
 * GET /menopause/report (the owner's 1 / 3 / 6 month preview). Health data (§11): never logged, never in a URL.
 */

const point = z.object({ month: z.string(), total: z.number() });

const treatmentItem = z
  .object({
    kind: z.string(),
    name: z.string(),
    dose: z.string().nullable(),
    started_on: z.string().nullable(),
    stopped_on: z.string().nullable(),
    days_taken: z.number(),
    days: z.number(),
    adherence_pct: z.number().nullable(),
  })
  .transform((i) => ({
    kind: i.kind,
    name: i.name,
    dose: i.dose,
    startedOn: i.started_on,
    stoppedOn: i.stopped_on,
    daysTaken: i.days_taken,
    days: i.days,
    adherencePct: i.adherence_pct,
  }));

export const menopauseSectionSchema = z
  .object({
    window: z.object({ from: z.string(), to: z.string(), days: z.number() }),
    stage: z.object({
      code: z.string().nullable(),
      months_without_period: z.number().nullable(),
      surgical: z.boolean().nullable(),
      hrt: z.boolean().nullable(),
    }),
    score: z.object({ first: point.nullable(), last: point.nullable(), max: z.number(), change: z.number() }).nullable(),
    hot_flashes: z.object({ total: z.number(), per_day: z.number().nullable(), tracked_days: z.number() }),
    night_sweats: z.object({ nights: z.number(), per_week: z.number().nullable() }),
    sleep: z.object({ avg_hours: z.number().nullable(), nights: z.number() }),
    bleeding: z.object({ events: z.number(), days: z.number(), dates: z.array(z.string()) }),
    blood_pressure: z.object({ systolic: z.number(), diastolic: z.number(), readings: z.number() }).nullable(),
    symptoms: z.object({
      tracked_days: z.number(),
      top: z.array(z.object({ key: z.string(), label: z.string(), days: z.number(), percent: z.number() })),
    }),
    treatment: z.object({
      hrt_adherence_pct: z.number().nullable(),
      items: z.array(treatmentItem),
      lifestyle: z.array(
        z.object({
          name: z.string(),
          weekly_goal: z.number().nullable(),
          goal_unit: z.string().nullable(),
          per_week: z.number().nullable(),
        }),
      ),
    }),
    side_effects: z.array(z.object({ code: z.string(), days: z.number(), first_on: z.string(), last_on: z.string() })),
    supplements: z.array(z.string()),
  })
  .transform((d) => ({
    window: d.window,
    stage: { code: d.stage.code, monthsWithoutPeriod: d.stage.months_without_period },
    score: d.score && d.score.first && d.score.last ? { first: d.score.first.total, last: d.score.last.total, max: d.score.max } : null,
    hotFlashes: { total: d.hot_flashes.total, perDay: d.hot_flashes.per_day, trackedDays: d.hot_flashes.tracked_days },
    nightSweats: { nights: d.night_sweats.nights, perWeek: d.night_sweats.per_week },
    sleepAvg: d.sleep.avg_hours,
    bleeding: { events: d.bleeding.events, dates: d.bleeding.dates },
    bloodPressure: d.blood_pressure,
    trackedDays: d.symptoms.tracked_days,
    topSymptoms: d.symptoms.top,
    hrtAdherencePct: d.treatment.hrt_adherence_pct,
    items: d.treatment.items,
    lifestyle: d.treatment.lifestyle.map((l) => ({
      name: l.name,
      weeklyGoal: l.weekly_goal,
      goalUnit: l.goal_unit,
      perWeek: l.per_week,
    })),
    sideEffects: d.side_effects.map((e) => ({ code: e.code, days: e.days, firstOn: e.first_on })),
    supplements: d.supplements,
  }));

export type MenopauseSection = z.output<typeof menopauseSectionSchema>;
