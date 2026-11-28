import type { AtlasCondition, AtlasRule } from '../api/atlas';

/**
 * «نقشهٔ پیام‌ها»: every message / call to action the app can show, by WHO sees it (life mode) and WHERE (surface),
 * with WHEN in plain words. Step 1 covers the two structured sources: the engine's rules (conditions are data) and
 * the `/messages` copy groups (conditions live in backend code, described once in `messageAtlas.when.<group>`).
 * Catalog- and code-driven messages join in later steps; the consolidation plan is to move layer 2 (the message
 * decision) into the engine, after which this page is simply the engine's full view.
 */
export const MODES = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen', 'companion'] as const;
export type Mode = (typeof MODES)[number];

/** Where in the app, in display order. */
export const SURFACES = [
  'urgentModal',
  'homeBanner',
  'smartTip',
  'dailyAdvice',
  'dailyChallenge',
  'homeCard',
  'modal',
  'inbox',
  'push',
  'sms',
  'pregnancyHome',
  'pregnancySetup',
  'postpartumHome',
  'postpartumMood',
  'menopauseHome',
  'companionHome',
  'analysis',
  'other',
  // Copy groups whose messages moved into the engine (rules tip.*, D-216): kept editable, shown nowhere.
  'retired',
] as const;
export type Surface = (typeof SURFACES)[number];

export type Source = 'rule' | 'group';

export interface AtlasEntry {
  id: string;
  source: Source;
  /** Empty = every mode. */
  modes: Mode[];
  surface: Surface;
  /** Rule: its internal title; group: the group key (labelled by `smartMessages.groups`). */
  name: string;
  /** Stable key shown small: rule_key or the group key. */
  key: string | null;
  href: string;
  rule?: AtlasRule;
  group?: string;
}

// ---------------------------------------------------------------------------
// Rules

const PLACEMENT_SURFACE: Record<string, Surface> = {
  status_banner: 'homeBanner',
  smart_tip: 'smartTip',
  daily_advice: 'dailyAdvice',
  daily_challenge: 'dailyChallenge',
  modal: 'modal',
  urgent_modal: 'urgentModal',
  alerts_inbox: 'inbox',
  alerts: 'inbox',
  push: 'push',
  sms: 'sms',
};

export function surfaceOfPlacement(placement: string): Surface {
  return PLACEMENT_SURFACE[placement] ?? 'other';
}

/** The fact groups (GET /messages/facts) that only exist in one life mode. `engine` = the cycle engine's day. */
const FACT_GROUP_MODES: Record<string, Mode[]> = {
  pregnancy: ['pregnancy'],
  pregnancy_alerts: ['pregnancy'],
  postpartum: ['postpartum'],
  menopause: ['menopause'],
  conditions: ['cycle', 'ttc'],
  contraception: ['cycle'],
};
const MODE_FACTS = new Set(['mode', 'life_mode']);
const isMode = (v: unknown): v is Mode => typeof v === 'string' && (MODES as readonly string[]).includes(v);

/**
 * Who a rule can reach: an explicit `mode` / `life_mode` condition wins; otherwise the life mode its facts belong
 * to (a pregnancy fact only exists for a pregnant user); otherwise everyone (`[]`).
 */
export function modesOfRule(rule: AtlasRule, factGroup: (fact: string) => string | undefined): Mode[] {
  const all = rule.conditions.flatMap((g) => g.items);
  const explicit = new Set<Mode>();
  for (const c of all) {
    if (!MODE_FACTS.has(c.fact)) continue;
    const values = Array.isArray(c.value) ? c.value : [c.value];
    if (c.op === 'eq' || c.op === 'in') values.filter(isMode).forEach((m) => explicit.add(m));
  }
  if (explicit.size > 0) return MODES.filter((m) => explicit.has(m));
  const implied = new Set<Mode>();
  for (const c of all) (FACT_GROUP_MODES[factGroup(c.fact) ?? ''] ?? []).forEach((m) => implied.add(m));
  return MODES.filter((m) => implied.has(m));
}

export interface ConditionWords {
  /** The fact's description in the UI language, or its name. */
  fact: (name: string) => string;
  op: (op: string) => string;
  yes: string;
  no: string;
  listJoin: string;
  and: string;
  or: string;
  /** «بین {a} و {b}» */
  between: (a: string, b: string) => string;
  num: (n: number) => string;
}

function valueText(value: unknown, w: ConditionWords): string {
  if (typeof value === 'boolean') return value ? w.yes : w.no;
  if (typeof value === 'number') return w.num(value);
  if (Array.isArray(value)) return value.map((v) => valueText(v, w)).join(w.listJoin);
  return value === null || value === undefined ? '' : String(value);
}

/** One condition in plain words: «تعداد روزهای تأخیر ≥ ۵». A true / false check reads as the fact (or «نه: fact»). */
export function conditionText(c: AtlasCondition, w: ConditionWords): string {
  const fact = w.fact(c.fact);
  if (c.op === 'between' && Array.isArray(c.value) && c.value.length === 2) {
    return `${fact} ${w.between(valueText(c.value[0], w), valueText(c.value[1], w))}`;
  }
  if (typeof c.value === 'boolean' && (c.op === 'eq' || c.op === 'ne')) {
    const holds = c.op === 'eq' ? c.value : !c.value;
    return holds ? fact : `${w.no}: ${fact}`;
  }
  return `${fact} ${w.op(c.op)} ${valueText(c.value, w)}`.trim();
}

