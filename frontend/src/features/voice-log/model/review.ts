import type { LogCategory } from '@/entities/health-log';

import type { VoiceCommitItem, VoiceDiaryTarget, VoiceResult, VoiceSuggestion, VoiceTarget } from '../api/voice';

/**
 * Review state of one recording (nbl_Voice_Review), pure: every suggestion becomes an item the user can
 * drop, re-pick (ambiguity chooser from `options[]`) or re-value (a single-choice param's options); items
 * are grouped by where they land. No React, no locale.
 */

/** One reading of a word — the suggestion itself or one of its `options[]`. */
export interface VoiceChoice {
  category: string;
  param: string;
  item: string | null;
  value: string | number | boolean;
  label: string;
}

export interface ReviewItem {
  id: string;
  target: VoiceTarget;
  /** What is saved now (the suggestion, an option, or the suggestion with another single-choice value). */
  choice: VoiceChoice;
  /** Suggestion first, then its options; more than one = «مطمئن نیستیم». */
  choices: VoiceChoice[];
  included: boolean;
}

export interface ReviewGroup {
  /** `log:<category>` or the diary target. */
  key: string;
  target: VoiceTarget;
  /** Taxonomy category of a log group; the target for a diary group. */
  category: string;
  items: ReviewItem[];
}

const DIARY: readonly VoiceDiaryTarget[] = ['hot_flash', 'pain_diary', 'pill', 'bladder'];

export function isDiaryTarget(target: string): target is VoiceDiaryTarget {
  return (DIARY as readonly string[]).includes(target);
}

function choiceOf(s: Pick<VoiceSuggestion, 'category' | 'param' | 'item' | 'value' | 'label'>): VoiceChoice {
  return { category: s.category, param: s.param, item: s.item, value: s.value, label: s.label };
}

/** Whether the log sheet shows this param (another mode's or a hidden category can't be saved here). */
export function inSheet(categories: readonly LogCategory[], c: Pick<VoiceChoice, 'category' | 'param'>): boolean {
  return categories.some((cat) => cat.code === c.category && cat.params.some((p) => p.code === c.param));
}

/**
 * Review items of a result: log suggestions the sheet can't show are dropped (as before), diary targets are
 * kept; options the sheet can't show are dropped from the chooser.
 */
export function reviewItems(result: VoiceResult, categories: readonly LogCategory[]): ReviewItem[] {
  const out: ReviewItem[] = [];
  result.suggestions.forEach((s, i) => {
    const primary = choiceOf(s);
    if (s.target === 'log' && !inSheet(categories, primary)) return;
    const options = s.target === 'log' ? s.options.filter((o) => inSheet(categories, o)).map(choiceOf) : s.options.map(choiceOf);
    out.push({ id: `${i}:${s.category}.${s.param}.${s.item ?? ''}`, target: s.target, choice: primary, choices: [primary, ...options], included: true });
  });
  return out;
}

/** Items grouped by log category / diary target, in first-mention order. */
export function groupItems(items: readonly ReviewItem[]): ReviewGroup[] {
  const groups: ReviewGroup[] = [];
  for (const item of items) {
    const category = item.target === 'log' ? item.choices[0].category : item.target;
    const key = item.target === 'log' ? `log:${category}` : item.target;
    let g = groups.find((x) => x.key === key);
    if (!g) {
      g = { key, target: item.target, category, items: [] };
      groups.push(g);
    }
    g.items.push(item);
  }
  return groups;
}

export function toggleItem(items: readonly ReviewItem[], id: string): ReviewItem[] {
  return items.map((i) => (i.id === id ? { ...i, included: !i.included } : i));
}

/** Picks another reading of an ambiguous item (always included afterwards). */
export function chooseItem(items: readonly ReviewItem[], id: string, index: number): ReviewItem[] {
  return items.map((i) => (i.id === id && i.choices[index] ? { ...i, choice: i.choices[index], included: true } : i));
}

/** Sets another option of a single-choice param (bleeding flow «کم · متوسط · زیاد»). */
export function revalueItem(items: readonly ReviewItem[], id: string, value: string, label: string): ReviewItem[] {
  return items.map((i) => (i.id === id ? { ...i, choice: { ...i.choice, value, label }, included: true } : i));
}

export function includedItems(items: readonly ReviewItem[]): ReviewItem[] {
  return items.filter((i) => i.included);
}

/** The included log items as suggestions for the sheet's merge (PUT /logs/days). */
export function logSuggestions(items: readonly ReviewItem[]): VoiceSuggestion[] {
  return includedItems(items)
    .filter((i) => i.target === 'log')
    .map((i) => ({ target: 'log', ...i.choice, confidence: 1, options: [] }));
}

/** The included diary items for POST /logs/voice/commit. */
export function commitItems(items: readonly ReviewItem[]): VoiceCommitItem[] {
  return includedItems(items).flatMap((i) =>
    isDiaryTarget(i.target) ? [{ category: i.target, param: i.choice.param, value: i.choice.value }] : [],
  );
}

/** Whether an included item is heavy bleeding (the board's «یک نکته» safety note). */
export function hasHeavyBleeding(items: readonly ReviewItem[]): boolean {
  return includedItems(items).some(
    (i) => i.choice.category === 'bleeding' && i.choice.param === 'flow' && (i.choice.value === 'heavy' || i.choice.value === 'very_heavy'),
  );
}

/** A server chip label «درد زیر شکم · متوسط» as the saved list's title + caption. */
export function splitLabel(label: string): { title: string; sub: string | null } {
  const i = label.indexOf(' · ');
  return i < 0 ? { title: label, sub: null } : { title: label.slice(0, i), sub: label.slice(i + 3) };
}

/** One row of the saved list (nbl_Voice_Saved). */
export interface SavedRow {
  key: string;
  /** Taxonomy category (log) or diary target — picks the icon. */
  category: string;
  label: string;
  /** Category / diary name: the row's title when the label has no « · » caption («حال» over «بی‌حوصله»). */
  group?: string;
}

/** Title + caption of a saved row. */
export function savedRowText(row: SavedRow): { title: string; sub: string | null } {
  const { title, sub } = splitLabel(row.label);
  if (sub || !row.group || row.group === title) return { title, sub };
  return { title: row.group, sub: title };
}

/** Today's-status rows on the entry screen (nbl_Voice_Entry «ثبت‌های امروز»), per mode. */
const STATUS_ROWS: Record<string, readonly string[]> = {
  menopause: ['symptoms', 'sleep', 'bleeding'],
  pregnancy: ['symptoms', 'mood', 'sleep'],
  postpartum: ['bleeding', 'mood', 'sleep'],
};
const DEFAULT_STATUS_ROWS = ['bleeding', 'pain', 'sleep'];

export function statusCategories(mode: string | null, categories: readonly LogCategory[]): LogCategory[] {
  const wanted = (mode && STATUS_ROWS[mode]) || DEFAULT_STATUS_ROWS;
  return wanted.flatMap((code) => categories.filter((c) => c.code === code));
}
