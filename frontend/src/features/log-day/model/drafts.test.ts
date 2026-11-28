import { beforeEach, describe, expect, it } from 'vitest';

import { draftKey, useLogDraftStore } from './drafts';

const store = () => useLogDraftStore.getState();

describe('log drafts', () => {
  beforeEach(() => store().reset());

  it('outlive the component: a closed sheet reopens with its edits', () => {
    const key = draftKey('2026-11-25');
    store().setValues(key, () => ({ mood: { feeling: ['calm'] } }));
    store().addVoice(key, ['mood.feeling']);
    expect(store().values[key]).toEqual({ mood: { feeling: ['calm'] } });
    expect(store().voice[key]).toEqual(['mood.feeling']);
  });

  it('are per day and per preset', () => {
    expect(draftKey('2026-11-25')).not.toBe(draftKey('2026-11-26'));
    expect(draftKey('2026-11-25')).not.toBe(draftKey('2026-11-25', 'menopause'));
  });

  it('clear one day after its save, and everything at reset', () => {
    const a = draftKey('2026-11-25');
    const b = draftKey('2026-11-26');
    store().setValues(a, () => ({}));
    store().setValues(b, () => ({}));
    store().clear(a);
    expect(Object.keys(store().values)).toEqual([b]);
    store().reset();
    expect(store().values).toEqual({});
  });
});
