/**
 * Where each message group shows up in the user app, so an editor knows what a row changes before editing it.
 * Traced from the consumers (backend-go/internal/messages/manager, pregnancy/v2, postpartum/guide, companion/home,
 * menomessages, conditionnudges → the frontend screens). The words live in `smartMessages.where.<group>` /
 * `smartMessages.areas.<area>`; this file only orders and classifies them.
 *
 * `notShown` is for a group no screen renders: since D-216 the cycle groups of the old `/messages/daily` bundle,
 * whose messages are engine rules now (`tip.*`, edited under «قواعد نمایش پیام»). The Android app is a TWA of
 * the web app (package web.ritme.app, frontend/public/.well-known/assetlinks.json), so web = Android here.
 * The pregnancy groups of the old `/messages/daily` bundle feed the pregnancy home's «نکتهٔ هوشمند» card
 * (frontend screens/pregnancy/model/smart-tip.ts) together with `pregnancy_week_tip`.
 */
export type MessageArea =
  | 'cycleHome'
  | 'conditions'
  | 'pregnancy'
  | 'postpartum'
  | 'menopause'
  | 'companion'
  | 'analysis'
  | 'notShown';

export const AREAS: readonly MessageArea[] = [
  'cycleHome',
  'conditions',
  'pregnancy',
  'postpartum',
  'menopause',
  'companion',
  'analysis',
  'notShown',
];

const GROUP_AREA: Record<string, MessageArea> = {
  condition_nudge: 'conditions',
  pregnancy_week_tip: 'pregnancy',
  pregnancy_alert: 'pregnancy',
  pregnancy_setup: 'pregnancy',
  pregnancy_override: 'pregnancy',
  correlation_pregnancy: 'pregnancy',
  nutrition_trimester: 'pregnancy',
  sleep_trimester: 'pregnancy',
  exercise_trimester: 'pregnancy',
  pregnancy_trimester: 'pregnancy',
  pregnancy_week: 'pregnancy',
  pregnancy_base: 'pregnancy',
  correlation_common: 'pregnancy',
  cycle_base_non_ttc: 'notShown',
  cycle_base_ttc: 'notShown',
  cycle_override: 'notShown',
  correlation_cycle: 'notShown',
  pattern: 'notShown',
  nutrition_cycle: 'notShown',
  nutrition_ttc: 'notShown',
  nutrition_symptom: 'notShown',
  sleep_cycle: 'notShown',
  sleep_symptom: 'notShown',
  exercise_cycle: 'notShown',
  exercise_symptom: 'notShown',
  exercise_ttc: 'notShown',
  postpartum_week_tip: 'postpartum',
  postpartum_alert: 'postpartum',
  postpartum_safety: 'postpartum',
  menopause_message: 'menopause',
  companion_tip: 'companion',
  bmi_message: 'analysis',
};

/** The area of a group; unknown (newly registered) groups land in `null` and are listed last. */
export function areaOf(group: string): MessageArea | null {
  return GROUP_AREA[group] ?? null;
}

/** `groups` bucketed by area in AREAS order (unknown groups under `null`, last), each bucket in input order. */
export function groupByArea(groups: readonly string[]): Array<{ area: MessageArea | null; groups: string[] }> {
  const buckets = new Map<MessageArea | null, string[]>();
  for (const g of groups) {
    const a = areaOf(g);
    buckets.set(a, [...(buckets.get(a) ?? []), g]);
  }
  return [...AREAS, null]
    .filter((a) => buckets.has(a))
    .map((a) => ({ area: a, groups: buckets.get(a) ?? [] }));
}

/** Words a composite item key is built from (`luteal_tip_2`, `menstrual_note`) — see `smartMessages.itemKeys`. */
const TIP = /^([a-z]+)_tip_(\d+)$/;
const NOTE = /^([a-z]+)_note$/;

export type ItemKeyLabel =
  | { kind: 'key'; key: string }
  | { kind: 'week'; n: number }
  | { kind: 'trimester'; n: number }
  | { kind: 'phaseTip'; phase: string; n: number }
  | { kind: 'phaseNote'; phase: string }
  | { kind: 'raw' };

const TRIMESTER_GROUPS = new Set(['pregnancy_trimester', 'nutrition_trimester', 'sleep_trimester', 'exercise_trimester']);
const WEEK_GROUPS = new Set(['pregnancy_week', 'pregnancy_week_tip', 'postpartum_week_tip']);

/** How to name an item key: numbered weeks / trimesters, companion phase slots, or a word of the dictionary. */
export function describeItemKey(group: string, key: string, known: (key: string) => boolean): ItemKeyLabel {
  if (/^\d+$/.test(key)) {
    if (TRIMESTER_GROUPS.has(group)) return { kind: 'trimester', n: Number(key) };
    if (WEEK_GROUPS.has(group)) return { kind: 'week', n: Number(key) };
  }
  const tip = TIP.exec(key);
  if (tip?.[1] && tip[2] && known(tip[1])) return { kind: 'phaseTip', phase: tip[1], n: Number(tip[2]) };
  const note = NOTE.exec(key);
  if (note?.[1] && known(note[1])) return { kind: 'phaseNote', phase: note[1] };
  if (known(key)) return { kind: 'key', key };
  return { kind: 'raw' };
}
