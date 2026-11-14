import { describe, expect, it } from 'vitest';

import { filterFeatures, missingCapabilities, modelChoices } from './features';

const row = (over: Partial<Parameters<typeof filterFeatures>[0][number]> & { key: string }) => ({
  title: {},
  declared: true,
  is_active: true,
  usable: true,
  routes: 1,
  model_id: 1,
  ...over,
});

describe('filterFeatures', () => {
  const items = [
    row({ key: 'assistant', title: { fa: 'دستیار سلامت', en: 'Health assistant' } }),
    row({ key: 'lab_analysis', usable: false, routes: 0, model_id: null }),
    row({ key: 'voice_log', is_active: false, usable: false }),
    row({ key: 'retired', declared: false, usable: false }),
  ];

  it('searches key and every title translation', () => {
    expect(filterFeatures(items, 'دستیار', 'all').map((f) => f.key)).toEqual(['assistant']);
    expect(filterFeatures(items, 'HEALTH', 'all').map((f) => f.key)).toEqual(['assistant']);
    expect(filterFeatures(items, 'lab', 'all').map((f) => f.key)).toEqual(['lab_analysis']);
  });

  it('filters by status', () => {
    expect(filterFeatures(items, '', 'no_model').map((f) => f.key)).toEqual(['lab_analysis']);
    expect(filterFeatures(items, '', 'inactive').map((f) => f.key)).toEqual(['voice_log']);
    expect(filterFeatures(items, '', 'undeclared').map((f) => f.key)).toEqual(['retired']);
    expect(filterFeatures(items, '', 'problems').map((f) => f.key)).toEqual(['lab_analysis', 'retired']);
  });
});

describe('modelChoices', () => {
  const models = [
    { id: 1, capabilities: ['chat', 'vision', 'json'], is_active: true },
    { id: 2, capabilities: ['chat'], is_active: true },
    { id: 3, capabilities: ['chat', 'vision', 'json'], is_active: false },
  ];

  it('offers active models with every needed capability, keeping the current one', () => {
    expect(modelChoices(models, ['vision', 'json'], null).map((m) => m.id)).toEqual([1]);
    expect(modelChoices(models, ['vision', 'json'], 2).map((m) => m.id)).toEqual([1, 2]);
    expect(modelChoices(models, ['chat'], null).map((m) => m.id)).toEqual([1, 2]);
  });

  it('names the missing capabilities', () => {
    expect(missingCapabilities(models[1]!, ['chat', 'vision', 'json'])).toEqual(['vision', 'json']);
  });
});
