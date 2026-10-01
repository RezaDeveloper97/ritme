import { diffInDays, fromApiDate } from '@/shared/lib/date';

import type { ContraceptionMethodCode, MethodReminder, MethodReminderKind } from './types';

/*
 * Missed-pill guidance (CB-CONTRA-03, nbl_Contra_Missed). The clinical copy is
 * the admin-editable catalog group `missed_pill_rules` (CB-CONTRA-01, meta
 * `{methods, missed?, severity, steps[], pack_week?}`) — never hard-coded here.
 */

export type MissedRuleSeverity = 'info' | 'caution' | 'urgent';

export interface MissedPillRule {
  code: string;
  /** Chip label for a count rule («۱ قرص (کمتر از ۴۸ ساعت دیر)»), headline of an urgent one. */
  title: string | null;
  /** The general-guidance note of a count rule, the copy of an urgent one. */
  body: string | null;
  /** Method codes the rule is for; null = every pill. */
  methods: string[] | null;
  /** 1 = one pill, 2 = two or more; null = shown with every count. */
  missed: number | null;
  severity: MissedRuleSeverity;
  steps: string[];
  /** The pack week the rule is about (`week1_unprotected`). */
  packWeek: number | null;
  needsReview: boolean;
}

export interface MissedGuide {
  /** The «چند قرص جا افتاده؟» chips, by count. Empty → no chips (progestogen-only pill). */
  countRules: MissedPillRule[];
  /** Steps + note shown without a count (a rule with steps and no `missed`). */
  baseRule: MissedPillRule | null;
  /** Danger cards. */
  urgent: MissedPillRule[];
}

/** The rules of the user's pill, split into count chips, the count-free rule and the danger cards. */
export function missedGuide(rules: readonly MissedPillRule[], method: ContraceptionMethodCode): MissedGuide {
  const mine = rules.filter((r) => !r.methods || r.methods.length === 0 || r.methods.includes(method));
  const countRules = mine
    .filter((r) => r.missed !== null && r.severity !== 'urgent')
    .sort((a, b) => (a.missed ?? 0) - (b.missed ?? 0));
  const baseRule = mine.find((r) => r.missed === null && r.severity !== 'urgent' && r.steps.length > 0) ?? null;
  const urgent = mine.filter((r) => r.severity === 'urgent' && (r.title || r.body));
  return { countRules, baseRule, urgent };
}

/** The care reminder a method created for `kind`, if the user still has it. */
export function methodReminder(
  reminders: readonly MethodReminder[],
  kind: MethodReminderKind,
): MethodReminder | null {
  return reminders.find((r) => r.kind === kind) ?? null;
}

/** Whole days from `today` to `date` (both `Y-m-d`); negative when past. */
export function daysUntil(date: string, today: string): number {
  return diffInDays(fromApiDate(date.slice(0, 10)), fromApiDate(today.slice(0, 10)));
}

/** Days before the next injection when «تزریق را زدم» appears (and any time after). */
export const INJECTION_DONE_WINDOW = 21;
/** The usual injection interval in weeks (the server plans +12 w). */
export const INJECTION_WEEKS = 12;
