import { fromStored, emptyConditions, summarize, toStored, usesFertilityFact, type ConditionsForm, type FactDef } from '@/features/rule-conditions';

import type { Rule } from '../api/rules';

export { summarize };

/**
 * Editor state of one message rule (strings for every input) and its mapping to / from the API payload.
 * A legacy rule (a/b split, one cap, one hour window, dates) round-trips unchanged unless the editor adds what only
 * the v6 columns can hold (a third variant, a second cap, a second hour window…).
 */

export interface VariantRow {
  key: string;
  weight: string;
  title: string;
  body: string;
  cta_label: string;
  cta_target: string;
}
export interface CapRow {
  count: string;
  period: string;
}
export interface HourRow {
  from: string;
  to: string;
}
export interface RuleForm {
  title: string;
  rule_key: string;
  level: string;
  kind: string;
  priority: string;
  placement: string;
  variants: VariantRow[];
  /** The rule is stored with the v6 `variants` column (else the legacy a/b split). */
  v6Variants: boolean;
  seen: string;
  confidence: string;
  contact: string;
  conditions: ConditionsForm;
  caps: CapRow[];
  v6Caps: boolean;
  cooldown_days: string;
  hours: HourRow[];
  v6Hours: boolean;
  /** `YYYY-MM-DDTHH:mm` (Tehran wall clock, the datetime-local format). */
  starts_at: string;
  ends_at: string;
  /** Legacy date window, passed through while starts_at / ends_at stay empty. */
  starts_on: string;
  ends_on: string;
  audience_percent: string;
  goal: string;
  goal_event: string;
  topics: string[];
}

/** Priority bands (backend rules.Band) — used for the default and the hint. */
export const BANDS: Record<string, { min: number; max: number }> = {
  safety: { min: 1000, max: 1000 },
  engine: { min: 800, max: 999 },
  clinical: { min: 100, max: 799 },
  content: { min: 100, max: 799 },
  product: { min: 1, max: 99 },
  marketing: { min: 1, max: 99 },
};

export const FERTILE_TOPIC = 'fertile_window';
export const MAX_VARIANT_KEYS = ['a', 'b', 'c', 'd', 'e', 'f'];

export function kindOfPriority(p: number): string | null {
  return Object.entries(BANDS).find(([, b]) => p >= b.min && p <= b.max)?.[0] ?? null;
}

/** Medical messages (clinical kinds, the two follow-up levels) need the medical advisor's approval. */
export function needsApproval(level: string, kind: string): boolean {
  return level === 'worth_followup' || level === 'follow_soon' || kind === 'safety' || kind === 'clinical';
}

export const emptyRow = (key: string, weight = '50'): VariantRow => ({ key, weight, title: '', body: '', cta_label: '', cta_target: '' });

export function emptyForm(): RuleForm {
  return {
    title: '',
    rule_key: '',
    level: 'info',
    kind: 'product',
    priority: '30',
    placement: 'push',
    variants: [emptyRow('a', '100')],
    v6Variants: false,
    seen: '',
    confidence: '',
    contact: '',
    conditions: emptyConditions(),
    caps: [{ count: '1', period: 'day' }],
    v6Caps: false,
    cooldown_days: '0',
    hours: [],
    v6Hours: false,
    starts_at: '',
    ends_at: '',
    starts_on: '',
    ends_on: '',
    audience_percent: '100',
    goal: '',
    goal_event: '',
    topics: [],
  };
}

const intOrNull = (s: string): number | null => (s.trim() === '' ? null : Number(s));
const isMedicalLevel = (l: string) => l === 'worth_followup' || l === 'follow_soon';

/** Datetime-local ↔ the API's `YYYY-MM-DD HH:mm:ss`. */
export const scheduleIn = (s: string | null): string => (s ? s.replace(' ', 'T').slice(0, 16) : '');
export const scheduleOut = (s: string): string => (s ? `${s.replace('T', ' ')}${s.length === 16 ? ':00' : ''}` : '');

export function fromRule(r: Rule): RuleForm {
  const v6Variants = (r.variants?.length ?? 0) > 0;
  const text = (key: string) => (key === 'a' ? r.content.a : key === 'b' ? r.content.b : r.content.variants[key]) ?? undefined;
  const rows: VariantRow[] = v6Variants
    ? (r.variants ?? []).map((v) => ({ ...emptyRow(v.key, String(v.weight)), ...text(v.key) }))
    : r.variants_enabled && r.content.b
      ? [
          { ...emptyRow('a', String(r.ab_split)), ...r.content.a },
          { ...emptyRow('b', String(100 - r.ab_split)), ...r.content.b },
        ]
      : [{ ...emptyRow('a', '100'), ...r.content.a }];
  const v6Caps = r.caps !== null;
  const caps: CapRow[] = v6Caps
    ? (r.caps ?? []).map((c) => ({ count: String(c.count), period: c.period }))
    : r.cap_count === null
      ? []
      : [{ count: String(r.cap_count), period: r.cap_period ?? 'day' }];
  const v6Hours = r.hour_windows !== null;
  const hours: HourRow[] = v6Hours
    ? (r.hour_windows ?? []).map(([from, to]) => ({ from: String(from), to: String(to) }))
    : r.hours_from !== null && r.hours_to !== null
      ? [{ from: String(r.hours_from), to: String(r.hours_to) }]
      : [];
  return {
    title: r.title,
    rule_key: r.rule_key ?? '',
    level: r.level,
    kind: r.kind,
    priority: String(r.priority),
    placement: r.placement,
    variants: rows,
    v6Variants,
    seen: r.content.seen ?? '',
    confidence: r.content.confidence ?? '',
    contact: r.content.contact ?? '',
    conditions: fromStored(r.conditions, r.exceptions),
    caps,
    v6Caps,
    cooldown_days: String(r.cooldown_days),
    hours,
    v6Hours,
    starts_at: scheduleIn(r.starts_at),
    ends_at: scheduleIn(r.ends_at),
    starts_on: r.starts_on ?? '',
    ends_on: r.ends_on ?? '',
    audience_percent: String(r.audience_percent),
    goal: r.goal ?? '',
    goal_event: r.goal_event ?? '',
    topics: r.topics ?? [],
  };
}

