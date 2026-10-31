import { describe, expect, it } from 'vitest';

import type { FactDef } from '../api/facts';
import { emptyConditions, fromStored, inputKind, onFactChange, opsFor, parseValue, toStored, usesFertilityFact, type Group } from './conditions';

const def = (name: string, type: FactDef['type'], extra: Partial<FactDef> = {}): FactDef => ({
  name, type, enum: null, unit: null, group: 'cycle', description: {}, member_safe: false, pending: null, ...extra,
});
const FACTS: FactDef[] = [
  def('days_to_period', 'int', { unit: 'days' }),
  def('mode', 'enum', { enum: ['cycle', 'ttc', 'pregnancy'] }),
  def('symptoms', 'set', { enum: ['cramps', 'nausea'] }),
  def('on_pill', 'bool'),
  def('current_period_start', 'date'),
  def('fertile_today', 'bool'),
];

describe('condition builder model', () => {
  it('offers only the operators valid for the fact type', () => {
    expect(opsFor('int')).toContain('between');
    expect(opsFor('bool')).toEqual(['eq', 'ne']);
    expect(opsFor('enum')).toEqual(['eq', 'ne', 'in', 'nin']);
    expect(opsFor('set')).toEqual(['contains', 'in', 'nin']);
    expect(opsFor('date')).not.toContain('gt');
  });

  it('picks the input by type and operator', () => {
    expect(inputKind(FACTS[0], 'between')).toBe('interval');
    expect(inputKind(FACTS[0], 'gte')).toBe('number');
    expect(inputKind(FACTS[1], 'in')).toBe('enum-multi');
    expect(inputKind(FACTS[1], 'eq')).toBe('enum-single');
    expect(inputKind(FACTS[3], 'eq')).toBe('bool');
    expect(inputKind(FACTS[4], 'eq')).toBe('date');
  });

  it('types the values by the fact', () => {
    expect(parseValue(FACTS[0], 'between', '1, 4')).toEqual([1, 4]);
    expect(parseValue(FACTS[0], 'gte', '7')).toBe(7);
    expect(parseValue(FACTS[1], 'in', 'cycle, ttc')).toEqual(['cycle', 'ttc']);
    expect(parseValue(FACTS[3], 'eq', 'true')).toBe(true);
    expect(parseValue(FACTS[4], 'eq', '2026-10-01')).toBe('2026-10-01');
    expect(parseValue(undefined, 'eq', '3')).toBe(3);
  });

  it('emits all_of / any_of / exclude as the stored groups and exceptions', () => {
    const f = emptyConditions();
    f.allOf = [{ fact: 'mode', op: 'in', value: 'cycle, ttc' }, { fact: '', op: 'eq', value: '' }];
    f.anyOf = [[{ fact: 'days_to_period', op: 'between', value: '1, 4' }, { fact: 'on_pill', op: 'eq', value: 'false' }]];
    f.exclude = [{ fact: 'symptoms', op: 'contains', value: 'nausea' }];
    const s = toStored(f, FACTS);
    expect(s.conditions).toEqual([
      { op: 'and', items: [{ fact: 'mode', op: 'in', value: ['cycle', 'ttc'] }] },
      { op: 'or', items: [{ fact: 'days_to_period', op: 'between', value: [1, 4] }, { fact: 'on_pill', op: 'eq', value: false }] },
    ]);
    expect(s.exceptions).toEqual([{ fact: 'symptoms', op: 'contains', value: 'nausea' }]);
  });

  it('round-trips stored groups (and groups merge, or groups stay)', () => {
    const groups: Group[] = [
      { op: 'and', items: [{ fact: 'mode', op: 'eq', value: 'cycle' }] },
      { op: 'or', items: [{ fact: 'days_to_period', op: 'lte', value: 3 }] },
      { op: 'or', items: [{ fact: 'on_pill', op: 'eq', value: true }] },
    ];
    const back = toStored(fromStored(groups, []), FACTS);
    expect(back.conditions).toEqual(groups);
  });

  it('resets the operator and value when the fact type changes', () => {
    const c = onFactChange({ fact: 'days_to_period', op: 'gte', value: '3' }, 'mode', FACTS);
    expect(c.op).toBe('eq');
    expect(c.value).toBe('');
    expect(onFactChange({ fact: 'mode', op: 'in', value: 'cycle' }, 'symptoms', FACTS).op).toBe('in');
  });

  it('detects a fertility fact', () => {
    const f = emptyConditions();
    f.allOf = [{ fact: 'fertile_today', op: 'eq', value: 'true' }];
    expect(usesFertilityFact(f, ['fertile_today'])).toBe(true);
    expect(usesFertilityFact(emptyConditions(), ['fertile_today'])).toBe(false);
  });
});
