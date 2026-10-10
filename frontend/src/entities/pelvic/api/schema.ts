import { z } from 'zod';

import {
  PELVIC_LEAKS,
  PELVIC_UTI_SYMPTOMS,
  type PelvicAlerts,
  type PelvicDiary,
  type PelvicLevel,
  type PelvicOverview,
  type PelvicProgram,
  type PelvicUtiSymptom,
  type PelvicWeekDay,
} from '../model/types';

/*
 * Zod parsers of `GET /pelvic` (+ program / session writes), `PUT /pelvic/diary/{date}` and `GET /catalog/pelvic_alerts`
 * (backend-go/contract/golden/pelvic). Lenient: a malformed level or day is dropped, never the screen.
 */

const ymd = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);
const text = z.string().trim().min(1).nullable().catch(null);
const positive = z.number().int().positive();

const levelSchema = z
  .object({
    code: z.string(),
    number: positive,
    title: text,
    body: text,
    hold_sec: positive,
    rest_sec: positive,
    reps: positive,
    sets: positive,
    session_sec: z.number().int().nonnegative(),
  })
  .transform(
    (l): PelvicLevel => ({
      code: l.code,
      number: l.number,
      title: l.title,
      body: l.body,
      holdSec: l.hold_sec,
      restSec: l.rest_sec,
      reps: l.reps,
      sets: l.sets,
      sessionSec: l.session_sec,
    }),
  );

const weekDaySchema = z.object({ date: ymd, done: z.boolean().catch(false) });

const programSchema = z
  .object({
    started_on: ymd,
    week: positive.catch(1),
    weeks_total: positive.catch(8),
    completed: z.boolean().catch(false),
    level: levelSchema.nullable().catch(null),
    streak_days: z.number().int().nonnegative().catch(0),
    today_done: z.boolean().catch(false),
    week_days: z.array(z.unknown()).catch([]),
  })
  .transform(
    (p): PelvicProgram => ({
      startedOn: p.started_on,
      week: p.week,
      weeksTotal: p.weeks_total,
      completed: p.completed,
      level: p.level,
      streakDays: p.streak_days,
      todayDone: p.today_done,
      weekDays: p.week_days.flatMap((raw): PelvicWeekDay[] => {
        const parsed = weekDaySchema.safeParse(raw);
        return parsed.success ? [parsed.data] : [];
      }),
    }),
  );

export const pelvicDiarySchema = z
  .object({
    date: ymd,
    leak: z.enum(PELVIC_LEAKS).nullable().catch(null),
    night_voids: z.number().int().nonnegative().nullable().catch(null),
    uti_symptoms: z.array(z.unknown()).catch([]),
    uti_alert: z.boolean().catch(false),
  })
  .transform(
    (d): PelvicDiary => ({
      date: d.date,
      leak: d.leak,
      nightVoids: d.night_voids,
      utiSymptoms: d.uti_symptoms.filter((s): s is PelvicUtiSymptom =>
        (PELVIC_UTI_SYMPTOMS as readonly unknown[]).includes(s),
      ),
      utiAlert: d.uti_alert,
    }),
  );

export const pelvicOverviewSchema = z
  .object({ program: programSchema.nullable().catch(null), diary_today: pelvicDiarySchema })
  .transform((o): PelvicOverview => ({ program: o.program, diaryToday: o.diary_today }));

const alertItemSchema = z.object({ code: z.string(), title: text });

export const pelvicAlertsSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): PelvicAlerts => {
    const titles = new Map<string, string | null>();
    for (const raw of g.items) {
      const parsed = alertItemSchema.safeParse(raw);
      if (parsed.success) titles.set(parsed.data.code, parsed.data.title);
    }
    return { utiWarning: titles.get('uti_warning') ?? null, suitability: titles.get('program_suitability') ?? null };
  });
