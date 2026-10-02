import { z } from 'zod';

import {
  BREAST_SYMPTOMS,
  DELIVERY_TYPES,
  EPDS_KINDS,
  LOCHIA_AMOUNTS,
  LOCHIA_COLORS,
  PAIN_LEVELS,
  PAIN_LOCATIONS,
  type AlertAction,
  type EpdsCheck,
  type EpdsQuestionnaire,
  type EpdsResult,
  type PostpartumAlert,
  type PostpartumCheckin,
  type PostpartumOverview,
  type PostpartumProfile,
  type PostpartumRecovery,
  type PostpartumStatus,
  type PostpartumText,
  type PostpartumWeekTip,
  type SafetyMessage,
} from '../model/types';

/*
 * Parsers of `/api/v1/postpartum*` (B-N5-01). Lenient on display fields
 * (`.catch`) so one odd value never blanks the home; strict on what a screen
 * can't do without. The safety message is parsed so that it can never be lost:
 * a bad action is dropped, never the message.
 */

const text = z.string().nullable().catch(null);
const int = z.number().int().nullable().catch(null);
const kind = z.enum(EPDS_KINDS);

function enumList<T extends readonly [string, ...string[]]>(values: T) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((list) => list.filter((v): v is T[number] => (values as readonly unknown[]).includes(v)));
}

const actionSchema = z.union([
  z.object({ type: z.literal('call'), number: z.string().regex(/^\d{3,6}$/), label: text.optional() }),
  z.object({ type: z.literal('open_check'), kind, label: text.optional() }),
]);

function toAction(raw: unknown): AlertAction | null {
  const parsed = actionSchema.safeParse(raw);
  if (!parsed.success) return null;
  const a = parsed.data;
  return a.type === 'call'
    ? { type: 'call', number: a.number, label: a.label ?? null }
    : { type: 'open_check', kind: a.kind, label: a.label ?? null };
}

const actionList = z
  .array(z.unknown())
  .catch([])
  .transform((list) => list.map(toAction).filter((a): a is AlertAction => a !== null));

export const alertSchema = z
  .object({
    key: z.string(),
    level: z.string().catch('info'),
    title: text,
    body: text,
    action_label: text.optional(),
    action: z.unknown().optional(),
  })
  .transform(
    (d): PostpartumAlert => ({
      key: d.key,
      level: d.level,
      title: d.title,
      body: d.body,
      actionLabel: d.action_label ?? null,
      action: toAction(d.action),
    }),
  );

const alertList = z
  .array(z.unknown())
  .catch([])
  .transform((list) =>
    list.flatMap((raw) => {
      const parsed = alertSchema.safeParse(raw);
      return parsed.success ? [parsed.data] : [];
    }),
  );

const profileSchema = z
  .object({
    birth_date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
    delivery_type: z.enum(DELIVERY_TYPES).nullable().catch(null),
    delivery_type_label: text,
    baby_count: z.number().int().catch(1),
    source: z.string().catch('direct'),
  })
  .transform(
    (d): PostpartumProfile => ({
      birthDate: d.birth_date,
      deliveryType: d.delivery_type,
      deliveryTypeLabel: d.delivery_type_label,
      babyCount: d.baby_count,
      source: d.source,
    }),
  );

const statusSchema = z
  .object({
    days_since_birth: z.number().int(),
    weeks: z.number().int(),
    days: z.number().int(),
    week: z.number().int(),
    phase: z.string().catch(''),
    phase_label: text,
    puerperium_days: z.number().int().catch(42),
    puerperium_days_left: z.number().int().catch(0),
    progress: z.number().catch(0),
  })
  .transform(
    (d): PostpartumStatus => ({
      daysSinceBirth: d.days_since_birth,
      weeks: d.weeks,
      days: d.days,
      week: d.week,
      phase: d.phase,
      phaseLabel: d.phase_label,
      puerperiumDays: d.puerperium_days,
      puerperiumDaysLeft: d.puerperium_days_left,
      progress: Math.max(0, Math.min(1, d.progress)),
    }),
  );

export const epdsCheckSchema = z
  .object({
    id: z.number().int(),
    kind,
    taken_on: z.string(),
    week: int,
    total: z.number().int(),
    max: z.number().int(),
    band: z.string().catch('low'),
    band_label: text,
    urgent: z.boolean().catch(false),
  })
  .transform(
    (d): EpdsCheck => ({
      id: d.id,
      kind: d.kind,
      takenOn: d.taken_on,
      week: d.week,
      total: d.total,
      max: d.max,
      band: d.band,
      bandLabel: d.band_label,
      urgent: d.urgent,
    }),
  );

