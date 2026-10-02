import { describe, expect, it } from 'vitest';

import type { LogCategory, LogDayValues, LogParam } from '@/entities/health-log';

import {
  hasItemRow,
  itemRowLevel,
  loggedItemCount,
  modePresetOf,
  POSTPARTUM_CARDS,
  PREGNANCY_CARDS,
  presetCards,
  rowAcceptsNo,
  rowAvailable,
  withItemRowLevel,
  type ItemRow,
  type PresetRowSpec,
} from './mode-presets';

function param(p: Partial<LogParam> & Pick<LogParam, 'code' | 'type'>): LogParam {
  return {
    label: p.code,
    modes: [],
    detail: false,
    alert: false,
    options: [],
    levels: [],
    score: null,
    range: null,
    scale: null,
    unit: null,
    maxLength: null,
    dynamic: false,
    source: null,
    ...p,
  };
}

const opt = (value: string) => ({ value, label: value, modes: null, legacyOnly: false });
const lv = (value: string) => ({ value, label: value });

function category(code: string, params: LogParam[]): LogCategory {
  return { code, label: code, group: { value: 'body', label: 'body' }, modes: [], conditions: {}, params };
}

const categories: LogCategory[] = [
  category('symptoms', [
    param({ code: 'digestive', type: 'items', options: ['nausea', 'heartburn'].map(opt), levels: ['yes', 'no', 'mild', 'moderate', 'severe'].map(lv) }),
  ]),
  category('pain', [param({ code: 'location', type: 'items', options: ['back', 'pelvis'].map(opt), levels: ['mild', 'moderate', 'severe'].map(lv) })]),
  category('measurements', [param({ code: 'weight', type: 'number' })]),
  category('baby', [param({ code: 'feeding', type: 'link', source: 'feeding' })]),
];

const nausea: ItemRow = { category: 'symptoms', param: 'digestive', item: 'nausea' };
const back: ItemRow = { category: 'pain', param: 'location', item: 'back' };

describe('modePresetOf', () => {
  it('applies to pregnancy and postpartum only', () => {
    expect(modePresetOf('pregnancy')).toBe('pregnancy');
    expect(modePresetOf('postpartum')).toBe('postpartum');
    expect(modePresetOf('menopause')).toBeNull();
    expect(modePresetOf('cycle')).toBeNull();
    expect(modePresetOf(null)).toBeNull();
  });

  it('maps each kind to its board cards', () => {
    expect(presetCards('pregnancy')).toBe(PREGNANCY_CARDS);
    expect(presetCards('postpartum')).toBe(POSTPARTUM_CARDS);
    expect(PREGNANCY_CARDS.map((c) => c.code)).toEqual(['symptoms', 'measure']);
    expect(POSTPARTUM_CARDS.map((c) => c.code)).toEqual(['mother', 'baby']);
  });
});

describe('availability', () => {
  it('needs the item in the taxonomy', () => {
    expect(hasItemRow(categories, nausea)).toBe(true);
    expect(hasItemRow(categories, { ...nausea, item: 'diarrhea' })).toBe(false);
    expect(hasItemRow(categories, { category: 'breasts', param: 'symptoms', item: 'redness' })).toBe(false);
  });

  it('checks panel params, sections and link params', () => {
    const weight: PresetRowSpec = { kind: 'panel', code: 'weight', icon: 'scale', tone: 'data', panel: 'measure', category: 'measurements', params: ['weight'] };
    const vitals: PresetRowSpec = { ...weight, code: 'vitals', params: ['bp_systolic'] };
    const meds: PresetRowSpec = { kind: 'section', code: 'meds', icon: 'tablet', tone: 'bloom', category: 'meds' };
    const feeding: PresetRowSpec = { kind: 'link', code: 'feeding', icon: 'sprout', tone: 'data', category: 'baby', param: 'feeding' };
    expect(rowAvailable(categories, weight)).toBe(true);
    expect(rowAvailable(categories, vitals)).toBe(false);
    expect(rowAvailable(categories, meds)).toBe(false);
    expect(rowAvailable(categories, feeding)).toBe(true);
    expect(rowAvailable(categories, { ...feeding, param: 'diapers' })).toBe(false);
  });
});

describe('severity rows', () => {
  it('stores «ندارم» only where the param has a no level', () => {
    expect(rowAcceptsNo(categories, nausea)).toBe(true);
    expect(rowAcceptsNo(categories, back)).toBe(false);
    expect(withItemRowLevel({}, nausea, 'no', true)).toEqual({ nausea: { level: 'no', score: null } });
    const values: LogDayValues = { pain: { location: { back: { level: 'mild', score: 3 } } } };
    expect(withItemRowLevel(values, back, 'no', false)).toEqual({});
  });

  it('re-picking the level clears the row and keeps the other items', () => {
    const values: LogDayValues = { symptoms: { digestive: { nausea: { level: 'moderate', score: null }, heartburn: { level: 'mild', score: null } } } };
    expect(withItemRowLevel(values, nausea, 'moderate', true)).toEqual({ heartburn: { level: 'mild', score: null } });
    expect(withItemRowLevel(values, nausea, 'severe', true)).toEqual({
      nausea: { level: 'severe', score: null },
      heartburn: { level: 'mild', score: null },
    });
  });

  it('drops a stale pain score on a new level', () => {
    const values: LogDayValues = { pain: { location: { back: { level: 'severe', score: 9 } } } };
    expect(withItemRowLevel(values, back, 'mild', false)).toEqual({ back: { level: 'mild', score: null } });
  });

  it('reads levels, a plain «دارم» as mild, and counts real symptoms', () => {
    const values: LogDayValues = {
      symptoms: { digestive: { nausea: { level: 'yes', score: null }, heartburn: { level: 'no', score: null } } },
      pain: { location: { back: { level: 'moderate', score: 5 } } },
    };
    expect(itemRowLevel(values, nausea)).toBe('mild');
    expect(itemRowLevel(values, { ...nausea, item: 'heartburn' })).toBe('no');
    expect(itemRowLevel(values, back)).toBe('moderate');
    expect(itemRowLevel({}, back)).toBeNull();
    expect(loggedItemCount(values, [nausea, { ...nausea, item: 'heartburn' }, back])).toBe(2);
  });
});
