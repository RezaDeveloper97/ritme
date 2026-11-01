import { emptyConditions, formatValue, parseValue, type CondForm, type ConditionsForm, type FactDef } from '@/features/rule-conditions';

import type { Translations } from '@/shared/ui';

import type { Challenge, ChallengeRule, RuleCondition } from '../api/challenges';

/** Start rules (T-M10-10) and the phases an `on_phase_start` challenge may open on (the API validates the same lists). */
export const START_RULES = ['on_join', 'on_phase_start', 'fixed_date'] as const;
export const START_PHASES = ['menstrual', 'follicular', 'fertile', 'luteal', 'period_expected'] as const;

/** The legacy `anytime` start is stored as `on_join`; `menstrual` (any day of the phase) is kept as read. */
export const normalizeStartRule = (r: string | null | undefined): string => (!r || r === 'anytime' ? 'on_join' : r);

const condIn = (c: RuleCondition): CondForm => ({ fact: c.fact, op: c.op, value: formatValue(c.value) });

/** Stored rule → builder state. A stored rule that cannot be read comes back flagged so the editor can warn. */
export function ruleToForm(rule: ChallengeRule | undefined): { form: ConditionsForm; invalid: boolean } {
  if (!rule) return { form: { ...emptyConditions(), allOf: [] }, invalid: false };
  return {
    form: { allOf: rule.all_of.map(condIn), anyOf: rule.any_of.map((g) => g.map(condIn)), exclude: rule.exclude.map(condIn) },
    invalid: rule.invalid === true,
  };
}

/** Builder state → the write shape; blank rows dropped. An empty rule is `{}` (the API reads it as «no rule»). */
export function formToRule(f: ConditionsForm, facts: readonly FactDef[]): Record<string, unknown> {
  const out = (c: CondForm) => ({ fact: c.fact.trim(), op: c.op, value: parseValue(facts.find((x) => x.name === c.fact.trim()), c.op, c.value) });
  const filled = (c: CondForm) => c.fact.trim() !== '';
  const rule: Record<string, unknown> = {};
  const all = f.allOf.filter(filled).map(out);
  const any = f.anyOf.map((g) => g.filter(filled).map(out)).filter((g) => g.length > 0);
  const exclude = f.exclude.filter(filled).map(out);
  if (all.length > 0) rule.all_of = all;
  if (any.length > 0) rule.any_of = any;
  if (exclude.length > 0) rule.exclude = exclude;
  return rule;
}

export interface TaskDraft {
  /** Stable React key (the server id when saved, else a counter). */
  key: string;
  id: number | null;
  dayFrom: string;
  dayTo: string;
  title: Translations;
}

export interface ProgramDraft {
  enabled: boolean;
  duration: string;
  startRule: string;
  startsOn: string;
  reminder: string;
  startPhase: string;
  eligibility: ConditionsForm;
  contraindications: ConditionsForm;
  /** A stored rule the API could not parse: saving would replace it, so the editor warns first. */
  eligibilityInvalid: boolean;
  contraindicationsInvalid: boolean;
  badgeCode: string;
  badgeTitle: Translations;
  plusDays: string;
  minDays: string;
  tasks: TaskDraft[];
}

let counter = 0;
export const newTask = (dayFrom = '1', dayTo = ''): TaskDraft => ({ key: `new-${counter++}`, id: null, dayFrom, dayTo, title: {} });

export function toDraft(row: Challenge | null): ProgramDraft {
  const p = row?.program ?? null;
  return {
    enabled: p !== null,
    duration: p ? String(p.duration_days) : '7',
    startRule: normalizeStartRule(p?.start_rule),
    startPhase: p?.start_phase ?? '',
    ...(() => {
      const e = ruleToForm(p?.eligibility);
      const c = ruleToForm(p?.contraindications);
      return { eligibility: e.form, eligibilityInvalid: e.invalid, contraindications: c.form, contraindicationsInvalid: c.invalid };
    })(),
    startsOn: p?.starts_on ?? '',
    reminder: p?.reminder_time ?? '',
    badgeCode: p?.reward_badge_code ?? '',
    badgeTitle: p?.reward_badge_title ?? {},
    plusDays: String(p?.reward_plus_days ?? 0),
    minDays: p?.completion_min_days != null ? String(p.completion_min_days) : '',
    tasks: (row?.tasks ?? []).map((t) => ({ key: String(t.id), id: t.id, dayFrom: String(t.day_from), dayTo: String(t.day_to), title: t.title })),
  };
}

const int = (s: string): number | null => (s.trim() === '' || Number.isNaN(Number(s)) ? null : Math.trunc(Number(s)));

/** An unreadable stored contraindication refuses everyone: saving it untouched keeps it (the key is omitted). */
function contraindicationsField(d: ProgramDraft, facts: readonly FactDef[]): Record<string, unknown> {
  const rule = formToRule(d.contraindications, facts);
  return d.contraindicationsInvalid && Object.keys(rule).length === 0 ? {} : { contraindications: rule };
}

/** The v2 part of the write body — empty when the challenge stays a legacy one-day pick. */
export function toPayload(d: ProgramDraft, facts: readonly FactDef[] = []): Record<string, unknown> {
  if (!d.enabled) return {};
  return {
    duration_days: int(d.duration),
    start_rule: d.startRule,
    starts_on: d.startRule === 'fixed_date' && d.startsOn ? d.startsOn : null,
    reminder_time: d.reminder || null,
    start_phase: d.startRule === 'on_phase_start' && d.startPhase ? d.startPhase : null,
    eligibility: formToRule(d.eligibility, facts),
    ...contraindicationsField(d, facts),
    reward_badge_code: d.badgeCode.trim() || null,
    reward_badge_title: d.badgeTitle,
    reward_plus_days: int(d.plusDays) ?? 0,
    completion_min_days: int(d.minDays),
    tasks: d.tasks.map((t, i) => ({
      ...(t.id !== null ? { id: t.id } : {}),
      day_from: int(t.dayFrom),
      day_to: int(t.dayTo) ?? int(t.dayFrom),
      title: t.title,
      sort_order: i,
    })),
  };
}

/** Day numbers 1..duration, for the per-day task overview. */
export function daysOf(duration: string): number[] {
  const n = int(duration);
  return n !== null && n >= 1 && n <= 366 ? Array.from({ length: n }, (_, i) => i + 1) : [];
}

/** The tasks that apply to a day (1-based). */
export function tasksOnDay(tasks: readonly TaskDraft[], day: number): TaskDraft[] {
  return tasks.filter((t) => {
    const from = int(t.dayFrom);
    const to = int(t.dayTo) ?? from;
    return from !== null && to !== null && from <= day && day <= to;
  });
}