/** The rule's «when»: its groups (joined by AND; inside a group by its own AND / OR) and its exceptions. */
export function whenOfRule(rule: AtlasRule, w: ConditionWords): { all: string[]; except: string[] } {
  const all = rule.conditions
    .filter((g) => g.items.length > 0)
    .map((g) => {
      const parts = g.items.map((c) => conditionText(c, w));
      const joined = parts.join(g.op === 'or' ? w.or : w.and);
      return g.op === 'or' && parts.length > 1 ? `(${joined})` : joined;
    });
  return { all, except: rule.exceptions.map((c) => conditionText(c, w)) };
}

export function entryOfRule(rule: AtlasRule, factGroup: (fact: string) => string | undefined): AtlasEntry {
  return {
    id: `rule:${rule.id}`,
    source: 'rule',
    modes: modesOfRule(rule, factGroup),
    surface: surfaceOfPlacement(rule.placement),
    name: rule.title,
    key: rule.rule_key,
    href: `/messages/rules/${rule.id}`,
    rule,
  };
}

// ---------------------------------------------------------------------------
// `/messages` copy groups — who and where each group reaches (the where / when words are i18n, keyed by group).

const C: Mode[] = ['cycle'];
const T: Mode[] = ['ttc'];
const CT: Mode[] = ['cycle', 'ttc'];
const P: Mode[] = ['pregnancy'];

export const GROUP_ENTRIES: Record<string, { modes: Mode[]; surface: Surface }> = {
  cycle_base_non_ttc: { modes: C, surface: 'retired' },
  cycle_base_ttc: { modes: T, surface: 'retired' },
  cycle_override: { modes: CT, surface: 'retired' },
  correlation_cycle: { modes: CT, surface: 'retired' },
  correlation_common: { modes: P, surface: 'pregnancyHome' },
  pattern: { modes: CT, surface: 'retired' },
  nutrition_cycle: { modes: CT, surface: 'retired' },
  nutrition_ttc: { modes: T, surface: 'retired' },
  nutrition_symptom: { modes: CT, surface: 'retired' },
  sleep_cycle: { modes: CT, surface: 'retired' },
  sleep_symptom: { modes: CT, surface: 'retired' },
  exercise_cycle: { modes: CT, surface: 'retired' },
  exercise_symptom: { modes: CT, surface: 'retired' },
  exercise_ttc: { modes: T, surface: 'retired' },
  condition_nudge: { modes: CT, surface: 'homeCard' },
  pregnancy_week_tip: { modes: P, surface: 'pregnancyHome' },
  pregnancy_override: { modes: P, surface: 'pregnancyHome' },
  correlation_pregnancy: { modes: P, surface: 'pregnancyHome' },
  nutrition_trimester: { modes: P, surface: 'pregnancyHome' },
  sleep_trimester: { modes: P, surface: 'pregnancyHome' },
  exercise_trimester: { modes: P, surface: 'pregnancyHome' },
  pregnancy_trimester: { modes: P, surface: 'pregnancyHome' },
  pregnancy_week: { modes: P, surface: 'pregnancyHome' },
  pregnancy_base: { modes: P, surface: 'pregnancyHome' },
  pregnancy_alert: { modes: P, surface: 'inbox' },
  pregnancy_setup: { modes: P, surface: 'pregnancySetup' },
  postpartum_week_tip: { modes: ['postpartum'], surface: 'postpartumHome' },
  postpartum_alert: { modes: ['postpartum'], surface: 'postpartumHome' },
  postpartum_safety: { modes: ['postpartum'], surface: 'postpartumMood' },
  menopause_message: { modes: ['menopause'], surface: 'menopauseHome' },
  companion_tip: { modes: ['companion'], surface: 'companionHome' },
  bmi_message: { modes: [], surface: 'analysis' },
};

export function groupEntries(): AtlasEntry[] {
  return Object.entries(GROUP_ENTRIES).map(([group, g]) => ({
    id: `group:${group}`,
    source: 'group',
    modes: g.modes,
    surface: g.surface,
    name: group,
    key: group,
    href: `/messages?group=${encodeURIComponent(group)}`,
    group,
  }));
}

// ---------------------------------------------------------------------------
// View

export interface AtlasFilter {
  /** `all` = no mode filter; a mode keeps its own entries and the ones for everyone. */
  mode: Mode | 'all';
  source: Source | 'all';
  /** Lower-cased search text; `text(entry)` is what it is matched against. */
  q: string;
}

export function filterEntries(entries: AtlasEntry[], f: AtlasFilter, text: (e: AtlasEntry) => string): AtlasEntry[] {
  const q = f.q.trim().toLowerCase();
  return entries.filter(
    (e) =>
      (f.mode === 'all' || e.modes.length === 0 || e.modes.includes(f.mode)) &&
      (f.source === 'all' || e.source === f.source) &&
      (q === '' || text(e).toLowerCase().includes(q)),
  );
}

/** Entries bucketed by surface in SURFACES order; inside a surface rules first (by priority, high first), then groups. */
export function bySurface(entries: AtlasEntry[]): Array<{ surface: Surface; entries: AtlasEntry[] }> {
  const rank = (e: AtlasEntry) => (e.rule ? -e.rule.priority : Number.MAX_SAFE_INTEGER);
  return SURFACES.map((surface) => ({
    surface,
    entries: entries.filter((e) => e.surface === surface).sort((a, b) => rank(a) - rank(b)),
  })).filter((b) => b.entries.length > 0);
}
