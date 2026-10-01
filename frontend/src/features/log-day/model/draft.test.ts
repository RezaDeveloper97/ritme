import { describe, expect, it } from 'vitest';

import type { LogCategory, LogParam } from '@/entities/health-log';

import {
  dayEntries,
  defaultLevel,
  diffDay,
  isEmptyValue,
  isGraded,
  levelFromScore,
  matchesSearch,
  normalizeNumber,
  normalizeSearch,
  parseDecimal,
  parseTileKey,
  setAllLevels,
  setItemScore,
  setValue,
  toggleItem,
  valuesEqual,
  type LabelContext,
} from './draft';

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

const opt = (value: string, label = value) => ({ value, label, modes: null, legacyOnly: false });
const lvl = (value: string, label = value) => ({ value, label });

const pain: LogCategory = {
  code: 'pain',
  label: 'درد',
  group: { value: 'body', label: 'بدن' },
  modes: ['cycle'],
  conditions: {},
  params: [
    param({ code: 'none', type: 'bool', label: 'بدون درد' }),
    param({
      code: 'location',
      type: 'items',
      label: 'محل',
      options: [opt('abdomen', 'شکم'), opt('head', 'سر')],
      levels: [lvl('mild', 'کم'), lvl('moderate', 'متوسط'), lvl('severe', 'شدید')],
      score: { min: 1, max: 10 },
    }),
  ],
};

const measurements: LogCategory = {
  code: 'measurements',
  label: 'وزن و دمای پایه',
  group: { value: 'measure', label: 'اندازه' },
  modes: ['cycle'],
  conditions: {},
  params: [
    param({ code: 'weight', type: 'number', label: 'وزن', scale: 2, range: { min: 20, max: 300 }, unit: lvl('kg', 'کیلو') }),
  ],
};

const ctx: LabelContext = {
  extraLabels: { custom_3: 'سفارشی من' },
  unknown: 'ثبت شده',
  formatNumber: (n, scale) => n.toFixed(scale),
};

describe('log-day draft', () => {
  it('treats empty strings, lists and objects as nothing logged', () => {
    expect(isEmptyValue('')).toBe(true);
    expect(isEmptyValue([])).toBe(true);
    expect(isEmptyValue({})).toBe(true);
    expect(isEmptyValue(false)).toBe(false);
    expect(isEmptyValue(0)).toBe(false);
  });

  it('compares multi lists without caring about order', () => {
    expect(valuesEqual(['a', 'b'], ['b', 'a'])).toBe(true);
    expect(valuesEqual({ x: { level: 'yes', score: null } }, { x: { level: 'yes', score: null } })).toBe(true);
    expect(valuesEqual(undefined, [])).toBe(true);
    expect(valuesEqual('a', 'b')).toBe(false);
  });

  it('sets and clears params immutably, dropping empty categories', () => {
    const a = setValue({}, 'mood', 'moods', ['calm']);
    expect(a).toEqual({ mood: { moods: ['calm'] } });
    const b = setValue(a, 'mood', 'moods', []);
    expect(b).toEqual({});
    expect(a).toEqual({ mood: { moods: ['calm'] } });
  });

  it('diffs the draft into one partial PUT body (null clears)', () => {
    const saved = { bleeding: { flow: 'medium', color: 'red' }, note: { text: 'x' } };
    const draft = { bleeding: { flow: 'heavy', color: 'red' }, mood: { moods: ['calm'] } };
    expect(diffDay(saved, draft)).toEqual({
      bleeding: { flow: 'heavy' },
      note: { text: null },
      mood: { moods: ['calm'] },
    });
    expect(diffDay(saved, saved)).toEqual({});
  });

  it('picks the right default level and grades pain', () => {
    const loc = pain.params[1];
    expect(isGraded(loc)).toBe(true);
    expect(defaultLevel(loc)).toBe('moderate');
    expect(defaultLevel(param({ code: 's', type: 'items', levels: [lvl('yes'), lvl('no'), lvl('mild')] }))).toBe('yes');
  });

  it('toggles items and maps scores to levels', () => {
    const one = toggleItem({}, 'abdomen', 'moderate');
    expect(one).toEqual({ abdomen: { level: 'moderate', score: null } });
    expect(toggleItem(one, 'abdomen', 'moderate')).toEqual({});
    expect(levelFromScore(2)).toBe('mild');
    expect(levelFromScore(5)).toBe('moderate');
    expect(levelFromScore(9)).toBe('severe');
    const scored = setItemScore(one, 'abdomen', 8);
    expect(scored.abdomen).toEqual({ level: 'severe', score: 8 });
    expect(setAllLevels(scored, 'mild').abdomen).toEqual({ level: 'mild', score: null });
    expect(setAllLevels(scored, 'severe').abdomen).toEqual({ level: 'severe', score: 8 });
  });

  it('lists what was logged with summaries', () => {
    const values = {
      pain: { location: { abdomen: { level: 'moderate', score: null } } },
      measurements: { weight: 58.4 },
    };
    const entries = dayEntries([pain, measurements], values, ctx);
    expect(entries.map((e) => e.label)).toEqual(['شکم', 'وزن']);
    expect(entries.map((e) => e.summary)).toEqual(['شکم · متوسط', '58.4 کیلو']);
  });

  it('labels unknown codes and custom items', () => {
    const mood: LogCategory = { ...pain, code: 'mood', params: [param({ code: 'moods', type: 'multi', options: [opt('calm', 'آرام')] })] };
    const entries = dayEntries([mood], { mood: { moods: ['calm', 'custom_3', 'legacy_x'] } }, ctx);
    expect(entries.map((e) => e.label)).toEqual(['آرام', 'سفارشی من', 'ثبت شده']);
  });

  it('parses numbers typed in any digit script', () => {
    expect(parseDecimal('۵۸٫۴')).toBe(58.4);
    expect(parseDecimal('36,5')).toBe(36.5);
    expect(parseDecimal('abc')).toBeNull();
    expect(normalizeNumber(measurements.params[0], 500)).toBe(300);
    expect(normalizeNumber(measurements.params[0], 58.456)).toBe(58.46);
  });

  it('parses tile keys and searches labels across keyboards', () => {
    expect(parseTileKey('pain')).toEqual({ category: 'pain', param: null });
    expect(parseTileKey('measurements.weight')).toEqual({ category: 'measurements', param: 'weight' });
    expect(normalizeSearch('سردرد‌ي')).toBe('سردردی');
    expect(matchesSearch(pain, 'شکم')).toBe(true);
    expect(matchesSearch(pain, 'وزن')).toBe(false);
    expect(matchesSearch(pain, '')).toBe(true);
  });
});
