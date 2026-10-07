import { describe, expect, it } from 'vitest';

import { DEFAULT_GROUPS, isValidFrom, parseGroup, preselectedGroups, sectionsOf, selectionOf, sharedReportUrl } from './selection';

describe('report selection', () => {
  it('maps the artboard defaults to API sections in screen order', () => {
    expect(sectionsOf(DEFAULT_GROUPS)).toEqual(['basics', 'conditions', 'medications', 'allergies', 'cycle', 'vitals']);
  });

  it('adds checkups + labs and pregnancies when switched on', () => {
    const s = sectionsOf({ ...DEFAULT_GROUPS, basics: false, health: false, checkups: true, pregnancies: true });
    expect(s).toEqual(['cycle', 'vitals', 'pregnancies', 'checkups', 'labs']);
  });

  it('drops `from` unless the range is custom', () => {
    expect(selectionOf({ range: '6m', from: '2026-01-01', groups: DEFAULT_GROUPS, question: 'q' })).toMatchObject({
      range: '6m',
      question: 'q',
    });
  });

  it('accepts a custom start before today and at most 3 years back', () => {
    expect(isValidFrom('2026-09-01', '2026-10-06')).toBe(true);
    expect(isValidFrom('2023-10-06', '2026-10-06')).toBe(true);
    expect(isValidFrom('2023-10-05', '2026-10-06')).toBe(false);
    expect(isValidFrom('2026-10-06', '2026-10-06')).toBe(false);
    expect(isValidFrom(null, '2026-10-06')).toBe(false);
  });

  it('builds the public URL from the token only', () => {
    expect(sharedReportUrl('https://stage.ritmeapp.ir', 'fa', 'abc')).toBe('https://stage.ritmeapp.ir/fa/shared/report/abc');
  });

  it('preselects the entry point group with the basics only (B-N6-04b)', () => {
    expect(parseGroup('vitals')).toBe('vitals');
    expect(parseGroup('checkups')).toBe('checkups');
    expect(parseGroup('labs')).toBeNull();
    expect(parseGroup(null)).toBeNull();
    expect(sectionsOf(preselectedGroups('vitals'))).toEqual(['basics', 'vitals']);
    expect(sectionsOf(preselectedGroups('checkups'))).toEqual(['basics', 'checkups', 'labs']);
  });
});
