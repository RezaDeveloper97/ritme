import { describe, expect, it } from 'vitest';

import {
  draftToInput,
  initialDraft,
  inviteGrants,
  nextStep,
  normalizeMobile,
  phoneProblem,
  previousStep,
  sameActions,
  sameChildIds,
  sameSignals,
  sharedSignals,
  toggleAction,
  toggleChildId,
} from './draft';

describe('invite wizard steps', () => {
  const partner = { spouse: false };
  const spouse = { spouse: true };

  it('runs intro → goal → scope → actions → generating → code → preview for a partner', () => {
    expect(nextStep('intro', partner)).toBe('goal');
    expect(nextStep('goal', partner)).toBe('scope');
    expect(nextStep('scope', partner)).toBe('actions');
    expect(nextStep('actions', partner)).toBe('generating');
    expect(nextStep('generating', partner)).toBe('code');
    expect(nextStep('code', partner)).toBe('preview');
  });

  it('adds the children step only for a spouse', () => {
    expect(nextStep('goal', spouse)).toBe('children');
    expect(nextStep('children', spouse)).toBe('scope');
    expect(previousStep('scope', spouse)).toBe('children');
    expect(previousStep('scope', partner)).toBe('goal');
  });

  it('never walks back past the created invite', () => {
    expect(previousStep('intro', partner)).toBeNull();
    expect(previousStep('code', partner)).toBeNull();
    expect(previousStep('preview', partner)).toBe('code');
    expect(previousStep('generating', partner)).toBe('actions');
  });

  it('starts with nothing shared, no goal and no consent', () => {
    const d = initialDraft();
    expect(d.consent).toBe(false);
    expect(d.goal).toBeNull();
    expect(Object.values(d.signals).every((on) => on === false)).toBe(true);
    expect(d.actionPrefs).toEqual([]);
  });
});

describe('normalizeMobile', () => {
  it.each([
    ['09121234567', '09121234567'],
    ['۰۹۱۲ ۱۲۳ ۴۵۶۷', '09121234567'],
    ['+98 912 123 4567', '09121234567'],
    ['00989121234567', '09121234567'],
    ['9121234567', '09121234567'],
  ])('%s → %s', (input, out) => {
    expect(normalizeMobile(input)).toBe(out);
  });

  it('rejects non-mobiles', () => {
    expect(normalizeMobile('0212345678')).toBeNull();
    expect(normalizeMobile('0912abc4567')).toBeNull();
    expect(phoneProblem('')).toBe(false);
    expect(phoneProblem('123')).toBe(true);
  });
});

describe('draftToInput', () => {
  const draft = { ...initialDraft(), consent: true, name: '  Ali ', phone: '۰۹۱۲۱۲۳۴۵۶۷' };

  it('starts with only the cycle section visible and sends the access she picked', () => {
    expect(inviteGrants()).toEqual({ cycle: 'view', symptoms: 'none', meds: 'none', appointments: 'none', pregnancy: 'none' });
    expect(draftToInput(draft)?.grants).toEqual(inviteGrants());
    const picked = { ...draft, grants: { ...draft.grants, cycle: 'none' as const, meds: 'edit' as const } };
    expect(draftToInput(picked)?.grants).toEqual({
      cycle: 'none',
      symptoms: 'none',
      meds: 'edit',
      appointments: 'none',
      pregnancy: 'none',
    });
  });

  it('sends a partner with the normalized phone and trimmed name', () => {
    expect(draftToInput(draft)).toMatchObject({ type: 'partner', displayName: 'Ali', phone: '09121234567' });
  });

  it('leaves the phone out when none was typed', () => {
    expect(draftToInput({ ...draft, phone: '' })).not.toHaveProperty('phone');
    expect(draftToInput({ ...draft, name: ' ' })).not.toHaveProperty('displayName');
  });

  it('refuses without consent or with a bad phone', () => {
    expect(draftToInput({ ...draft, consent: false })).toBeNull();
    expect(draftToInput({ ...draft, phone: '123' })).toBeNull();
  });

  it('sends the preferences, defaulting the goal to unknown', () => {
    expect(draftToInput(draft)?.preferences).toEqual({
      goal: 'unknown',
      signals: { pain: false, mood: false, energy: false, sleep: false },
      actionPrefs: [],
    });
    const picked = {
      ...draft,
      goal: 'support' as const,
      signals: { ...draft.signals, mood: true },
      actionPrefs: ['time' as const, 'food' as const],
    };
    expect(draftToInput(picked)?.preferences).toEqual({
      goal: 'support',
      signals: { pain: false, mood: true, energy: false, sleep: false },
      actionPrefs: ['time', 'food'],
    });
  });

  it('makes a spouse and shares the picked children only then', () => {
    expect(draftToInput({ ...draft, spouse: true, childIds: [3, 1] })).toMatchObject({ type: 'spouse', childIds: [3, 1] });
    expect(draftToInput({ ...draft, spouse: true, childIds: [] })).not.toHaveProperty('childIds');
    expect(draftToInput({ ...draft, spouse: false, childIds: [3] })).not.toHaveProperty('childIds');
  });
});

describe('selection helpers', () => {
  it('toggles child ids', () => {
    expect(toggleChildId([], 4)).toEqual([4]);
    expect(toggleChildId([4, 2], 4)).toEqual([2]);
    expect(toggleChildId([2], 4)).toEqual([2, 4]);
  });

  it('keeps the picked order of actions', () => {
    expect(toggleAction([], 'gift')).toEqual(['gift']);
    expect(toggleAction(['gift'], 'food')).toEqual(['gift', 'food']);
    expect(toggleAction(['gift', 'food'], 'gift')).toEqual(['food']);
  });

  it('compares child sets regardless of order', () => {
    expect(sameChildIds([1, 2], [2, 1])).toBe(true);
    expect(sameChildIds([1], [1, 2])).toBe(false);
    expect(sameChildIds([], [])).toBe(true);
  });

  it('compares actions in order and signals by key', () => {
    expect(sameActions(['food', 'gift'], ['food', 'gift'])).toBe(true);
    expect(sameActions(['food', 'gift'], ['gift', 'food'])).toBe(false);
    const off = initialDraft().signals;
    expect(sameSignals(off, { ...off })).toBe(true);
    expect(sameSignals(off, { ...off, pain: true })).toBe(false);
  });

  it('lists the shared signals in canonical order, mood never offered', () => {
    // mood stays «never shared» (SPEC §1.7): even a stored true never reaches the preview.
    expect(sharedSignals({ pain: true, mood: true, energy: false, sleep: true })).toEqual(['pain', 'sleep']);
  });
});
