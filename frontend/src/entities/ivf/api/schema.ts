import { z } from 'zod';

import {
  IVF_ROLES,
  IVF_ROUTES,
  IVF_STAGES,
  type IvfCycle,
  type IvfDose,
  type IvfDoseDay,
  type IvfHome,
  type IvfCyclePatch,
  type IvfCycleStartInput,
  type IvfNextAppointment,
  type IvfProtocol,
  type IvfStageInfo,
} from '../model/types';

const nullableText = z.string().nullable().catch(null);
const nullableInt = z.number().int().nullable().catch(null);

const stepSchema = z.object({
  stage: z.enum(IVF_STAGES),
  status: z.enum(['done', 'current', 'todo']).catch('todo'),
  date: nullableText,
});

export const ivfCycleSchema = z
  .object({
    id: z.number().int(),
    number: z.number().int().min(1).catch(1),
    protocol: nullableText,
    stage: z.enum(IVF_STAGES),
    status: z.enum(['open', 'closed']).catch('open'),
    started_on: z.string(),
    stim_started_on: nullableText,
    retrieval_at: nullableText,
    transfer_at: nullableText,
    next_scan_at: nullableText,
    beta_on: nullableText,
    notify_companion: z.boolean().catch(false),
    stage_day: nullableInt,
    days_to_beta: nullableInt,
    timeline: z.array(z.unknown()).catch([]),
  })
  .transform(
    (d): IvfCycle => ({
      id: d.id,
      number: d.number,
      protocol: d.protocol,
      stage: d.stage,
      status: d.status,
      startedOn: d.started_on,
      stimStartedOn: d.stim_started_on,
      retrievalAt: d.retrieval_at,
      transferAt: d.transfer_at,
      nextScanAt: d.next_scan_at,
      betaOn: d.beta_on,
      notifyCompanion: d.notify_companion,
      stageDay: d.stage_day,
      daysToBeta: d.days_to_beta,
      // A malformed step is dropped, never the timeline.
      timeline: d.timeline.flatMap((raw) => {
        const step = stepSchema.safeParse(raw);
        return step.success ? [step.data] : [];
      }),
    }),
  );

const doseSchema = z
  .object({
    med_id: z.number().int(),
    name: z.string(),
    dose: z.union([z.string(), z.number()]).nullable().catch(null),
    unit: nullableText,
    route: z.enum(IVF_ROUTES).catch('other'),
    role: z.enum(IVF_ROLES).catch('other'),
    is_trigger: z.boolean().catch(false),
    date: z.string(),
    slot: z.string().regex(/^\d{2}:\d{2}$/),
    taken: z.boolean().catch(false),
    site: nullableText,
  })
  .transform(
    (d): IvfDose => ({
      medId: d.med_id,
      name: d.name,
      dose: d.dose === null ? null : String(d.dose),
      unit: d.unit,
      route: d.route,
      role: d.role,
      isTrigger: d.is_trigger,
      date: d.date,
      slot: d.slot,
      taken: d.taken,
      site: d.site,
    }),
  );

export const ivfDoseDaySchema = z
  .object({ date: z.string(), doses: z.array(z.unknown()).catch([]) })
  .transform(
    (d): IvfDoseDay => ({
      date: d.date,
      doses: d.doses.flatMap((raw) => {
        const dose = doseSchema.safeParse(raw);
        return dose.success ? [dose.data] : [];
      }),
    }),
  );

const nextAppointmentSchema = z
  .object({
    kind: z.enum(['scan', 'retrieval', 'transfer', 'beta']),
    appointment: z.object({
      id: z.number().int(),
      title: z.string(),
      scheduled_at: z.string(),
      days_until: nullableInt,
      prep: z
        .array(z.object({ text: z.string() }).passthrough())
        .catch([])
        .default([]),
    }),
  })
  .transform(
    (d): IvfNextAppointment => ({
      kind: d.kind,
      id: d.appointment.id,
      title: d.appointment.title,
      scheduledAt: d.appointment.scheduled_at,
      daysUntil: d.appointment.days_until,
      prep: d.appointment.prep[0]?.text.trim() || null,
    }),
  );

export const ivfHomeSchema = z
  .object({
    enabled: z.boolean().catch(false),
    cycle: ivfCycleSchema.nullable().catch(null),
    cycles_count: z.number().int().catch(0),
    today: ivfDoseDaySchema,
    next_appointment: nextAppointmentSchema.nullable().catch(null),
    companion: z
      .object({ linked: z.boolean().catch(false), notify: z.boolean().catch(false) })
      .catch({ linked: false, notify: false }),
  })
  .transform(
    (d): IvfHome => ({
      enabled: d.enabled,
      cycle: d.cycle,
      cyclesCount: d.cycles_count,
      today: d.today,
      nextAppointment: d.next_appointment,
      companion: d.companion,
    }),
  );

/** The part of `IvfMedsView` (dose log / undo answers) the home reuses. */
export const ivfMedsTodaySchema = z.object({ today: ivfDoseDaySchema }).transform((d) => d.today);

const text = z.string().trim().min(1).nullable().catch(null);

/** `GET /catalog/ivf_stages` → `data`; malformed items are dropped, never the list. */
export const ivfStagesSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): IvfStageInfo[] =>
    g.items.flatMap((raw) => {
      const item = z.object({ code: z.string(), title: text, body: text }).safeParse(raw);
      return item.success ? [item.data] : [];
    }),
  );

/** `GET /catalog/ivf_protocols` → `data` (CB-IVF-06b setup); malformed items are dropped, never the list. */
export const ivfProtocolsSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): IvfProtocol[] =>
    g.items.flatMap((raw) => {
      const item = z.object({ code: z.string(), title: text, body: text }).safeParse(raw);
      return item.success ? [item.data] : [];
    }),
  );

/** `POST /ivf/cycles` body (CB-IVF-06b): null keys are left out → the API defaults. */
export function toIvfCycleStartBody(input: IvfCycleStartInput): Record<string, string> {
  const body: Record<string, string> = {};
  if (input.protocol) body.protocol = input.protocol;
  if (input.stage) body.stage = input.stage;
  if (input.startedOn) body.started_on = input.startedOn;
  if (input.stimStartedOn) body.stim_started_on = input.stimStartedOn;
  return body;
}

const PATCH_KEYS: readonly (readonly [keyof IvfCyclePatch, string])[] = [
  ['protocol', 'protocol'],
  ['stage', 'stage'],
  ['startedOn', 'started_on'],
  ['stimStartedOn', 'stim_started_on'],
  ['nextScanAt', 'next_scan_at'],
  ['retrievalAt', 'retrieval_at'],
  ['transferAt', 'transfer_at'],
  ['betaOn', 'beta_on'],
];

/** `PUT /ivf/cycles/current` body (CB-IVF-06b): only the keys present (a `null` clears the date). */
export function toIvfCyclePatchBody(patch: IvfCyclePatch): Record<string, string | null> {
  const body: Record<string, string | null> = {};
  for (const [key, wire] of PATCH_KEYS) {
    const value = patch[key];
    if (value !== undefined) body[wire] = value;
  }
  return body;
}
