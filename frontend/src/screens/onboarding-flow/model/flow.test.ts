import { describe, expect, it } from 'vitest';

import { isOnboardingStep, onboardingRoute } from '@/entities/user';

import { FLOW_ROUTES, flowStepFromPath, landingRoute, nextStep, previousRoute, RESUME_KEYS } from './flow';
import { toggleListItem } from './state';

describe('onboarding v2 flow', () => {
  it('walks the women path through each goal branch', () => {
    expect(nextStep('name', {})).toBe('gender');
    expect(nextStep('gender', { gender: 'female' })).toBe('goal');
    expect(nextStep('goal', { goal: 'cycle' })).toBe('cycle');
    expect(nextStep('goal', { goal: 'ttc' })).toBe('cycle');
    expect(nextStep('goal', { goal: 'pregnancy' })).toBe('pregnancy');
    expect(nextStep('goal', { goal: 'menopause' })).toBe('menopause');
    for (const branch of ['cycle', 'pregnancy', 'menopause'] as const) expect(nextStep(branch, {})).toBe('conditions');
    expect(nextStep('conditions', {})).toBe('health');
    expect(nextStep('health', {})).toBe('ready');
  });

  it('sends a man to the partner-code stub, then Ready', () => {
    expect(nextStep('gender', { gender: 'male' })).toBe('partner');
    expect(nextStep('partner', { gender: 'male' })).toBe('ready');
    expect(previousRoute('partner', null)).toBe(FLOW_ROUTES.gender);
  });

  it('backs out to known routes, branch-aware', () => {
    expect(previousRoute('name', null)).toBe('/signup');
    expect(previousRoute('conditions', 'pregnancy')).toBe(FLOW_ROUTES.pregnancy);
    expect(previousRoute('conditions', 'menopause')).toBe(FLOW_ROUTES.menopause);
    expect(previousRoute('conditions', null)).toBe(FLOW_ROUTES.cycle);
    expect(previousRoute('health', 'ttc')).toBe(FLOW_ROUTES.conditions);
  });

  it('lands each goal on its home', () => {
    expect(landingRoute('cycle', false)).toBe('/home');
    expect(landingRoute('ttc', false)).toBe('/home');
    expect(landingRoute('menopause', false)).toBe('/home');
    expect(landingRoute('pregnancy', true)).toBe('/pregnancy');
    expect(landingRoute('pregnancy', false)).toBe('/pregnancy/onboarding');
    expect(landingRoute(null, false)).toBe('/home');
    expect(landingRoute(null, false, 'male')).toBe('/companion');
    expect(landingRoute('cycle', false, 'male')).toBe('/companion');
    expect(landingRoute('cycle', false, 'female')).toBe('/home');
  });

  it('maps every screen to a resume key the middleware understands', () => {
    for (const key of Object.values(RESUME_KEYS)) expect(isOnboardingStep(key)).toBe(true);
    // The resume key leads back to the screen itself, or to a legacy route that redirects to it.
    expect(onboardingRoute(RESUME_KEYS.name as never)).toBe(FLOW_ROUTES.name);
    expect(onboardingRoute(RESUME_KEYS.goal as never)).toBe(FLOW_ROUTES.goal);
    expect(onboardingRoute(RESUME_KEYS.pregnancy as never)).toBe(FLOW_ROUTES.pregnancy);
    expect(onboardingRoute(RESUME_KEYS.conditions as never)).toBe(FLOW_ROUTES.conditions);
  });

  it('reads the step from a locale-prefixed path', () => {
    expect(flowStepFromPath('/fa/onboarding/cycle')).toBe('cycle');
    expect(flowStepFromPath('/en/onboarding/intention')).toBe('goal');
    expect(flowStepFromPath('/fa/onboarding/setting-up')).toBe('ready');
    expect(flowStepFromPath('/fa/onboarding/weight')).toBeNull();
    expect(flowStepFromPath('/fa/home')).toBeNull();
  });
});

describe('toggleListItem', () => {
  it('keeps «none» exclusive', () => {
    expect(toggleListItem(null, 'none')).toEqual([]);
    expect(toggleListItem([], 'pcos')).toEqual(['pcos']);
    expect(toggleListItem(['pcos', 'other'], 'pcos')).toEqual(['other']);
    expect(toggleListItem(['pcos'], 'none')).toEqual([]);
    expect(toggleListItem(null, 'iud')).toEqual(['iud']);
  });
});
