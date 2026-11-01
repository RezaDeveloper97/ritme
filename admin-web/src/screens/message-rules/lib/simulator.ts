import type { SimSlot, SimulationV6 } from '../api/rules';

/** Drop reasons the simulator can answer (rules.SimulateSlot); the i18n keys live in `messageSim.reasons`. */
export const SIM_REASONS = [
  'inactive',
  'not_approved',
  'kind_not_allowed',
  'schedule',
  'hours',
  'conditions',
  'exclude',
  'audience',
  'requires_user_input',
  'low_confidence',
  'contraception',
  'post_safety_24h',
  'quiet_hours',
  'cap_day',
  'cap_week',
  'cap_cycle',
  'cooldown',
  'slot_cap',
  'capacity',
  'missing_variable',
] as const;

/** i18n key of a reason; a reason a newer server adds falls back to `unknown` instead of a missing-key error. */
export function reasonKey(reason: string): string {
  return (SIM_REASONS as readonly string[]).includes(reason) ? reason : 'unknown';
}

export interface SimRow {
  slot: string;
  capacity: number | null;
  winners: { ruleId: number; title: string; variant: string; version: number | null; messageTitle: string; messageBody: string; level: string }[];
  dropped: { ruleId: number; title: string; reasonKey: string; reason: string; kind: string }[];
}

/** The slots that have something to say first (winners, then drops); the rest keep the server order. */
export function toRows(sim: SimulationV6): SimRow[] {
  const rows = sim.slots.map((s: SimSlot) => ({
    slot: s.slot,
    capacity: s.capacity,
    winners: s.winners.map((w) => ({
      ruleId: w.rule_id,
      title: w.title,
      variant: w.variant,
      version: w.version,
      messageTitle: w.message_title,
      messageBody: w.message_body,
      level: w.level,
    })),
    dropped: s.dropped.map((d) => ({ ruleId: d.rule_id, title: d.title, reasonKey: reasonKey(d.reason), reason: d.reason, kind: d.kind })),
  }));
  return rows.sort((a, b) => Number(b.winners.length > 0) - Number(a.winners.length > 0));
}

/** The engine summary as label/value pairs, empty values dropped (arrays joined). */
export function engineSummary(engine: Record<string, unknown>): { key: string; value: string }[] {
  return Object.entries(engine)
    .filter(([, v]) => v !== null && v !== undefined && v !== '' && !(Array.isArray(v) && v.length === 0))
    .map(([key, v]) => ({ key, value: Array.isArray(v) ? v.join(', ') : String(v) }));
}

/** A `{user_id}` target from the input: digits are an id, an `RT-…` code needs resolving. */
export function parseUserRef(input: string): { kind: 'none' } | { kind: 'id'; id: number } | { kind: 'code'; code: string } | { kind: 'invalid' } {
  const v = input.trim();
  if (v === '') return { kind: 'none' };
  if (/^\d{1,12}$/.test(v)) return { kind: 'id', id: Number(v) };
  if (/^RT-[A-Za-z0-9]{6,8}$/i.test(v)) return { kind: 'code', code: v.toUpperCase() };
  return { kind: 'invalid' };
}
