import { describe, expect, it } from 'vitest';

import { hintFor } from './hints';
import {
  audiencesError,
  cleanTranslations,
  formatMeta,
  invalidAudiences,
  isValidCode,
  moveItem,
  parseAudiences,
  parseMeta,
  partialUpdate,
  planReorder,
} from './payload';

const row = (id: number, sort_order: number) => ({ id, sort_order, title: { fa: `ع${id}` }, is_active: true });

describe('codes', () => {
  it('accepts lowercase snake case starting with a letter', () => {
    expect(isValidCode('teen_faq')).toBe(true);
    expect(isValidCode('a1')).toBe(true);
    expect(isValidCode('1a')).toBe(false);
    expect(isValidCode('Teen')).toBe(false);
    expect(isValidCode('')).toBe(false);
    expect(isValidCode('a'.repeat(65))).toBe(false);
  });
});

describe('audiences', () => {
  it('splits on commas, spaces and «،», lowercases and dedupes', () => {
    expect(parseAudiences(' Teen, menopause،teen  ttc ')).toEqual(['teen', 'menopause', 'ttc']);
    expect(parseAudiences('')).toEqual([]);
  });
  it('reports codes the API would reject', () => {
    expect(invalidAudiences(['teen', '9x', 'a'.repeat(33)])).toEqual(['9x', 'a'.repeat(33)]);
  });
  it('finds the first audiences.* error in a 422 bag', () => {
    expect(audiencesError({ 'title.fa': ['x'], 'audiences.1': ['bad'] })).toBe('bad');
    expect(audiencesError(undefined)).toBeUndefined();
  });
});

describe('meta', () => {
  it('blank, null and {} clear the column', () => {
    expect(parseMeta('  ')).toEqual({ ok: true, value: null });
    expect(parseMeta('null')).toEqual({ ok: true, value: null });
    expect(parseMeta('{}')).toEqual({ ok: true, value: null });
  });
  it('accepts an object and rejects bad JSON, lists, scalars and oversize', () => {
    expect(parseMeta('{"severity":"urgent","cta":{"fa":"تماس"}}')).toEqual({
      ok: true,
      value: { severity: 'urgent', cta: { fa: 'تماس' } },
    });
    expect(parseMeta('{bad')).toEqual({ ok: false, reason: 'json' });
    expect(parseMeta('[1]')).toEqual({ ok: false, reason: 'object' });
    expect(parseMeta('3')).toEqual({ ok: false, reason: 'object' });
    expect(parseMeta(JSON.stringify({ x: 'ی'.repeat(9000) }))).toEqual({ ok: false, reason: 'size' });
  });
  it('formats stored meta as pretty raw UTF-8', () => {
    expect(formatMeta(null)).toBe('');
    expect(formatMeta({})).toBe('');
    expect(formatMeta({ a: 'ب' })).toBe('{\n  "a": "ب"\n}');
  });
});

describe('write bodies', () => {
  it('cleanTranslations trims and nulls empties', () => {
    expect(cleanTranslations({ fa: ' x ', en: '' })).toEqual({ fa: 'x' });
    expect(cleanTranslations({ fa: ' ' })).toBeNull();
  });
  it('a partial update always carries the required title', () => {
    expect(partialUpdate(row(4, 2), { is_active: false })).toEqual({ title: { fa: 'ع4' }, is_active: false });
  });
});

describe('reorder', () => {
  it('moveItem moves and ignores out-of-range moves', () => {
    expect(moveItem([1, 2, 3], 2, 0)).toEqual([3, 1, 2]);
    expect(moveItem([1, 2], 1, 2)).toEqual([1, 2]);
  });
  it('swapping neighbours reuses their slots (two PUTs)', () => {
    const rows = [row(1, 10), row(2, 20), row(3, 30)];
    expect(planReorder(moveItem(rows, 2, 1))).toEqual([
      { id: 3, sort_order: 20 },
      { id: 2, sort_order: 30 },
    ]);
  });
  it('moving to the top shifts only the rows in between', () => {
    const rows = [row(1, 1), row(2, 2), row(3, 3), row(4, 4)];
    expect(planReorder(moveItem(rows, 2, 0))).toEqual([
      { id: 3, sort_order: 1 },
      { id: 1, sort_order: 2 },
      { id: 2, sort_order: 3 },
    ]);
  });
  it('renumbers 1…n when stored orders collide', () => {
    const rows = [row(1, 0), row(2, 0), row(3, 0)];
    expect(planReorder(moveItem(rows, 0, 2))).toEqual([
      { id: 2, sort_order: 1 },
      { id: 3, sort_order: 2 },
      { id: 1, sort_order: 3 },
    ]);
  });
  it('an unchanged order needs no writes', () => {
    expect(planReorder([row(1, 1), row(2, 5)])).toEqual([]);
  });
});

describe('hints', () => {
  it('matches exact groups, then suffix conventions, else generic', () => {
    expect(hintFor('pelvic_levels').key).toBe('pelvicLevels');
    expect(hintFor('meno_alerts').key).toBe('alerts');
    expect(hintFor('teen_faq')).toEqual({ key: 'faq', example: null });
    expect(hintFor('whatever').key).toBe('generic');
  });
});
