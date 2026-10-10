import { describe, expect, it } from 'vitest';

import { homeNudges } from './home-nudges';
import type { ConditionNudge } from './types';

const heavy: ConditionNudge = {
  key: 'heavy_bleeding',
  program: 'heavy_bleeding',
  title: 'T',
  body: 'B',
  action: 'Open',
  link: '/programs/bleeding',
  doctor: null,
  days: 3,
};

describe('homeNudges', () => {
  it('shows nothing without a hit', () => {
    expect(homeNudges(undefined, undefined)).toEqual([]);
    expect(homeNudges([], undefined)).toEqual([]);
  });
  it('shows the hit for a user who is not enrolled', () => {
    expect(homeNudges([heavy], { programs: [{ code: 'heavy_bleeding', enrolled: false, enrolledOn: null }] })).toEqual([heavy]);
  });
  it('hides it once she is enrolled in that program', () => {
    expect(homeNudges([heavy], { programs: [{ code: 'heavy_bleeding', enrolled: true, enrolledOn: '2026-10-01' }] })).toEqual([]);
  });
});
