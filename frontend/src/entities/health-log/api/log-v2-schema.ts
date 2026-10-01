import { z } from 'zod';

import type {
  LogCategory,
  LogDay,
  LogDaysRange,
  LogParam,
  LogParamValue,
  LogPreferences,
  LogTaxonomy,
} from '../model/log-v2';

/**
 * Boundary validation for the taxonomy v2 endpoints (`/logs/*`, B-N3-01/02). snake_case on the wire,
 * camelCase in the app. Privacy (§11): day payloads are health data — never logged from here.
 */

const labeled = z.object({ value: z.string(), label: z.string() });
const range = z.object({ min: z.number(), max: z.number() }).nullable();

const option = z.object({
  value: z.string(),
  label: z.string(),
  modes: z.array(z.string()).nullable().default(null),
  legacy_only: z.boolean().default(false),
});

const param = z
  .object({
    code: z.string(),
    type: z.enum(['single', 'multi', 'items', 'number', 'integer', 'text', 'text_items', 'bool', 'link']),
    label: z.string(),
    modes: z.array(z.string()).default([]),
    detail: z.boolean().default(false),
    alert: z.boolean().default(false),
    options: z.array(option).default([]),
    levels: z.array(labeled).default([]),
    score: range.default(null),
    range: range.default(null),
    scale: z.number().nullable().default(null),
    unit: labeled.nullable().default(null),
    max_length: z.number().nullable().default(null),
    dynamic: z.boolean().default(false),
    source: z.string().nullable().default(null),
  })
  .transform(
    (p): LogParam => ({
      code: p.code,
      type: p.type,
      label: p.label,
      modes: p.modes,
      detail: p.detail,
      alert: p.alert,
      options: p.options.map((o) => ({ value: o.value, label: o.label, modes: o.modes, legacyOnly: o.legacy_only })),
      levels: p.levels,
      score: p.score,
      range: p.range,
      scale: p.scale,
      unit: p.unit,
      maxLength: p.max_length,
      dynamic: p.dynamic,
      source: p.source,
    }),
  );

// An empty PHP-style object may arrive as `[]`; both mean "none".
const emptyAsRecord = <T extends z.ZodTypeAny>(inner: T) =>
  z.union([z.record(z.string(), inner), z.array(z.never()).transform(() => ({}) as Record<string, z.infer<T>>)]);

const category = z
  .object({
    code: z.string(),
    label: z.string(),
    group: labeled,
    modes: z.array(z.string()).default([]),
    conditions: emptyAsRecord(labeled).default({}),
    params: z.array(param).default([]),
  })
  .transform((c): LogCategory => c);

/** GET /logs/taxonomy */
export const logTaxonomySchema = z
  .object({ mode: z.string().nullable(), categories: z.array(category) })
  .transform((t): LogTaxonomy => t);

const itemValue = z.object({ level: z.string(), score: z.number().nullable().default(null) });

/** One stored param value, any of the PUT shapes; anything else is dropped rather than failing the day. */
const paramValue = z.union([
  z.string(),
  z.number(),
  z.boolean(),
  z.array(z.string()),
  z.record(z.string(), itemValue),
  z.record(z.string(), z.string()),
]);

function cleanValues(raw: unknown): Record<string, Record<string, LogParamValue>> {
  const out: Record<string, Record<string, LogParamValue>> = {};
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return out;
  for (const [cat, params] of Object.entries(raw as Record<string, unknown>)) {
    if (!params || typeof params !== 'object' || Array.isArray(params)) continue;
    const next: Record<string, LogParamValue> = {};
    for (const [code, value] of Object.entries(params as Record<string, unknown>)) {
      if (value === null || value === undefined) continue;
      if (Array.isArray(value) && value.length === 0) {
        next[code] = [];
        continue;
      }
      const parsed = paramValue.safeParse(value);
      if (parsed.success) next[code] = parsed.data as LogParamValue;
    }
    if (Object.keys(next).length) out[cat] = next;
  }
  return out;
}

/** GET|PUT|DELETE /logs/days/{date} */
export const logDaySchema = z
  .object({ date: z.string(), categories: z.unknown() })
  .transform((d): LogDay => ({ date: d.date, categories: cleanValues(d.categories) }));

/** GET /logs/days?from&to */
export const logDaysRangeSchema = z
  .object({ from: z.string(), to: z.string(), days: z.array(logDaySchema) })
  .transform((r): LogDaysRange => r);

/** GET /logs/preferences */
export const logPreferencesSchema = z
  .object({
    mode: z.string(),
    phase: z.string().nullable().default(null),
    is_default: z.boolean().default(true),
    max_pinned: z.number().default(8),
    pinned: z.array(z.string()).default([]),
    categories: z
      .array(
        z.object({
          code: z.string(),
          label: z.string(),
          hidden: z.boolean().default(false),
          pinned: z.boolean().default(false),
          custom_param: z.string().nullable().default(null),
        }),
      )
      .default([]),
    max_custom_items: z.number().default(20),
    custom_items: z
      .array(
        z.object({
          id: z.number(),
          code: z.string(),
          category: z.string(),
          param: z.string(),
          label: z.string(),
        }),
      )
      .default([]),
  })
  .transform(
    (p): LogPreferences => ({
      mode: p.mode,
      phase: p.phase,
      isDefault: p.is_default,
      maxPinned: p.max_pinned,
      pinned: p.pinned,
      categories: p.categories.map((c) => ({
        code: c.code,
        label: c.label,
        hidden: c.hidden,
        pinned: c.pinned,
        customParam: c.custom_param,
      })),
      maxCustomItems: p.max_custom_items,
      customItems: p.custom_items,
    }),
  );
