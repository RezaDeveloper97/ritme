import { z } from 'zod';

import type { ChildMeasurement, GrowthVerdict, IndicatorValue } from '@/entities/child';

import type { GrowthIndicator, GrowthPoint, GrowthReferenceRow, GrowthSeries, MeasurementList } from '../model/types';

/*
 * Parsers of `GET /children/{id}/measurements` and `GET /children/{id}/growth`
 * (B-N5-02, `internal/children/growthview.go`). Strict on ids and dates, lenient
 * (`.catch`) on display fields; a bad reference row is dropped, never fatal.
 */

const text = z.string().nullable().catch(null);
const num = z.number().nullable().catch(null);
const date = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

const verdictSchema = z
  .object({ status: z.enum(['normal', 'check', 'unknown']).catch('unknown'), label: z.string().catch('') })
  .transform((d): GrowthVerdict => d)
  .catch({ status: 'unknown', label: '' });

const indicatorSchema = z
  .object({ value: z.number(), percentile: num, in_band: z.boolean().nullable().catch(null) })
  .transform((d): IndicatorValue => ({ value: d.value, percentile: d.percentile, inBand: d.in_band }))
  .nullable()
  .catch(null);

export const measurementSchema = z
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

/** Keeps the rows that parse (a malformed one never blanks the list). */
function each<T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((list) =>
      list.flatMap((raw) => {
        const parsed = schema.safeParse(raw);
        return parsed.success ? [parsed.data] : [];
      }),
    );
}

export const measurementListSchema = z
  .object({
    measurements: each(measurementSchema),
    verdict: verdictSchema,
    disclaimer: text,
  })
  .transform((d): MeasurementList => ({ measurements: d.measurements, verdict: d.verdict, disclaimer: d.disclaimer }));

const referenceRowSchema = z
  .object({ month: z.number(), p3: z.number(), p15: z.number(), p50: z.number(), p85: z.number(), p97: z.number() })
  .transform((d): GrowthReferenceRow => d);

const pointSchema = z
  .object({
    id: z.number().int().nullable().catch(null),
    measured_on: date,
    age_months: z.number(),
    value: z.number(),
    percentile: num,
    in_band: z.boolean().nullable().catch(null),
  })
  .transform(
    (d): GrowthPoint => ({
      id: d.id,
      measuredOn: d.measured_on,
      ageMonths: d.age_months,
      value: d.value,
      percentile: d.percentile,
      inBand: d.in_band,
    }),
  );

export const growthSeriesSchema = z
  .object({
    indicator: z.enum(['weight', 'length', 'head']),
    label: z.string().catch(''),
    unit_label: z.string().catch(''),
    available: z.boolean().catch(false),
    reason: text,
    range: z
      .object({ from_month: z.number().catch(0), to_month: z.number().catch(12) })
      .catch({ from_month: 0, to_month: 12 }),
    band: z.object({ label: z.string().catch('') }).catch({ label: '' }),
    median_label: text,
    reference: each(referenceRowSchema),
    points: each(pointSchema),
    latest: pointSchema.nullable().catch(null),
    verdict: verdictSchema,
    disclaimer: text,
  })
  .transform(
    (d): GrowthSeries => ({
      indicator: d.indicator as GrowthIndicator,
      label: d.label,
      unitLabel: d.unit_label,
      available: d.available && d.reference.length > 0,
      reason: d.reason,
      fromMonth: d.range.from_month,
      toMonth: Math.max(d.range.to_month, d.range.from_month + 1),
      bandLabel: d.band.label,
      medianLabel: d.median_label,
      reference: d.reference,
      points: d.points,
      latest: d.latest,
      verdict: d.verdict,
      disclaimer: d.disclaimer,
    }),
  );
