import type { FactDef, FactType } from '../api/facts';

/** Condition editor state (strings for every input) and its mapping to the stored groups / exceptions. */

export interface Condition {
  fact: string;
  op: string;
  value?: unknown;
}
export interface Group {
  op: 'and' | 'or';
  items: Condition[];
}
export interface CondForm {
  fact: string;
  op: string;
  /** Text of the input; lists and intervals are comma-separated («a, b» / «1, 4»). */
  value: string;
}
/**
 * The three sections of the builder. Stored: `allOf` = the `and` group, each `anyOf` entry = one `or` group (groups
 * are joined by AND), `exclude` = the exceptions (the message is withheld when any of them holds).
 */
export interface ConditionsForm {
  allOf: CondForm[];
  anyOf: CondForm[][];
  exclude: CondForm[];
}

export const emptyCond = (): CondForm => ({ fact: '', op: 'eq', value: '' });
export const emptyConditions = (): ConditionsForm => ({ allOf: [emptyCond()], anyOf: [], exclude: [] });

/** The operators valid for a fact type (mirrors backend rules.CheckFact). */
export function opsFor(type: FactType | undefined): readonly string[] {
  switch (type) {
    case 'int':
    case 'float':
      return ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'between'];
    case 'bool':
    case 'date':
      return ['eq', 'ne'];
    case 'enum':
      return ['eq', 'ne', 'in', 'nin'];
    case 'set':
      return ['contains', 'in', 'nin'];
    default:
      return ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'in', 'nin', 'between', 'contains'];
  }
}

/** Which input a (type, operator) pair needs. */
export type InputKind = 'number' | 'interval' | 'bool' | 'date' | 'enum-single' | 'enum-multi' | 'text';
export function inputKind(def: FactDef | undefined, op: string): InputKind {
  if (!def) return 'text';
  if (def.type === 'bool') return 'bool';
  if (def.type === 'date') return 'date';
  if (def.type === 'int' || def.type === 'float') return op === 'between' ? 'interval' : 'number';
  if (def.enum && def.enum.length > 0) return op === 'in' || op === 'nin' ? 'enum-multi' : 'enum-single';
  return 'text';
}

const scalar = (s: string): unknown => {
  const v = s.trim();
  if (v === 'true') return true;
  if (v === 'false') return false;
  return v !== '' && Number.isFinite(Number(v)) ? Number(v) : v;
};
const list = (t: string): string[] => (t.trim() === '' ? [] : t.split(/[,،]/).map((s) => s.trim()).filter(Boolean));

/** Typed value of an input: numbers for numeric facts, booleans for bool, strings for enum / set / date. */
export function parseValue(def: FactDef | undefined, op: string, text: string): unknown {
  const t = text.trim();
  if (def) {
    const numeric = def.type === 'int' || def.type === 'float';
    if (op === 'between') return list(t).map((s) => (Number.isFinite(Number(s)) ? Number(s) : s));
    if (op === 'in' || op === 'nin') return list(t);
    if (numeric) return t !== '' && Number.isFinite(Number(t)) ? Number(t) : t;
    if (def.type === 'bool') return t === 'true' ? true : t === 'false' ? false : t;
    return t;
  }
  if (op === 'in' || op === 'nin' || op === 'between') return list(t).map(scalar);
  return scalar(t);
}

export function formatValue(value: unknown): string {
  if (Array.isArray(value)) return value.map(String).join(', ');
  return value === undefined || value === null ? '' : String(value);
}

const find = (facts: readonly FactDef[], name: string) => facts.find((f) => f.name === name);

const condOut = (facts: readonly FactDef[]) => (c: CondForm): Condition => ({
  fact: c.fact.trim(),
  op: c.op,
  value: parseValue(find(facts, c.fact.trim()), c.op, c.value),
});
const condIn = (c: Condition): CondForm => ({ fact: c.fact, op: c.op, value: formatValue(c.value) });
const filled = (c: CondForm) => c.fact.trim() !== '';

/** Form → stored `conditions` (groups) and `exceptions`. Blank rows are dropped. */
export function toStored(f: ConditionsForm, facts: readonly FactDef[]): { conditions: Group[]; exceptions: Condition[] } {
  const out = condOut(facts);
  const groups: Group[] = [];
  const all = f.allOf.filter(filled).map(out);
  if (all.length > 0) groups.push({ op: 'and', items: all });
  for (const g of f.anyOf) {
    const items = g.filter(filled).map(out);
    if (items.length > 0) groups.push({ op: 'or', items });
  }
  return { conditions: groups, exceptions: f.exclude.filter(filled).map(out) };
}

/** Stored → form. All `and` groups merge into one «all of» (AND of ANDs is unchanged); each `or` group stays one. */
export function fromStored(groups: readonly Group[], exceptions: readonly Condition[]): ConditionsForm {
  const allOf = groups.filter((g) => g.op === 'and').flatMap((g) => g.items.map(condIn));
  const anyOf = groups.filter((g) => g.op === 'or').map((g) => g.items.map(condIn));
  return {
    allOf: allOf.length === 0 && anyOf.length === 0 ? [emptyCond()] : allOf,
    anyOf,
    exclude: exceptions.map(condIn),
  };
}

/** Every fact name a form reads (conditions and exceptions). */
export function factsUsed(f: ConditionsForm): string[] {
  return [...f.allOf, ...f.anyOf.flat(), ...f.exclude].filter(filled).map((c) => c.fact.trim());
}

/** Whether the form reads a fertility fact (then the `fertile_window` topic is required, M-06). */
export function usesFertilityFact(f: ConditionsForm, fertilityFacts: readonly string[]): boolean {
  return factsUsed(f).some((n) => fertilityFacts.includes(n));
}

/** Pick a value-compatible default when the fact (and so the valid operators) changes. */
export function onFactChange(c: CondForm, fact: string, facts: readonly FactDef[]): CondForm {
  const def = find(facts, fact);
  const ops = opsFor(def?.type);
  const op = ops.includes(c.op) ? c.op : (ops[0] ?? 'eq');
  return { fact, op, value: ops.includes(c.op) && def?.type === find(facts, c.fact)?.type ? c.value : '' };
}

/** One-line summary («mode = pregnancy ∧ days_since_weight ≥ 7»). */
export const OP_SYMBOL: Record<string, string> = { eq: '=', ne: '≠', gt: '>', gte: '≥', lt: '<', lte: '≤', in: '∈', nin: '∉', between: '…', contains: '∋' };
export function summarize(groups: readonly Group[]): string {
  return groups
    .map((g) => g.items.map((c) => `${c.fact} ${OP_SYMBOL[c.op] ?? c.op} ${formatValue(c.value)}`).join(g.op === 'or' ? ' ∨ ' : ' ∧ '))
    .join(' ∧ ');
}
