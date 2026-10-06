import type { LogCategory, LogDayValues } from '@/entities/health-log';
import type { IconName, Tone } from '@/shared/ui';

import { asItems, type ItemsValue } from './draft';

/**
 * Pregnancy and postpartum presets of the log sheet (B-N3-06, nbl_/nbd_Log_Sheet_Preg and _Post): the
 * boards' grouped rows laid over ordinary taxonomy slots. Like the menopause preset, the grouping is
 * presentation only — `GET /logs/taxonomy` doesn't expose it, and a slot the taxonomy doesn't return for
 * the mode is simply not drawn.
 */

export type ModePresetKind = 'pregnancy' | 'postpartum';

/** Which of the two presets a taxonomy mode gets (`null` = bloom's plain layout or the menopause preset). */
export function modePresetOf(mode: string | null): ModePresetKind | null {
  return mode === 'pregnancy' || mode === 'postpartum' ? mode : null;
}

/** One severity row inside an expandable group: an item of an `items` param. */
export interface ItemRow {
  category: string;
  param: string;
  item: string;
}

/**
 * A row of a preset card. `items` expands in place into severity rows; `panel` opens a detail panel
 * (bleeding, pain + body map, measurements); `section` opens that category's accordion in «همه موارد»;
 * `link` is a tile fed by another feature (feeding, diapers…) — it navigates, or shows «به‌زودی».
 */
export type PresetRowSpec =
  | { kind: 'items'; code: string; icon: IconName; tone: Tone; rows: readonly ItemRow[]; section?: string }
  | {
      kind: 'panel';
      code: string;
      icon: IconName;
      tone: Tone;
      panel: 'bleeding' | 'pain' | 'measure';
      category: string;
      params?: readonly string[];
      /** B-N6-02: the row opens these screens instead of the panel (copy `rows.<code>.add.<key>`). */
      links?: ReadonlyArray<{ key: string; href: string }>;
    }
  | { kind: 'section'; code: string; icon: IconName; tone: Tone; category: string }
  | { kind: 'link'; code: string; icon: IconName; tone: Tone; category: string; param: string };

export interface PresetCardSpec {
  code: string;
  rows: readonly PresetRowSpec[];
}

const row = (category: string, param: string, item: string): ItemRow => ({ category, param, item });

/** nbl_Log_Sheet_Preg: «علائم بارداری» (گوارش · بدن · ادراری و تناسلی) and «اندازه‌گیری». */
export const PREGNANCY_CARDS: readonly PresetCardSpec[] = [
  {
    code: 'symptoms',
    rows: [
      {
        kind: 'items',
        code: 'digestive',
        icon: 'gut',
        tone: 'warm',
        rows: [
          row('symptoms', 'digestive', 'nausea'),
          row('symptoms', 'digestive', 'heartburn'),
          row('symptoms', 'digestive', 'constipation'),
          row('symptoms', 'digestive', 'bloating'),
        ],
      },
      {
        kind: 'items',
        code: 'body',
        icon: 'symptom',
        tone: 'bloom',
        rows: [
          row('pain', 'location', 'back'),
          row('pain', 'location', 'pelvis'),
          row('symptoms', 'general', 'leg_cramps'),
          row('symptoms', 'general', 'swelling'),
        ],
      },
      {
        kind: 'items',
        code: 'urogenital',
        icon: 'urine',
        tone: 'brand',
        rows: [row('urogenital', 'symptoms', 'frequent_urination'), row('urogenital', 'symptoms', 'urination_burning')],
        // «ترشحات» has its own category (consistency, amount…): the group ends with a link to it.
        section: 'discharge',
      },
    ],
  },
  {
    code: 'measure',
    rows: [
      { kind: 'panel', code: 'weight', icon: 'scaleSquare', tone: 'data', panel: 'measure', category: 'measurements', params: ['weight'] },
      {
        kind: 'panel',
        code: 'vitals',
        icon: 'heartLine',
        tone: 'brand',
        panel: 'measure',
        category: 'measurements',
        params: ['bp_systolic', 'bp_diastolic', 'blood_sugar'],
        // B-N6-02: timed readings live in Vitals now; the day's log values still show here and in Vitals (read-only).
        links: [
          { key: 'bp', href: '/vitals/bp/new' },
          { key: 'glucose', href: '/vitals/glucose/new' },
        ],
      },
      { kind: 'section', code: 'meds', icon: 'tablet', tone: 'bloom', category: 'meds' },
    ],
  },
];

