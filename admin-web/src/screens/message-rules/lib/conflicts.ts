import type { Conflict, FactRange } from '../api/rules';

/** One overlap fact as text: «cycle_day 5–9», «mode ∈ a, b», «tags ∋ x». */
export function describeRange(r: FactRange): string {
  const not = r.not.length > 0 ? ` ∖ ${r.not.join(', ')}` : '';
  if (r.kind === 'number') {
    const lo = r.min === null ? '' : `${r.min_open ? '>' : '≥'} ${r.min}`;
    const hi = r.max === null ? '' : `${r.max_open ? '<' : '≤'} ${r.max}`;
    if (!r.min_open && !r.max_open && r.min !== null && r.min === r.max) return `${r.fact} = ${r.min}${not}`;
    return `${r.fact} ${[lo, hi].filter(Boolean).join(' ') || '∈ ℝ'}${not}`;
  }
  if (r.kind === 'set') {
    const parts = [
      ...(r.contains ?? []).map((m) => `∋ ${m}`),
      ...(r.any_of ?? []).map((l) => `∩ {${l.join(', ')}}`),
    ];
    return `${r.fact} ${parts.join(' ') || '∈ *'}${not}`;
  }
  return `${r.fact} ${r.values ? `∈ ${r.values.join(', ')}` : '∈ *'}${not}`;
}

export interface ConflictView {
  ruleId: number;
  title: string;
  priority: number;
  safety: boolean;
  slot: string | null;
  winner: Conflict['winner'];
  /** The winner is picked by recency, not by safety or priority. */
  tieBreak: boolean;
  /** Both can be shown together (list slot or capacity > 1): no real conflict. */
  coShown: boolean;
  facts: string[];
  /** `from → to` (open ends dropped), null when both are open. */
  schedule: string | null;
  hours: string | null;
  audience: number | null;
}

const hhmm = (h: number) => `${String(h).padStart(2, '0')}:00`;

export function toConflictView(c: Conflict): ConflictView {
  const o = c.overlap;
  const schedule = o && (o.starts_at || o.ends_at) ? `${o.starts_at ?? '…'} → ${o.ends_at ?? '…'}` : null;
  return {
    ruleId: c.rule_id,
    title: c.title,
    priority: c.priority,
    safety: c.safety,
    slot: c.slot,
    winner: c.winner,
    tieBreak: c.tie_break,
    coShown: c.co_shown,
    facts: (o?.facts ?? []).map(describeRange),
    schedule,
    hours: o?.hours && o.hours.length > 0 ? o.hours.map(([a, b]) => `${hhmm(a)}–${hhmm(b)}`).join(', ') : null,
    audience: o?.audience_percent ?? null,
  };
}
