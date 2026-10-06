import { z } from 'zod';

import type { HealthRecord, Section } from '../model/types';

/*
 * Boundary schemas of `GET /api/v1/health-record` (bloom B-N6-03). Sections arrive as an ordered list
 * `[{key, editable, empty, data}]`; the screen reads them by key. Unknown sections (roadmap CB-REC / CB-MENO
 * providers) are ignored here until a slice reads them.
 */

const num = z.number();
const nnum = z.number().nullable();
const nstr = z.string().nullable();
const codes = z.array(z.string()).nullable();

const valueSummary = z
  .object({ avg: num, min: num, max: num, readings: num, in_target_percent: num.optional() })
  .nullable()
  .transform((v) =>
    v ? { avg: v.avg, min: v.min, max: v.max, readings: v.readings, inTargetPercent: v.in_target_percent ?? null } : null,
  );

const basics = z
  .object({
    height_cm: nnum,
    weight_kg: nnum,
    bmi: z.object({ value: num, category: z.string() }).nullable(),
    blood_type: nstr,
    blood_type_source: z.enum(['record', 'pregnancy']).nullable(),
  })
  .transform((d) => ({
    heightCm: d.height_cm,
    weightKg: d.weight_kg,
    bmi: d.bmi,
    bloodType: d.blood_type,
    bloodTypeSource: d.blood_type_source,
  }));

const conditions = z
  .object({ chronic_illnesses: codes, gyn_conditions: codes, answered: z.boolean() })
  .transform((d) => ({ chronicIllnesses: d.chronic_illnesses, gynConditions: d.gyn_conditions, answered: d.answered }));

const medications = z
  .object({
    items: z.array(
      z.object({
        id: num,
        title: z.string(),
        dose: nstr,
        recurrence: z.string(),
        weekdays: z.array(num),
        times: z.array(z.string()),
        notes: nstr.optional(),
      }),
    ),
    profile_medications: codes,
  })
  .transform((d) => ({
    items: d.items.map((m) => ({ ...m, notes: m.notes ?? null })),
    profileMedications: d.profile_medications,
  }));

const allergies = z.object({ items: codes, answered: z.boolean() });

const cycle = z
  .object({
    based_on: num,
    median_cycle: nnum,
    variation: nnum,
    median_period: nnum,
    regularity: z.string(),
    last_period_start: nstr,
    top_symptoms: z.array(z.object({ key: z.string(), label: z.string(), days: num })),
  })
  .transform((d) => ({
    basedOn: d.based_on,
    medianCycle: d.median_cycle,
    variation: d.variation,
    medianPeriod: d.median_period,
    regularity: d.regularity,
    lastPeriodStart: d.last_period_start,
    topSymptoms: d.top_symptoms,
  }));

const vitals = z
  .object({
    days: num,
    blood_pressure: z
      .object({ systolic: num, diastolic: num, readings: num, tone: z.string() })
      .nullable(),
    heart_rate: valueSummary,
    glucose_fasting: valueSummary,
    glucose_after_meal: valueSummary,
    glucose_other: valueSummary,
  })
  .transform((d) => ({
    days: d.days,
    bloodPressure: d.blood_pressure,
    heartRate: d.heart_rate,
    glucoseFasting: d.glucose_fasting,
    glucoseAfterMeal: d.glucose_after_meal,
    glucoseOther: d.glucose_other,
  }));

export const pregnancyEntrySchema = z
  .object({
    id: nnum,
    source: z.enum(['tracked', 'manual']),
    outcome: z.enum(['ongoing', 'birth', 'vaginal', 'cesarean', 'ended']),
    date: nstr,
    baby_count: nnum,
    editable: z.boolean(),
  })
  .transform((e) => ({
    id: e.id,
    source: e.source,
    outcome: e.outcome,
    date: e.date,
    babyCount: e.baby_count,
    editable: e.editable,
  }));

const pregnancies = z
  .object({ pregnancies_count: num, births_count: num, items: z.array(pregnancyEntrySchema) })
  .transform((d) => ({ pregnanciesCount: d.pregnancies_count, birthsCount: d.births_count, items: d.items }));

const checkups = z.object({
  items: z.array(
    z
      .object({ id: num, title: z.string(), done_on: z.string(), result: z.string() })
      .transform((c) => ({ id: c.id, title: c.title, doneOn: c.done_on, result: c.result })),
  ),
});

const labs = z.object({
  items: z.array(
    z
      .object({
        id: num,
        title: z.string(),
        date: z.string(),
        marker_count: num,
        attention_count: num,
        all_normal: z.boolean(),
        attention: z.array(z.object({ name: z.string(), state_label: z.string() })),
      })
      .transform((l) => ({
        id: l.id,
        title: l.title,
        date: l.date,
        markerCount: l.marker_count,
        attentionCount: l.attention_count,
        allNormal: l.all_normal,
        attention: l.attention.map((a) => ({ name: a.name, stateLabel: a.state_label })),
      })),
  ),
});

const SECTION_SCHEMAS = {
  basics,
  conditions,
  medications,
  allergies,
  cycle,
  vitals,
  pregnancies,
  checkups,
  labs,
} as const;

type SectionKey = keyof typeof SECTION_SCHEMAS;

const rawSection = z.object({ key: z.string(), editable: z.boolean(), empty: z.boolean(), data: z.unknown() });

export const healthRecordSchema = z
  .object({
    date: z.string(),
    updated_at: nstr,
    person: z.object({ name: nstr, age: nnum, gender: nstr, life_mode: z.string() }),
    sections: z.array(rawSection),
  })
  .transform((d, ctx): HealthRecord => {
    const byKey = new Map(d.sections.map((s) => [s.key, s]));
    const read = <K extends SectionKey>(key: K): Section<z.output<(typeof SECTION_SCHEMAS)[K]>> => {
      const s = byKey.get(key);
      if (!s) {
        ctx.addIssue({ code: 'custom', message: `section ${key} missing` });
        return z.NEVER;
      }
      return { editable: s.editable, empty: s.empty, data: SECTION_SCHEMAS[key].parse(s.data) as z.output<(typeof SECTION_SCHEMAS)[K]> };
    };
    return {
      date: d.date,
      updatedAt: d.updated_at,
      person: { name: d.person.name, age: d.person.age, gender: d.person.gender, lifeMode: d.person.life_mode },
      basics: read('basics'),
      conditions: read('conditions'),
      medications: read('medications'),
      allergies: read('allergies'),
      cycle: read('cycle'),
      vitals: read('vitals'),
      pregnancies: read('pregnancies'),
      checkups: read('checkups'),
      labs: read('labs'),
    };
  });
