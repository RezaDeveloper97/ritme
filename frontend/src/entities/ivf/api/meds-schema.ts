import { z } from 'zod';

import {
  IVF_STOCK_UNITS,
  type IvfGuidance,
  type IvfInjectionSite,
  type IvfInventory,
  type IvfMed,
  type IvfMedInput,
  type IvfMedPreset,
  type IvfMedsView,
  type IvfSites,
  type IvfTrigger,
} from '../model/meds';
import { IVF_ROLES, IVF_ROUTES } from '../model/types';
import { ivfCycleSchema, ivfDoseDaySchema } from './schema';

/* `GET /ivf/meds` (IvfMedsView) + the catalog groups of nbl_IVF_Meds (CB-IVF-03). */

const nullableText = z.string().nullable().catch(null);
const nullableInt = z.number().int().nullable().catch(null);
const clock = z.string().regex(/^\d{2}:\d{2}$/);
const amount = z
  .union([z.string(), z.number()])
  .nullable()
  .catch(null)
  .transform((v) => (v === null ? null : String(v)));

/** Keep the valid items of a list, never fail the list. */
function each<T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((items) =>
      items.flatMap((raw) => {
        const item = schema.safeParse(raw);
        return item.success ? [item.data] : [];
      }),
    );
}

const inventorySchema = z
  .object({
    stock_units: z.number().int(),
    stock_unit: z.enum(IVF_STOCK_UNITS).catch('other'),
    doses_per_unit: z.number().int().min(1).catch(1),
    counted_at: z.string(),
    doses_left: z.number().int().min(0).catch(0),
    units_left: z.number().int().min(0).catch(0),
    days_left: nullableInt,
    runs_out_on: nullableText,
    low: z.boolean().catch(false),
  })
  .transform(
    (d): IvfInventory => ({
      stockUnits: d.stock_units,
      stockUnit: d.stock_unit,
      dosesPerUnit: d.doses_per_unit,
      countedAt: d.counted_at,
      dosesLeft: d.doses_left,
      unitsLeft: d.units_left,
      daysLeft: d.days_left,
      runsOutOn: d.runs_out_on,
      low: d.low,
    }),
  );

const medSchema = z
  .object({
    id: z.number().int(),
    reminder_id: z.number().int().nullable().catch(null),
    name: z.string(),
    role: z.enum(IVF_ROLES).catch('other'),
    route: z.enum(IVF_ROUTES).catch('other'),
    is_trigger: z.boolean().catch(false),
    trigger_at: nullableText,
    dose: amount,
    unit: nullableText,
    times: each(clock),
    starts_on: nullableText,
    ends_on: nullableText,
    is_active: z.boolean().catch(true),
    notes: nullableText,
    inventory: inventorySchema.nullable().catch(null),
  })
  .transform(
    (d): IvfMed => ({
      id: d.id,
      reminderId: d.reminder_id,
      name: d.name,
      role: d.role,
      route: d.route,
      isTrigger: d.is_trigger,
      triggerAt: d.trigger_at,
      dose: d.dose,
      unit: d.unit,
      times: d.times,
      startsOn: d.starts_on,
      endsOn: d.ends_on,
      isActive: d.is_active,
      notes: d.notes,
      inventory: d.inventory,
    }),
  );

const triggerSchema = z
  .object({
    med_id: z.number().int(),
    name: z.string(),
    dose: amount,
    unit: nullableText,
    trigger_at: z.string(),
    taken: z.boolean().catch(false),
  })
  .transform(
    (d): IvfTrigger => ({
      medId: d.med_id,
      name: d.name,
      dose: d.dose,
      unit: d.unit,
      triggerAt: d.trigger_at,
      taken: d.taken,
    }),
  );

const sitesSchema = z
  .object({
    codes: each(z.string().min(1)),
    last: z.object({ site: z.string(), date: z.string(), slot: z.string() }).nullable().catch(null),
    suggested: nullableText,
  })
  .catch({ codes: [], last: null, suggested: null })
  .transform((d): IvfSites => ({ codes: d.codes, last: d.last, suggested: d.suggested }));

export const ivfMedsViewSchema = z
  .object({
    cycle: ivfCycleSchema.nullable().catch(null),
    trigger: triggerSchema.nullable().catch(null),
    today: ivfDoseDaySchema,
    tomorrow: ivfDoseDaySchema,
    sites: sitesSchema,
    meds: each(medSchema),
  })
  .transform(
    (d): IvfMedsView => ({
      cycle: d.cycle,
      trigger: d.trigger,
      today: d.today,
      tomorrow: d.tomorrow,
      sites: d.sites,
      meds: d.meds,
    }),
  );

/** `IvfMedInput` → the API's snake_case body. */
export function toIvfMedBody(input: IvfMedInput): Record<string, unknown> {
  return {
    name: input.name,
    role: input.role,
    route: input.route,
    dose: input.dose,
    unit: input.unit,
    times: input.times,
    trigger_at: input.triggerAt,
    starts_on: input.startsOn,
    ends_on: input.endsOn,
    notes: input.notes,
    stock_units: input.stockUnits,
    stock_unit: input.stockUnits === null ? null : input.stockUnit,
    doses_per_unit: input.stockUnits === null ? null : input.dosesPerUnit,
  };
}

const text = z.string().trim().min(1).nullable().catch(null);
const catalogMeta = z.record(z.string(), z.unknown()).nullable().catch(null);

/** `GET /catalog/{group}` → `data.items` (malformed items dropped, never the list). */
function catalogItems<T>(item: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .object({ items: each(item) })
    .catch({ items: [] })
    .transform((g) => g.items);
}

export const ivfInjectionSitesSchema = catalogItems(
  z.object({ code: z.string().min(1), title: text, body: text, meta: catalogMeta }).transform(
    (d): IvfInjectionSite => {
      const region = z.enum(['abdomen', 'thigh', 'arm']).safeParse(d.meta?.region);
      const side = z.enum(['right', 'left']).safeParse(d.meta?.side);
      return {
        code: d.code,
        title: d.title,
        body: d.body,
        region: region.success ? region.data : null,
        side: side.success ? side.data : null,
      };
    },
  ),
);

const presetMeta = z.object({
  role: z.enum(IVF_ROLES),
  route: z.enum(IVF_ROUTES).catch('other'),
  unit: nullableText.default(null),
  times: each(clock).default([]),
  stock_unit: z.enum(IVF_STOCK_UNITS).nullable().catch(null).default(null),
});

export const ivfMedPresetsSchema = catalogItems(
  z.object({ code: z.string().min(1), title: text, meta: presetMeta }).transform(
    (d): IvfMedPreset => ({
      code: d.code,
      title: d.title,
      role: d.meta.role,
      route: d.meta.route,
      unit: d.meta.unit,
      times: d.meta.times,
      stockUnit: d.meta.stock_unit,
    }),
  ),
);

export const ivfGuidanceSchema = catalogItems(
  z.object({ code: z.string().min(1), title: text, body: text }).transform((d): IvfGuidance => d),
);