const checkinSchema = z
  .object({
    due: kind.nullable().catch(null),
    next_due_on: text,
    last: epdsCheckSchema.nullable().catch(null),
  })
  .transform((d): PostpartumCheckin => ({ due: d.due, nextDueOn: d.next_due_on, last: d.last }));

export const recoverySchema = z
  .object({
    date: z.string(),
    lochia_amount: z.enum(LOCHIA_AMOUNTS).nullable().catch(null),
    lochia_color: z.enum(LOCHIA_COLORS).nullable().catch(null),
    pain_level: z.enum(PAIN_LEVELS).nullable().catch(null),
    pain_locations: enumList(PAIN_LOCATIONS),
    breasts: enumList(BREAST_SYMPTOMS).nullable().catch(null),
    feeds_count: int,
    sleep_hours: z.number().nullable().catch(null),
    alerts: alertList,
  })
  .transform(
    (d): PostpartumRecovery => ({
      date: d.date,
      lochiaAmount: d.lochia_amount,
      lochiaColor: d.lochia_color,
      painLevel: d.pain_level,
      painLocations: d.pain_locations,
      breasts: d.breasts,
      feedsCount: d.feeds_count,
      sleepHours: d.sleep_hours,
      alerts: d.alerts,
    }),
  );

const textBlock = z
  .object({ title: text, body: text })
  .transform((d): PostpartumText => ({ title: d.title, body: d.body }));

const weekTipSchema = z
  .object({ week: z.number().int(), title: text, body: text })
  .transform((d): PostpartumWeekTip => ({ week: d.week, title: d.title, body: d.body }));

export const overviewSchema = z
  .object({
    active: z.boolean(),
    mode: z.string().catch('cycle'),
    setup_required: z.boolean().catch(false),
    profile: profileSchema.nullable().catch(null),
    status: statusSchema.nullable().catch(null),
    checkin: checkinSchema.nullable().catch(null),
    today: recoverySchema.nullable().catch(null),
    alerts: alertList,
    week_tip: weekTipSchema.nullable().catch(null),
    call_when: textBlock.nullable().catch(null),
  })
  .transform(
    (d): PostpartumOverview => ({
      active: d.active,
      mode: d.mode,
      setupRequired: d.setup_required,
      profile: d.profile,
      status: d.status,
      checkin: d.checkin,
      today: d.today,
      alerts: d.alerts,
      weekTip: d.week_tip,
      callWhen: d.call_when,
    }),
  );

export const questionnaireSchema = z
  .object({
    kind,
    intro: text,
    disclaimer: text,
    max: z.number().int(),
    items: z
      .array(
        z.object({
          code: z.string(),
          number: z.number().int().catch(0),
          text: z.string(),
          options: z.array(z.object({ score: z.number().int().min(0).max(3), label: z.string() })).min(2),
        }),
      )
      .min(1),
  })
  .transform(
    (d): EpdsQuestionnaire => ({
      kind: d.kind,
      intro: d.intro,
      disclaimer: d.disclaimer,
      max: d.max,
      items: d.items.map((it) => ({ code: it.code, number: it.number, text: it.text, options: it.options })),
    }),
  );

const safetySchema = z
  .object({
    level: z.string(),
    title: text,
    body: text,
    actions: actionList,
  })
  .transform((d): SafetyMessage => ({ level: d.level, title: d.title, body: d.body, actions: d.actions }));

export const epdsResultSchema = z
  .object({
    check: epdsCheckSchema,
    // Never `.catch(null)` on an urgent message: a malformed one still renders (screen falls back to the numbers).
    safety: z
      .unknown()
      .optional()
      .transform((raw): SafetyMessage | null => {
        if (raw === null || raw === undefined) return null;
        const parsed = safetySchema.safeParse(raw);
        if (parsed.success) return parsed.data;
        const level = typeof raw === 'object' && 'level' in raw ? String((raw as { level: unknown }).level) : 'urgent';
        return { level, title: null, body: null, actions: [] };
      }),
    follow_up: kind.nullable().catch(null),
    checkin: checkinSchema.nullable().catch(null),
  })
  .transform(
    (d): EpdsResult => ({ check: d.check, safety: d.safety, followUp: d.follow_up, checkin: d.checkin }),
  );
