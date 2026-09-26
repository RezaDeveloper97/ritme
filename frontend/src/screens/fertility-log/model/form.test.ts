import { describe, expect, it } from 'vitest';

import type { FertilityDay } from '@/entities/fertility';
import { toFertilityDayBody } from '@/features/log-fertility-day';

import { changedInput, checkBbt, fromDay, isDirty, parseFocus, parseLogDate, toggleSymptom } from './form';

const day: FertilityDay = {
  date: '2026-09-20',
  cycleDay: 12,
  lh: 'negative',
  mucus: 'creamy',
  bbt: 36.4,
  bbtTime: null,
  intercourse: null,
  symptoms: ['bloating'],
  note: 'hi',
  chance: { level: 'high', label: null, bars: 4 },
};
const fmt = (v: number) => v.toFixed(2);

describe('fertility log form', () => {
  it('round-trips: load → edit → save payload has only changed fields', () => {
    const loaded = fromDay(day, fmt);
    expect(isDirty(day, loaded)).toBe(false);
    expect(changedInput(day, loaded)).toEqual({});

    const edited = {
      ...loaded,
      lh: 'positive' as const,
      bbtText: '۳۶٫۶۱',
      symptoms: toggleSymptom(toggleSymptom(loaded.symptoms, 'spotting'), 'ovarian_pain'),
      note: '  ',
    };
    expect(isDirty(day, edited)).toBe(true);
    const input = changedInput(day, edited);
    expect(toFertilityDayBody(input!)).toEqual({
      lh: 'positive',
      bbt: 36.61,
      symptoms: ['ovarian_pain', 'bloating', 'spotting'],
      note: null,
    });
  });

  it('clearing chips sends explicit null', () => {
    const s = { ...fromDay(day, fmt), lh: null, bbtText: '' };
    expect(changedInput(day, s)).toEqual({ lh: null, bbt: null });
  });

  it('rejects out-of-range and junk BBT', () => {
    expect(checkBbt('39')).toEqual({ ok: false, reason: 'range' });
    expect(checkBbt('abc')).toEqual({ ok: false, reason: 'invalid' });
    expect(changedInput(day, { ...fromDay(day, fmt), bbtText: '34.9' })).toBeNull();
  });

  it('parses query params', () => {
    expect(parseFocus('bbt')).toBe('bbt');
    expect(parseFocus('x')).toBeNull();
    expect(parseLogDate('2026-09-01', '2026-09-26')).toBe('2026-09-01');
    expect(parseLogDate('2027-01-01', '2026-09-26')).toBe('2026-09-26');
    expect(parseLogDate('bad', '2026-09-26')).toBe('2026-09-26');
  });
});
