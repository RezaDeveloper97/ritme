import { describe, expect, it } from 'vitest';

import type { SimulationV6 } from '../api/rules';
import { SIM_REASONS, engineSummary, parseUserRef, reasonKey, toRows } from './simulator';

const sim = {
  source: 'synthetic', at: '', date: '2026-11-01', time: '10:00:00', zone: 'Asia/Tehran', history: 'empty', facts_overridden: [],
  engine: { main_phase: 'luteal', cycle_day: 21, warnings: [], confidence: null },
  notes: [],
  slots: [
    { slot: 'push', capacity: 1, winners: [], dropped: [{ rule_id: 3, title: 'x', kind: 'product', priority: 5, reason: 'cap_day' }] },
    { slot: 'smart_tip', capacity: 1, winners: [{ rule_id: 1, rule_key: null, title: 'tip', kind: 'engine', level: 'info', priority: 1, version: 2, variant: 'b', message_title: 'T', message_body: 'B', cta_label: null }], dropped: [{ rule_id: 2, title: 'y', kind: 'engine', priority: 1, reason: 'brand_new_reason' }] },
  ],
} as unknown as SimulationV6;

describe('simulator view-model', () => {
  it('maps every known reason to a label key and unknown ones to «unknown»', () => {
    for (const r of SIM_REASONS) expect(reasonKey(r)).toBe(r);
    expect(reasonKey('brand_new_reason')).toBe('unknown');
  });
  it('puts slots with a winner first and keeps winners, variants and drop reasons', () => {
    const rows = toRows(sim);
    expect(rows.map((r) => r.slot)).toEqual(['smart_tip', 'push']);
    expect(rows[0]?.winners[0]).toMatchObject({ variant: 'b', version: 2, messageTitle: 'T' });
    expect(rows[0]?.dropped[0]?.reasonKey).toBe('unknown');
    expect(rows[1]?.dropped[0]?.reasonKey).toBe('cap_day');
  });
  it('summarises the engine without empty values', () => {
    expect(engineSummary(sim.engine)).toEqual([
      { key: 'main_phase', value: 'luteal' },
      { key: 'cycle_day', value: '21' },
    ]);
  });
  it('reads a user as id or RT code', () => {
    expect(parseUserRef('')).toEqual({ kind: 'none' });
    expect(parseUserRef(' 42 ')).toEqual({ kind: 'id', id: 42 });
    expect(parseUserRef('rt-ab12cd')).toEqual({ kind: 'code', code: 'RT-AB12CD' });
    expect(parseUserRef('hello')).toEqual({ kind: 'invalid' });
  });
});
