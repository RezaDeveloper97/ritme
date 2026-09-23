import { describe, expect, it } from 'vitest';

import {
  applyDoneKeys,
  clampV2Week,
  datingPreviewBody,
  defaultHighlightTone,
  highlightGlyph,
  isDatingPreviewReady,
  isV2Week,
  moodGlyph,
  taskProgress,
  toggleDoneKey,
  weekRelation,
} from './v2';
import { HIGHLIGHT_ICONS, MOODS } from './v2-types';

const tasks = [
  { key: 'folic', text: 'a', done: true },
  { key: 'book_nt', text: 'b', done: false },
  { key: 'blood', text: 'c', done: false },
];

describe('week helpers', () => {
  it('bounds weeks to 1–42', () => {
    expect([0, 1, 42, 43, 8.5].map(isV2Week)).toEqual([false, true, true, false, false]);
    expect([-3, 0, 8.4, 50, Number.NaN].map(clampV2Week)).toEqual([1, 1, 8, 42, 1]);
  });

  it('classifies chips relative to the current week', () => {
    expect(weekRelation(7, 8)).toBe('past');
    expect(weekRelation(8, 8)).toBe('current');
    expect(weekRelation(9, 8)).toBe('future');
    expect(weekRelation(9, null)).toBe('future');
  });
});

describe('task helpers', () => {
  it('counts done tasks («۲ از ۳»)', () => {
    expect(taskProgress(tasks)).toEqual({ done: 1, total: 3 });
    expect(taskProgress([])).toEqual({ done: 0, total: 0 });
  });

  it('toggles a key without duplicates and keeps order', () => {
    expect(toggleDoneKey(['folic'], 'book_nt')).toEqual(['folic', 'book_nt']);
    expect(toggleDoneKey(['folic', 'book_nt'], 'folic')).toEqual(['book_nt']);
  });

  it('applies a done list to a task list', () => {
    expect(applyDoneKeys(tasks, ['blood']).map((t) => t.done)).toEqual([false, false, true]);
  });
});

describe('glyphs', () => {
  it('has a glyph and default tone for every admin-pickable icon', () => {
    for (const icon of HIGHLIGHT_ICONS) {
      expect(highlightGlyph(icon)).toBeTruthy();
      expect(['brand', 'pink', 'teal']).toContain(defaultHighlightTone(icon));
    }
    expect(highlightGlyph(null)).toBe('sparkle');
    expect(defaultHighlightTone('heart')).toBe('pink');
    expect(defaultHighlightTone('eye')).toBe('teal');
  });

  it('maps every mood to a distinct face, best first', () => {
    const faces = MOODS.map(moodGlyph);
    expect(new Set(faces).size).toBe(5);
    expect(moodGlyph(5)).toBe('faceGreat');
    expect(moodGlyph(1)).toBe('faceHard');
  });
});

describe('dating preview input', () => {
  it('sends only the chosen source fields', () => {
    expect(
      datingPreviewBody({ source: 'lmp', lmp_date: '2026-07-25', manual_weeks: 8, ultrasound_date: 'x' }),
    ).toEqual({ source: 'lmp', lmp_date: '2026-07-25' });
    expect(datingPreviewBody({ source: 'ultrasound', ultrasound_date: '2026-09-16', ultrasound_weeks: 7 })).toEqual({
      source: 'ultrasound',
      ultrasound_date: '2026-09-16',
      ultrasound_weeks: 7,
      ultrasound_days: 0,
    });
    expect(datingPreviewBody({ source: 'manual', manual_weeks: 8, manual_days: 3 })).toEqual({
      source: 'manual',
      manual_weeks: 8,
      manual_days: 3,
    });
  });

  it('is ready only when the chosen source is complete', () => {
    expect(isDatingPreviewReady(null)).toBe(false);
    expect(isDatingPreviewReady({ source: 'lmp' })).toBe(false);
    expect(isDatingPreviewReady({ source: 'lmp', lmp_date: '2026-07-25' })).toBe(true);
    expect(isDatingPreviewReady({ source: 'ultrasound', ultrasound_date: '2026-09-16' })).toBe(false);
    expect(isDatingPreviewReady({ source: 'ultrasound', ultrasound_date: '2026-09-16', ultrasound_weeks: 0 })).toBe(true);
    expect(isDatingPreviewReady({ source: 'manual', manual_weeks: 8 })).toBe(true);
  });
});
