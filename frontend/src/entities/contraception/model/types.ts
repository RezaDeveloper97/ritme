/*
 * Domain shapes of `/api/v1/contraception` (CB-CONTRA-01, OpenAPI tag
 * Contraception). camelCase on this side; the parsers in `api/schema.ts` map
 * the snake_case wire format. Health data (CLAUDE.md §11): never log these.
 */

export const CONTRACEPTION_METHODS = [
  'combined_pill',
  'progestin_pill',
  'copper_iud',
  'hormonal_iud',
  'injection',
  'implant',
  'condom',
  'other',
] as const;
export type ContraceptionMethodCode = (typeof CONTRACEPTION_METHODS)[number];

/** Pack layouts of the combined pill. The progestogen-only pill is always 28 active pills. */
export const PACK_TYPES = ['21_7', '28', '24_4'] as const;
export type PackType = (typeof PACK_TYPES)[number];

export type PillDayKind = 'active' | 'placebo' | 'break';
/** `null` on break days; `untracked` = a day before the method was set up. */
export type PillDayStatus = 'taken' | 'missed' | 'pending' | 'upcoming' | 'untracked' | null;

export interface PillDay {
  date: string;
  kind: PillDayKind;
  status: PillDayStatus;
}

export interface PackDay extends PillDay {
  /** 1…28 */
  day: number;
}

export interface PillReminder {
  enabled: boolean;
  /** HH:MM, Tehran */
  time: string;
}

export interface ContraceptionMethod {
  method: ContraceptionMethodCode;
  packType: PackType | null;
  packStartedOn: string | null;
  packsLeft: number | null;
  reminder: PillReminder | null;
  insertedOn: string | null;
  iudLifetimeYears: number | null;
  followupOn: string | null;
  followupDone: boolean;
  iudReplaceOn: string | null;
  injectedOn: string | null;
  nextInjectionOn: string | null;
  replaceOn: string | null;
}

export interface PillPack {
  packNumber: number;
  packDay: number;
  packWeek: number;
  packLength: number;
  activeDays: number;
  today: PillDay;
  days: PackDay[];
  streakDays: number;
  missedCount: number;
  nextPackOn: string;
  packsLeft: number | null;
  runsOutOn: string | null;
  refillOn: string | null;
}

export type MethodReminderKind =
  | 'pill_refill'
  | 'iud_string_check'
  | 'iud_followup'
  | 'iud_replacement'
  | 'injection_next'
  | 'implant_replacement';

export interface MethodReminder {
  kind: MethodReminderKind | (string & {});
  reminderId: number;
  type: string;
  title: string;
  dueOn: string | null;
  scheduledAt: string | null;
  recurrence: string;
  isActive: boolean;
}

export interface ContraceptionOverview {
  /** The «track contraception» switch (bloom B-N2-01/03). */
  tracking: boolean;
  method: ContraceptionMethod | null;
  pill: PillPack | null;
  reminders: MethodReminder[];
}

/** Body of `PUT /contraception/method` (wire format). */
export interface MethodPayload {
  method: ContraceptionMethodCode;
  pack_type?: PackType | null;
  pack_started_on?: string | null;
  packs_left?: number | null;
  reminder_time?: string | null;
  reminder_enabled?: boolean;
  inserted_on?: string | null;
  iud_lifetime_years?: number | null;
  followup_done?: boolean;
  injected_on?: string | null;
  replace_on?: string | null;
}
