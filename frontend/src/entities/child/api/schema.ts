import { z } from 'zod';

import {
  CHILD_DELIVERY_TYPES,
  CHILD_SEXES,
  type Child,
  type ChildHome,
  type ChildMeasurement,
  type ChildrenList,
  type CompanionChild,
  type GrowthStatus,
  type IndicatorValue,
  type LearnTipPreview,
  type VaccineSummary,
  type VaccineVisit,
} from '../model/types';

/*
 * Parsers of `/api/v1/children*` (B-N5-02). Strict on identity (id, name,
 * birth date, role), lenient (`.catch`) on display fields so one odd value never
 * blanks a screen.
 */

const text = z.string().nullable().catch(null);
const num = z.number().nullable().catch(null);
const date = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

const visitSchema = z
  .object({
    code: z.string(),
    age_months: z.number().int().catch(0),
    label: z.string().catch(''),
    due_date: date,
    days_left: z.number().int().catch(0),
    status: z.string().catch('upcoming'),
    status_label: text,
    given: z.number().int().catch(0),
    total: z.number().int().catch(0),
    dose_names: z
      .array(z.string().nullable())
      .catch([])
      .transform((l) => l.filter((s): s is string => typeof s === 'string' && s !== '')),
  })
  .transform(
    (d): VaccineVisit => ({
      code: d.code,
      ageMonths: d.age_months,
      label: d.label,
      dueDate: d.due_date,
      daysLeft: d.days_left,
      status: d.status,
      statusLabel: d.status_label,
      given: d.given,
      total: d.total,
      doseNames: d.dose_names,
    }),
  );

const vaccinesSchema = z
  .object({
    given: z.number().int().catch(0),
    total: z.number().int().catch(0),
    completed_visits: z.number().int().catch(0),
    up_to_date: z.boolean().catch(false),
    complete: z.boolean().catch(false),
    next: visitSchema.nullable().catch(null),
  })
  .transform(
    (d): VaccineSummary => ({
      given: d.given,
      total: d.total,
      completedVisits: d.completed_visits,
      upToDate: d.up_to_date,
      complete: d.complete,
      next: d.next,
    }),
  );

const EMPTY_VACCINES: VaccineSummary = { given: 0, total: 0, completedVisits: 0, upToDate: false, complete: false, next: null };

const growthSchema = z.object({
  status: z.enum(['normal', 'check', 'unknown']).catch('unknown' as GrowthStatus),
  label: z.string().catch(''),
});

const childFields = {
  id: z.number().int(),
  name: z.string(),
  initial: z.string().catch(''),
  birth_date: date,
  sex: z.enum(CHILD_SEXES).nullable().catch(null),
  sex_label: text,
  delivery_type: z.enum(CHILD_DELIVERY_TYPES).nullable().catch(null),
  delivery_type_label: text,
  birth: z
    .object({ weight_kg: num, length_cm: num, head_cm: num })
    .catch({ weight_kg: null, length_cm: null, head_cm: null }),
  age: z.object({
    days: z.number().int().catch(0),
    weeks: z.number().int().catch(0),
    months: z.number().int().catch(0),
    years: z.number().int().catch(0),
    days_in_month: z.number().int().catch(0),
    label: z.string().catch(''),
  }),
  role: z.enum(['owner', 'shared']),
  can_edit: z.boolean().catch(false),
  owner_name: text,
  vaccines: vaccinesSchema.catch(EMPTY_VACCINES),
  growth: growthSchema.catch({ status: 'unknown', label: '' }),
};

type RawChild = z.infer<z.ZodObject<typeof childFields>>;

function toChild(d: RawChild): Child {
  return {
    id: d.id,
    name: d.name,
    initial: d.initial || Array.from(d.name.trim())[0] || '',
    birthDate: d.birth_date,
    sex: d.sex,
    sexLabel: d.sex_label,
    deliveryType: d.delivery_type,
    deliveryTypeLabel: d.delivery_type_label,
    birth: { weightKg: d.birth.weight_kg, lengthCm: d.birth.length_cm, headCm: d.birth.head_cm },
    age: {
      days: d.age.days,
      weeks: d.age.weeks,
      months: d.age.months,
      years: d.age.years,
      daysInMonth: d.age.days_in_month,
      label: d.age.label,
    },
    role: d.role,
    canEdit: d.role === 'owner' && d.can_edit,
    ownerName: d.owner_name,
    vaccines: d.vaccines,
    growth: d.growth,
  };
}

export const childSchema = z.object(childFields).transform(toChild);