/** Sum of the variant weights (shown next to the variants). */
export const weightSum = (rows: readonly VariantRow[]): number => rows.reduce((n, r) => n + (Number(r.weight) || 0), 0);

/** A legacy rule keeps the a/b columns while it is still `a` or `a`+`b` with weights adding up to 100. */
function legacyVariantsFit(f: RuleForm): boolean {
  if (f.v6Variants) return false;
  const [a, b, ...rest] = f.variants;
  if (!a || rest.length > 0 || a.key !== 'a') return false;
  if (!b) return true;
  return b.key === 'b' && Number(a.weight) + Number(b.weight) === 100 && Number(a.weight) >= 1 && Number(b.weight) >= 1;
}

/** Topics sent: the fertility topic is added whenever a fertility fact is read (M-06). */
export function topicsOut(f: RuleForm, facts: readonly FactDef[], fertilityFacts: readonly string[]): string[] {
  void facts;
  return usesFertilityFact(f.conditions, fertilityFacts) && !f.topics.includes(FERTILE_TOPIC) ? [...f.topics, FERTILE_TOPIC] : f.topics;
}

const variantOut = (v: VariantRow) => ({ title: v.title, body: v.body, cta_label: v.cta_label, cta_target: v.cta_target });

export function toPayload(f: RuleForm, facts: readonly FactDef[], fertilityFacts: readonly string[] = []): Record<string, unknown> {
  const safety = f.kind === 'safety';
  const content: Record<string, unknown> = {};
  const out: Record<string, unknown> = {};
  const [a, b] = f.variants;
  if (a) content.a = variantOut(a);
  if (legacyVariantsFit(f)) {
    if (b) content.b = variantOut(b);
    out.variants_enabled = f.variants.length > 1;
    out.ab_split = b ? Number(a?.weight) : null;
  } else {
    const extra: Record<string, unknown> = {};
    for (const v of f.variants.slice(1)) {
      if (v.key === 'b') content.b = variantOut(v);
      else extra[v.key] = variantOut(v);
    }
    if (Object.keys(extra).length > 0) content.variants = extra;
    out.variants = f.variants.map((v) => ({ key: v.key, weight: Number(v.weight) }));
    out.variants_enabled = f.variants.length > 1;
    out.ab_split = null;
  }
  if (isMedicalLevel(f.level)) {
    content.seen = f.seen;
    content.confidence = f.confidence;
  }
  if (f.level === 'follow_soon') content.contact = f.contact;

  const caps = f.caps.filter((c) => c.count.trim() !== '');
  if (safety) {
    out.cap_count = null;
    out.cap_period = '';
  } else if (!f.v6Caps && caps.length <= 1) {
    out.cap_count = caps[0] ? Number(caps[0].count) : null;
    out.cap_period = caps[0]?.period ?? '';
  } else {
    out.caps = caps.map((c) => ({ count: Number(c.count), period: c.period }));
    out.cap_count = null;
    out.cap_period = '';
  }
  const hours = f.hours.filter((h) => h.from.trim() !== '' && h.to.trim() !== '');
  if (!f.v6Hours && hours.length <= 1) {
    out.hours_from = hours[0] ? Number(hours[0].from) : null;
    out.hours_to = hours[0] ? Number(hours[0].to) : null;
  } else {
    out.hour_windows = hours.map((h) => [Number(h.from), Number(h.to)]);
    out.hours_from = null;
    out.hours_to = null;
  }
  const stored = toStored(f.conditions, facts);
  const topics = topicsOut(f, facts, fertilityFacts);
  return {
    ...out,
    title: f.title.trim(),
    ...(f.rule_key.trim() ? { rule_key: f.rule_key.trim() } : {}),
    level: f.level,
    kind: f.kind,
    priority: Number(f.priority),
    placement: f.placement,
    content,
    conditions: stored.conditions,
    exceptions: stored.exceptions,
    cooldown_days: Number(f.cooldown_days || 0),
    starts_at: scheduleOut(f.starts_at),
    ends_at: scheduleOut(f.ends_at),
    starts_on: f.starts_at ? '' : f.starts_on,
    ends_on: f.ends_at ? '' : f.ends_on,
    audience_percent: intOrNull(f.audience_percent) ?? 100,
    goal: f.goal.trim(),
    goal_event: f.goal_event,
    ...(topics.length > 0 ? { topics } : {}),
  };
}
