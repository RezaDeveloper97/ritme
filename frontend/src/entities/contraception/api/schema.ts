import { z } from 'zod';

import {
  CONTRACEPTION_METHODS,
  PACK_TYPES,
  type ContraceptionMethod,
  type ContraceptionOverview,
  type MethodReminder,
  type PackDay,
  type PillDay,
  type PillPack,
} from '../model/types';
import type { MissedPillRule } from '../model/missed';

/*
 * Boundary parsers for `/api/v1/contraception` (CLAUDE.md §10 — zod at the
 * edge). Shapes: backend-go/api/openapi.yaml `ContraceptionOverview` and the
 * goldens in backend-go/contract/golden/contraception/.
 */

const date = z.string();
const nullableDate = date.nullable().optional().transform((v) => v ?? null);

const pillDaySchema = z
  .object({
    date,
    kind: z.enum(['active', 'placebo', 'break']),
    status: z.enum(['taken', 'missed', 'pending', 'upcoming', 'untracked']).nullable(),
  })
  .transform((d): PillDay => ({ date: d.date, kind: d.kind, status: d.status }));

const packDaySchema = z
  .object({
    day: z.number().int(),
    date,
    kind: z.enum(['active', 'placebo', 'break']),
    status: z.enum(['taken', 'missed', 'pending', 'upcoming', 'untracked']).nullable(),
  })
  .transform((d): PackDay => ({ day: d.day, date: d.date, kind: d.kind, status: d.status }));

const methodSchema = z
  .object({
    method: z.enum(CONTRACEPTION_METHODS),
    pack_type: z.enum(PACK_TYPES).nullable().optional(),
    pack_started_on: nullableDate,
    packs_left: z.number().int().nullable().optional(),
    reminder: z.object({ enabled: z.boolean(), time: z.string() }).nullable().optional(),
    inserted_on: nullableDate,
    iud_lifetime_years: z.number().int().nullable().optional(),
    followup_on: nullableDate,
    followup_done: z.boolean().optional(),
    iud_replace_on: nullableDate,
    injected_on: nullableDate,
    next_injection_on: nullableDate,
    replace_on: nullableDate,
  })
  .transform(
    (m): ContraceptionMethod => ({
      method: m.method,
      packType: m.pack_type ?? null,
      packStartedOn: m.pack_started_on,
      packsLeft: m.packs_left ?? null,
      reminder: m.reminder ?? null,
      insertedOn: m.inserted_on,
      iudLifetimeYears: m.iud_lifetime_years ?? null,
      followupOn: m.followup_on,
      followupDone: m.followup_done ?? false,
      iudReplaceOn: m.iud_replace_on,
      injectedOn: m.injected_on,
      nextInjectionOn: m.next_injection_on,
      replaceOn: m.replace_on,
    }),
  );

const pillSchema = z
  .object({
    pack_number: z.number().int(),
    pack_day: z.number().int(),
    pack_week: z.number().int(),
    pack_length: z.number().int(),
    active_days: z.number().int(),
    today: pillDaySchema,
    days: z.array(packDaySchema),
    streak_days: z.number().int(),
    missed_count: z.number().int(),
    next_pack_on: date,
    packs_left: z.number().int().nullable().optional(),
    runs_out_on: nullableDate,
    refill_on: nullableDate,
  })
  .transform(
    (p): PillPack => ({
      packNumber: p.pack_number,
      packDay: p.pack_day,
      packWeek: p.pack_week,
      packLength: p.pack_length,
      activeDays: p.active_days,
      today: p.today,
      days: p.days,
      streakDays: p.streak_days,
      missedCount: p.missed_count,
      nextPackOn: p.next_pack_on,
      packsLeft: p.packs_left ?? null,
      runsOutOn: p.runs_out_on,
      refillOn: p.refill_on,
    }),
  );

const reminderSchema = z
  .object({
    kind: z.string(),
    reminder_id: z.number().int(),
    type: z.string(),
    title: z.string(),
    due_on: nullableDate,
    scheduled_at: z.string().nullable().optional(),
    recurrence: z.string(),
    is_active: z.boolean(),
  })
  .transform(
    (r): MethodReminder => ({
      kind: r.kind,
      reminderId: r.reminder_id,
      type: r.type,
      title: r.title,
      dueOn: r.due_on,
      scheduledAt: r.scheduled_at ?? null,
      recurrence: r.recurrence,
      isActive: r.is_active,
    }),
  );

export const contraceptionOverviewSchema = z
  .object({
    tracking: z.boolean(),
    method: methodSchema.nullable(),
    pill: pillSchema.nullable(),
    reminders: z.array(reminderSchema).default([]),
  })
  .transform(
    (o): ContraceptionOverview => ({
      tracking: o.tracking,
      method: o.method,
      pill: o.pill,
      reminders: o.reminders,
    }),
  );

/*
 * `GET /catalog/missed_pill_rules` (CB-CORE-03 catalog, CB-CONTRA-01 seed).
 * The server picks `{fa, en}` objects inside `meta` for the request locale, so
 * a step is normally a string; a still-unpicked object falls back to its first
 * non-empty value. Unknown severities read as `info`.
 */
const localized = z.union([z.string(), z.record(z.string(), z.unknown())]).transform((v) => {
  if (typeof v === 'string') return v;
  const first = Object.values(v).find((x) => typeof x === 'string' && x.trim() !== '');
  return typeof first === 'string' ? first : '';
});

const missedMetaSchema = z
  .object({
    methods: z.array(z.string()).nullable().optional(),
    missed: z.number().int().nullable().optional(),
    severity: z.string().optional(),
    steps: z.array(localized).nullable().optional(),
    pack_week: z.number().int().nullable().optional(),
  })
  .passthrough();

const missedItemSchema = z
  .object({
    code: z.string(),
    title: z.string().nullable(),
    body: z.string().nullable(),
    meta: missedMetaSchema.nullable().optional(),
    needs_review: z.boolean().optional(),
  })
  .transform((item): MissedPillRule => {
    const meta = item.meta ?? {};
    const severity = meta.severity === 'urgent' || meta.severity === 'caution' ? meta.severity : 'info';
    return {
      code: item.code,
      title: item.title,
      body: item.body,
      methods: meta.methods ?? null,
      missed: meta.missed ?? null,
      severity,
      steps: (meta.steps ?? []).filter((s) => s.trim() !== ''),
      packWeek: meta.pack_week ?? null,
      needsReview: item.needs_review ?? false,
    };
  });

export const missedRulesSchema = z
  .object({ items: z.array(missedItemSchema).default([]) })
  .transform((g): MissedPillRule[] => g.items);