export const childrenListSchema = z
  .object({
    children: z
      .array(z.unknown())
      .catch([])
      .transform((list) =>
        list.flatMap((raw) => {
          const parsed = childSchema.safeParse(raw);
          return parsed.success ? [parsed.data] : [];
        }),
      ),
    count: z.number().int().catch(0),
    owned_count: z.number().int().catch(0),
    max_children: z.number().int().catch(10),
    can_add: z.boolean().catch(true),
    sharing_note: text,
  })
  .transform(
    (d): ChildrenList => ({
      children: d.children,
      count: d.children.length,
      ownedCount: d.owned_count,
      maxChildren: d.max_children,
      canAdd: d.can_add,
      sharingNote: d.sharing_note,
    }),
  );

const indicatorSchema = z
  .object({ value: z.number(), percentile: num, in_band: z.boolean().nullable().catch(null) })
  .transform((d): IndicatorValue => ({ value: d.value, percentile: d.percentile, inBand: d.in_band }))
  .nullable()
  .catch(null);

const measurementSchema = z
  .object({
    id: z.number().int().nullable().catch(null),
    source: z.string().catch('measurement'),
    measured_on: date,
    age: z.object({ label: text }).catch({ label: null }),
    weight: indicatorSchema.optional(),
    length: indicatorSchema.optional(),
    head: indicatorSchema.optional(),
  })
  .transform(
    (d): ChildMeasurement => ({
      id: d.id,
      source: d.source,
      measuredOn: d.measured_on,
      ageLabel: d.age.label,
      weight: d.weight ?? null,
      length: d.length ?? null,
      head: d.head ?? null,
    }),
  );

const learnTipSchema = z
  .object({
    code: z.string(),
    topic: z.string().catch(''),
    topic_label: text,
    title: text,
    minutes: z.number().int().nullable().catch(null),
  })
  .transform(
    (d): LearnTipPreview => ({ code: d.code, topic: d.topic, topicLabel: d.topic_label, title: d.title, minutes: d.minutes }),
  );

export const childHomeSchema = z
  .object({
    ...childFields,
    latest: measurementSchema.nullable().catch(null),
    milestones: z
      .object({
        band_months: z.number().int(),
        label: z.string().catch(''),
        checked: z.number().int().catch(0),
        total: z.number().int().catch(0),
      })
      .nullable()
      .catch(null),
    this_week: z
      .object({ weeks: z.number().int().catch(0), months: z.number().int().catch(0), body: text })
      .nullable()
      .catch(null),
    learn: z
      .object({ count: z.number().int().catch(0), featured: learnTipSchema.nullable().catch(null) })
      .catch({ count: 0, featured: null }),
    today: z.unknown().optional(),
  })
  .transform(
    (d): ChildHome => ({
      ...toChild(d),
      latest: d.latest,
      milestones: d.milestones
        ? { bandMonths: d.milestones.band_months, label: d.milestones.label, checked: d.milestones.checked, total: d.milestones.total }
        : null,
      thisWeek: d.this_week,
      learn: d.learn,
      today: d.today ?? null,
    }),
  );

const companionChildSchema = z
  .object({
    id: z.number().int(),
    name: z.string(),
    initial: z.string().catch(''),
    sex: z.enum(CHILD_SEXES).nullable().catch(null),
    age: z.object({ label: z.string().catch('') }).catch({ label: '' }),
    next_vaccine: z
      .object({
        label: z.string().catch(''),
        due_date: date,
        days_left: z.number().int().catch(0),
        status: z.string().catch('upcoming'),
      })
      .nullable()
      .catch(null),
  })
  .transform(
    (d): CompanionChild => ({
      id: d.id,
      name: d.name,
      initial: d.initial || Array.from(d.name.trim())[0] || '',
      sex: d.sex,
      ageLabel: d.age.label,
      nextVaccine: d.next_vaccine
        ? {
            label: d.next_vaccine.label,
            dueDate: d.next_vaccine.due_date,
            daysLeft: d.next_vaccine.days_left,
            status: d.next_vaccine.status,
          }
        : null,
    }),
  );

/**
 * The companion home's `child` card (`{count, items}` or null) as typed
 * children; anything unparseable → []. The companion entity keeps it `unknown`.
 */
export function parseCompanionChildren(raw: unknown): CompanionChild[] {
  const parsed = z.object({ items: z.array(z.unknown()) }).safeParse(raw);
  if (!parsed.success) return [];
  return parsed.data.items.flatMap((item) => {
    const c = companionChildSchema.safeParse(item);
    return c.success ? [c.data] : [];
  });
}