/** nbl_Log_Sheet_Post: «مادر» (خون‌ریزی · درد و بخیه · سینه‌ها · حال روحی) and «نوزاد». */
export const POSTPARTUM_CARDS: readonly PresetCardSpec[] = [
  {
    code: 'mother',
    rows: [
      {
        kind: 'panel',
        code: 'lochia',
        icon: 'drop',
        tone: 'period',
        panel: 'bleeding',
        category: 'bleeding',
        params: ['lochia_amount', 'lochia_color', 'clots', 'clot_size', 'odor'],
      },
      { kind: 'panel', code: 'pain', icon: 'symptom', tone: 'bloom', panel: 'pain', category: 'pain' },
      {
        kind: 'items',
        code: 'breasts',
        icon: 'heart',
        tone: 'bloom',
        rows: [row('breasts', 'symptoms', 'engorgement'), row('breasts', 'symptoms', 'nipple_pain'), row('breasts', 'symptoms', 'redness')],
      },
      { kind: 'section', code: 'mood', icon: 'smile', tone: 'brand', category: 'mood' },
    ],
  },
  {
    code: 'baby',
    rows: [
      { kind: 'link', code: 'feeding', icon: 'sprout', tone: 'data', category: 'baby', param: 'feeding' },
      { kind: 'link', code: 'baby_sleep', icon: 'moon', tone: 'brand', category: 'baby', param: 'baby_sleep' },
      { kind: 'link', code: 'diapers', icon: 'box', tone: 'warm', category: 'baby', param: 'diapers' },
    ],
  },
];

export function presetCards(kind: ModePresetKind): readonly PresetCardSpec[] {
  return kind === 'pregnancy' ? PREGNANCY_CARDS : POSTPARTUM_CARDS;
}

/** Whether the taxonomy returns the row's category (and, when the row names params, at least one of them). */
export function rowAvailable(categories: readonly LogCategory[], spec: PresetRowSpec): boolean {
  switch (spec.kind) {
    case 'items':
      return spec.rows.some((r) => hasItemRow(categories, r));
    case 'panel': {
      const cat = categories.find((c) => c.code === spec.category);
      if (!cat) return false;
      return !spec.params || cat.params.some((p) => spec.params!.includes(p.code));
    }
    case 'section':
      return categories.some((c) => c.code === spec.category);
    case 'link':
      return !!categories.find((c) => c.code === spec.category)?.params.some((p) => p.code === spec.param && p.type === 'link');
  }
}

/** Whether the taxonomy offers the item row (its param exists and lists the item). */
export function hasItemRow(categories: readonly LogCategory[], r: ItemRow): boolean {
  const param = categories.find((c) => c.code === r.category)?.params.find((p) => p.code === r.param);
  return !!param && param.type === 'items' && param.options.some((o) => o.value === r.item);
}

/** The four steps of a severity row; «ندارم» first. */
export const ROW_LEVELS = ['no', 'mild', 'moderate', 'severe'] as const;
export type RowLevel = (typeof ROW_LEVELS)[number];

/** Whether the param stores an explicit «ندارم» (symptoms do; pain locations don't — «ندارم» = no entry). */
export function rowAcceptsNo(categories: readonly LogCategory[], r: ItemRow): boolean {
  const param = categories.find((c) => c.code === r.category)?.params.find((p) => p.code === r.param);
  return !!param && param.levels.some((l) => l.value === 'no');
}

/** The row's level, `null` when nothing is logged. A plain «دارم» (yes) reads as «کم». */
export function itemRowLevel(values: LogDayValues, r: ItemRow): RowLevel | null {
  const level = asItems(values[r.category]?.[r.param])[r.item]?.level;
  if (level === 'yes') return 'mild';
  return (ROW_LEVELS as readonly string[]).includes(level ?? '') ? (level as RowLevel) : null;
}

/**
 * The param's new items value after picking `level` on a row: picking the current level again clears it,
 * «ندارم» on a param without a `no` level removes the item. Other items of the param are kept.
 */
export function withItemRowLevel(values: LogDayValues, r: ItemRow, level: RowLevel, noLevel: boolean): ItemsValue {
  const items = { ...asItems(values[r.category]?.[r.param]) };
  const current = itemRowLevel(values, r);
  if (current === level || (level === 'no' && !noLevel)) delete items[r.item];
  // A new level drops a stale 1–10 pain score (the pain panel sets both together).
  else items[r.item] = { level, score: null };
  return items;
}

/** How many item rows of a group carry a real symptom (anything but «ندارم»). */
export function loggedItemCount(values: LogDayValues, rows: readonly ItemRow[]): number {
  return rows.filter((r) => {
    const level = itemRowLevel(values, r);
    return level !== null && level !== 'no';
  }).length;
}
