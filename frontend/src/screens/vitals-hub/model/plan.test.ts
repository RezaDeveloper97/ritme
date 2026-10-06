import { describe, expect, it } from 'vitest';

import type { PlanItem } from '@/entities/vital';

import { nextFreeItem, planIssues, toggleDay, withType, type PlanSlots } from './plan';

const slots: PlanSlots = { bp: ['morning', 'evening'], hr: ['morning', 'evening'], glucose: ['fasting', 'before_meal', 'after_meal', 'bedtime'] };
const item = (type: PlanItem['type'], slot: string, days: number[] = [0, 1]): PlanItem => ({ type, slot, days, remindAt: null });

describe('plan editor rules', () => {
  it('adds the first free type + slot', () => {
    expect(nextFreeItem([], slots)).toMatchObject({ type: 'bp', slot: 'morning' });
    expect(nextFreeItem([item('bp', 'morning')], slots)).toMatchObject({ type: 'bp', slot: 'evening' });
    const full = Object.entries(slots).flatMap(([t, ss]) => ss.map((s) => item(t as PlanItem['type'], s)));
    expect(nextFreeItem(full, slots)).toBeNull();
  });

  it('switching type picks a valid free slot', () => {
    const items = [item('bp', 'morning'), item('glucose', 'fasting')];
    expect(withType(items, 1, 'bp', slots).slot).toBe('evening');
    expect(withType(items, 0, 'glucose', slots).slot).toBe('before_meal');
  });

  it('flags duplicates and empty days', () => {
    expect(planIssues([item('bp', 'morning'), item('bp', 'morning')], slots)).toEqual({ 1: 'duplicate' });
    expect(planIssues([item('bp', 'morning', [])], slots)).toEqual({ 0: 'needDay' });
    expect(planIssues([item('hr', 'fasting')], slots)).toEqual({ 0: 'slot' });
  });

  it('toggles weekdays in order', () => {
    expect(toggleDay([0, 3], 1)).toEqual([0, 1, 3]);
    expect(toggleDay([0, 1, 3], 1)).toEqual([0, 3]);
  });
});
