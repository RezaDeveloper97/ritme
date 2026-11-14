import { describe, expect, it } from 'vitest';

import type { CompanionSignals } from '@/entities/companion';

import { statusCopy } from './status';

const none: CompanionSignals = { pain: null, mood: null, energy: null, sleep: null };

describe('statusCopy', () => {
  it('is the phase sentence alone when nothing is shared', () => {
    expect(statusCopy('luteal', none)).toEqual({ phase: 'luteal', signal: null });
  });

  it('ignores signals that are shared but off today', () => {
    expect(statusCopy('follicular', { pain: false, mood: false, energy: false, sleep: false }).signal).toBeNull();
  });

  it('adds exactly one signal, physical ones first', () => {
    expect(statusCopy('menstrual', { pain: true, mood: true, energy: true, sleep: null }).signal).toBe('pain');
    expect(statusCopy('general', { pain: false, mood: true, energy: true, sleep: true }).signal).toBe('energy');
    expect(statusCopy('fertile', { ...none, mood: true }).signal).toBe('mood');
  });

  it('keeps the phase as given (pregnancy and general included)', () => {
    expect(statusCopy('pregnancy', { ...none, sleep: true })).toEqual({ phase: 'pregnancy', signal: 'sleep' });
  });
});
