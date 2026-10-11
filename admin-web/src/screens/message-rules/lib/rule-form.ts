import type { Condition, Group, Rule } from '../api/rules';

/** Editor state of one message rule (strings for every input) and its mapping to / from the API payload. */

export interface CondForm {
  fact: string;
  op: string;
  value: string;
}
export interface GroupForm {
  op: 'and' | 'or';
  items: CondForm[];
}
export interface VariantForm {
  title: string;
  body: string;
  cta_label: string;
  cta_target: string;
}
export interface RuleForm {
  title: string;
  level: string;
  kind: string;
  priority: string;
  placement: string;
  a: VariantForm;
  b: VariantForm;
  variants_enabled: boolean;
  ab_split: string;
  seen: string;
  confidence: string;
  contact: string;
  groups: GroupForm[];
  exceptions: CondForm[];
  cap_count: string;
  cap_period: string;
  cooldown_days: string;
  hours_from: string;
  hours_to: string;
  starts_on: string;
  ends_on: string;
  audience_percent: string;
  goal: string;
}

/** Priority bands (backend rules.Band) — used for the default and the hint. */
export const BANDS: Record<string, { min: number; max: number }> = {
  safety: { min: 1000, max: 1000 },
  engine: { min: 800, max: 999 },
  clinical: { min: 100, max: 799 },
  product: { min: 1, max: 99 },
};

export function kindOfPriority(p: number): string | null {
  return Object.entries(BANDS).find(([, b]) => p >= b.min && p <= b.max)?.[0] ?? null;
}

/** Medical messages (clinical kinds, the two follow-up levels) need the medical advisor's approval. */
export function needsApproval(level: string, kind: string): boolean {
  return level === 'worth_followup' || level === 'follow_soon' || kind === 'safety' || kind === 'clinical';
}

export const emptyVariant = (): VariantForm => ({ title: '', body: '', cta_label: '', cta_target: '' });

export function emptyForm(): RuleForm {
  return {
    title: '',
    level: 'info',
    kind: 'product',
    priority: '30',
    placement: 'smart_tip',
    a: emptyVariant(),
    b: emptyVariant(),
    variants_enabled: false,
    ab_split: '50',
    seen: '',
    confidence: '',
    contact: '',
    groups: [{ op: 'and', items: [{ fact: '', op: 'eq', value: '' }] }],
    exceptions: [],
    cap_count: '1',
    cap_period: 'day',
    cooldown_days: '0',
    hours_from: '',
    hours_to: '',
    starts_on: '',
    ends_on: '',
    audience_percent: '100',
    goal: '',
  };
}

/** `1, 4` → [1, 4]; `cycle, ttc` → ['cycle','ttc'] for list ops; true/false/numbers for the rest. */
export function parseValue(op: string, text: string): unknown {
  const t = text.trim();
  const scalar = (s: string): unknown => {
    const v = s.trim();
    if (v === 'true') return true;
    if (v === 'false') return false;
    return v !== '' && Number.isFinite(Number(v)) ? Number(v) : v;
  };
  if (op === 'in' || op === 'nin' || op === 'between') {
    return t === '' ? [] : t.split(/[,،]/).map(scalar);
  }
  return scalar(t);
}

export function formatValue(value: unknown): string {
  if (Array.isArray(value)) return value.map(String).join(', ');
  return value === undefined || value === null ? '' : String(value);
}

const condOut = (c: CondForm): Condition => ({ fact: c.fact.trim(), op: c.op, value: parseValue(c.op, c.value) });
const condIn = (c: Condition): CondForm => ({ fact: c.fact, op: c.op, value: formatValue(c.value) });
const intOrNull = (s: string): number | null => (s.trim() === '' ? null : Number(s));

function variantOut(v: VariantForm) {
  return { title: v.title, body: v.body, cta_label: v.cta_label, cta_target: v.cta_target };
}

export function toPayload(f: RuleForm): Record<string, unknown> {
  const content: Record<string, unknown> = { a: variantOut(f.a) };
  if (f.variants_enabled) content.b = variantOut(f.b);
  if (f.level === 'worth_followup' || f.level === 'follow_soon') {
    content.seen = f.seen;
    content.confidence = f.confidence;
  }
  if (f.level === 'follow_soon') content.contact = f.contact;
  const cap = intOrNull(f.cap_count);
  return {
    title: f.title.trim(),
    level: f.level,
    kind: f.kind,
    priority: Number(f.priority),
    placement: f.placement,
    content,
    variants_enabled: f.variants_enabled,
    ab_split: f.variants_enabled ? Number(f.ab_split) : null,
    conditions: f.groups
      .map((g) => ({ op: g.op, items: g.items.filter((c) => c.fact.trim() !== '').map(condOut) }))
      .filter((g) => g.items.length > 0),
    exceptions: f.exceptions.filter((c) => c.fact.trim() !== '').map(condOut),
    cap_count: cap,
    cap_period: cap === null ? '' : f.cap_period,
    cooldown_days: Number(f.cooldown_days || 0),
    hours_from: intOrNull(f.hours_from),
    hours_to: intOrNull(f.hours_to),
    starts_on: f.starts_on,
    ends_on: f.ends_on,
    audience_percent: intOrNull(f.audience_percent) ?? 100,
    goal: f.goal.trim(),
  };
}

export function fromRule(r: Rule): RuleForm {
  const b = r.content.b;
  return {
    title: r.title,
    level: r.level,
    kind: r.kind,
    priority: String(r.priority),
    placement: r.placement,
    a: { ...emptyVariant(), ...r.content.a },
    b: { ...emptyVariant(), ...(b ?? {}) },
    variants_enabled: r.variants_enabled,
    ab_split: String(r.ab_split),
    seen: r.content.seen ?? '',
    confidence: r.content.confidence ?? '',
    contact: r.content.contact ?? '',
    groups: r.conditions.length
      ? r.conditions.map((g: Group) => ({ op: g.op, items: g.items.map(condIn) }))
      : [{ op: 'and', items: [{ fact: '', op: 'eq', value: '' }] }],
    exceptions: r.exceptions.map(condIn),
    cap_count: r.cap_count === null ? '' : String(r.cap_count),
    cap_period: r.cap_period ?? 'day',
    cooldown_days: String(r.cooldown_days),
    hours_from: r.hours_from === null ? '' : String(r.hours_from),
    hours_to: r.hours_to === null ? '' : String(r.hours_to),
    starts_on: r.starts_on ?? '',
    ends_on: r.ends_on ?? '',
    audience_percent: String(r.audience_percent),
    goal: r.goal ?? '',
  };
}

/** One-line rule summary for the list («mode = pregnancy ∧ days_since_weight ≥ 7»). */
const OP_SYMBOL: Record<string, string> = { eq: '=', ne: '≠', gt: '>', gte: '≥', lt: '<', lte: '≤', in: '∈', nin: '∉', between: '…', contains: '∋' };
export function summarize(groups: Group[]): string {
  return groups
    .map((g) => g.items.map((c) => `${c.fact} ${OP_SYMBOL[c.op] ?? c.op} ${formatValue(c.value)}`).join(g.op === 'or' ? ' ∨ ' : ' ∧ '))
    .join(' ∧ ');
}
