import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { fertilityCycleSchema, ttcHubSchema } from '../api/ttc';
import { barShares, dayX, lhCells, linePath, tempScale, tempY, tryingShare, type ChartBox } from './ttc';

const golden = (name: string): unknown =>
  JSON.parse(readFileSync(resolve(__dirname, '../../../../../backend-go/internal/analysis/testdata/golden', `${name}.json`), 'utf8'));

describe('TTC schemas against the Go engine goldens', () => {
  it('parses the Plus hub', () => {
    const hub = ttcHubSchema.parse(golden('ttc_plus_hub'));
    expect(hub.trying.cycles).toBe(4);
    expect(hub.trying.referral).toEqual({ ageBand: 'under_35', thresholdMonths: 12, due: false });
    expect(hub.bbt.data?.points[0]).toEqual({ day: 1, value: 36.35 });
    expect(hub.lh.data?.positiveDay).toBe(13);
    expect(hub.timing.data?.inWindow).toBe(2);
    expect(hub.luteal.data).toEqual({ days: 14, status: 'normal', cycles: 2 });
    expect(hub.regularity.data?.status).toBe('regular');
    expect(hub.cycles.map((c) => c.ovulationSource)).toEqual(['bbt', 'lh', 'bbt', 'lh']);
  });

  it('parses the free hub with locked Plus cards', () => {
    const hub = ttcHubSchema.parse(golden('ttc_free_hub'));
    expect(hub.timing).toEqual({ plus: true, locked: true, ready: false, data: null });
    expect(hub.mucus.locked).toBe(true);
  });

  it('parses the empty and no-log hubs', () => {
    expect(ttcHubSchema.parse(golden('ttc_empty_hub')).cycles).toEqual([]);
    const nologs = ttcHubSchema.parse(golden('ttc_nologs_hub'));
    expect(nologs.trying.since).toBeNull();
    expect(nologs.trying.referral.ageBand).toBe('unknown');
  });

  it('parses the fertility detail', () => {
    const f = fertilityCycleSchema.parse(golden('ttc_plus_fertility_first.en'));
    expect(f.cycle).toMatchObject({ index: 1, total: 4, current: false, length: 28 });
    expect(f.chart.highDays).toEqual([15, 16, 17]);
    expect(f.chart.coverline).toBe(36.35);
    expect(f.ovulation).toEqual({ day: 14, source: 'bbt', confirmed: true });
    expect(f.lh.daysBeforeOvulation).toBe(1);
    expect(f.luteal.data).toEqual({ days: 14, status: 'normal' });
    expect(f.prevStart).toBeNull();
  });
});

describe('lhCells', () => {
  it('centres on the positive test', () => {
    const cells = lhCells(
      [
        { day: 11, value: 'negative' },
        { day: 12, value: 'faint' },
        { day: 13, value: 'positive' },
      ],
      13,
    );
    expect(cells.map((c) => c.day)).toEqual([9, 10, 11, 12, 13, 14, 15]);
    expect(cells.map((c) => c.value)).toEqual([null, null, 'negative', 'faint', 'positive', null, null]);
  });
  it('ends on the last test without a positive and never starts before day 1', () => {
    expect(lhCells([{ day: 3, value: 'negative' }], null)[0].day).toBe(1);
    expect(lhCells([{ day: 12, value: 'negative' }], null).map((c) => c.day)).toEqual([6, 7, 8, 9, 10, 11, 12]);
    expect(lhCells([], null)).toEqual([]);
  });
});

describe('chart geometry', () => {
  const box: ChartBox = { width: 320, height: 200, left: 30, right: 310, top: 10, bottom: 190 };
  it('snaps the scale to 0.2 and keeps it at least 0.6 tall', () => {
    const s = tempScale([36.25, 36.3, 36.75]);
    expect(s.min).toBe(36.2);
    expect(s.max).toBe(36.8);
    expect(s.ticks).toEqual([36.2, 36.4, 36.6, 36.8]);
    const flat = tempScale([36.5]);
    expect(flat.max - flat.min).toBeGreaterThanOrEqual(0.6 - 1e-9);
  });
  it('maps days left → right and temperatures bottom → top', () => {
    expect(dayX(1, 28, box)).toBe(30);
    expect(dayX(28, 28, box)).toBe(310);
    const s = tempScale([36.2, 36.8]);
    expect(tempY(s.min, s, box)).toBe(190);
    expect(tempY(s.max, s, box)).toBe(10);
    expect(linePath([{ day: 1, value: 36.2 }], 28, s, box)).toMatch(/^M30 /);
  });
  it('shares and ring', () => {
    expect(barShares([28, 14])).toEqual([1, 0.5]);
    expect(tryingShare(3, 12)).toBe(0.25);
    expect(tryingShare(20, 6)).toBe(1);
  });
});
