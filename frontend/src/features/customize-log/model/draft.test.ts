import { describe, expect, it } from 'vitest';

import type { LogPreferences } from '@/entities/health-log';

import {
  applyDraft,
  diffDraft,
  draftFromPrefs,
  isDirty,
  isPinned,
  labelProblem,
  moveCategory,
  normalizeLabel,
  pinsFull,
  togglePin,
  toggleHidden,
  type CustomizeDraft,
} from './draft';

function prefs(): LogPreferences {
  const cat = (code: string, hidden = false, pinned = false) => ({ code, label: code, hidden, pinned, customParam: null });
  return {
    mode: 'cycle',
    phase: 'luteal',
    isDefault: true,
    maxPinned: 8,
    pinned: ['bleeding', 'pain', 'measurements.weight'],
    categories: [cat('bleeding', false, true), cat('pain', false, true), cat('mood'), cat('measurements'), cat('sex', true)],
    maxCustomItems: 20,
    customItems: [],
  };
}

describe('draftFromPrefs', () => {
  it('takes order, hidden and tile keys', () => {
    expect(draftFromPrefs(prefs())).toEqual({
      order: ['bleeding', 'pain', 'mood', 'measurements', 'sex'],
      hidden: ['sex'],
      pinned: ['bleeding', 'pain', 'measurements.weight'],
    });
  });
});

describe('moveCategory', () => {
  const d = draftFromPrefs(prefs());
  it('moves and clamps', () => {
    expect(moveCategory(d, 0, 2).order).toEqual(['pain', 'mood', 'bleeding', 'measurements', 'sex']);
    expect(moveCategory(d, 4, -3).order[0]).toBe('sex');
    expect(moveCategory(d, 1, 99).order.at(-1)).toBe('pain');
  });
  it('returns the same draft for a no-op', () => {
    expect(moveCategory(d, 2, 2)).toBe(d);
    expect(moveCategory(d, 9, 0)).toBe(d);
  });
});

describe('pins', () => {
  const p = prefs();
  const base = draftFromPrefs(p);

  it('sees a param tile as its category pinned', () => {
    expect(isPinned(base, 'measurements')).toBe(true);
    expect(isPinned(base, 'mood')).toBe(false);
  });

  it('unpins every key of the category and restores the server key on re-pin', () => {
    const off = togglePin(base, 'measurements', p.pinned, 8);
    expect(off.pinned).toEqual(['bleeding', 'pain']);
    const on = togglePin(off, 'measurements', p.pinned, 8);
    expect(on.pinned).toEqual(['bleeding', 'pain', 'measurements.weight']);
  });

  it('pins a new category in list order', () => {
    expect(togglePin(base, 'mood', p.pinned, 8).pinned).toEqual(['bleeding', 'pain', 'mood', 'measurements.weight']);
  });

  it('refuses a pin past the cap', () => {
    const full: CustomizeDraft = { ...base, pinned: ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'] };
    expect(pinsFull(full, 8)).toBe(true);
    expect(togglePin(full, 'mood', p.pinned, 8)).toBe(full);
    expect(pinsFull(base, 8)).toBe(false);
  });
});

describe('diffDraft', () => {
  const p = prefs();
  const base = draftFromPrefs(p);

  it('is empty without changes', () => {
    expect(diffDraft(base, base)).toEqual({});
    expect(isDirty(base, base)).toBe(false);
  });

  it('sends only the lists that changed', () => {
    expect(diffDraft(base, moveCategory(base, 0, 1))).toEqual({ order: ['pain', 'bleeding', 'mood', 'measurements', 'sex'] });
    expect(diffDraft(base, toggleHidden(base, 'mood'))).toEqual({ hidden: ['mood', 'sex'] });
    expect(diffDraft(base, togglePin(base, 'pain', p.pinned, 8))).toEqual({ pinned: ['bleeding', 'measurements.weight'] });
  });

  it('compares hidden as a set', () => {
    const reordered = { ...base, hidden: ['sex'] };
    const twice = toggleHidden(toggleHidden(base, 'mood'), 'mood');
    expect(diffDraft(base, reordered)).toEqual({});
    expect(diffDraft(base, twice)).toEqual({});
  });
});

describe('applyDraft', () => {
  it('reorders, hides and drops hidden tiles', () => {
    const p = prefs();
    const d = toggleHidden(moveCategory(draftFromPrefs(p), 1, 0), 'bleeding');
    const next = applyDraft(p, d);
    expect(next.categories.map((c) => c.code)).toEqual(['pain', 'bleeding', 'mood', 'measurements', 'sex']);
    expect(next.categories[1]).toMatchObject({ hidden: true, pinned: false });
    expect(next.pinned).toEqual(['pain', 'measurements.weight']);
    expect(next.isDefault).toBe(false);
  });
});

describe('labels', () => {
  it('normalizes whitespace', () => {
    expect(normalizeLabel('  قهوه   تلخ ')).toBe('قهوه تلخ');
  });
  it('flags empty, too long and duplicate labels', () => {
    expect(labelProblem('   ', [])).toBe('empty');
    expect(labelProblem('x'.repeat(41), [])).toBe('tooLong');
    expect(labelProblem('x'.repeat(40), [])).toBeNull();
    expect(labelProblem(' Coffee ', ['coffee'])).toBe('duplicate');
    expect(labelProblem('Tea', ['coffee'])).toBeNull();
  });
});
