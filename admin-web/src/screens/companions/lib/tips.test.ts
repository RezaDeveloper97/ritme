import { describe, expect, it } from 'vitest';

import { draftOf, fillName, move, sameDraft, tipsBody } from './tips';

describe('companion tips form', () => {
  const texts = {
    customized: false,
    note: { body: '{name} is tired.', source: 'built_in' as const },
    tips: [
      { slot: 1, title: 'Tea', body: 'Make tea', source: 'built_in' as const },
      { slot: 2, title: '', body: 'hidden', source: 'built_in' as const },
      { slot: 3, title: 'Walk', body: '', source: 'built_in' as const },
    ],
  };

  it('drops hidden slots from the draft', () => {
    expect(draftOf(texts)).toEqual({
      note: '{name} is tired.',
      tips: [
        { title: 'Tea', body: 'Make tea' },
        { title: 'Walk', body: '' },
      ],
    });
    expect(draftOf(undefined)).toEqual({ note: '', tips: [] });
  });

  it('builds the PUT body: trimmed, blanks to null, empty tips dropped', () => {
    const body = tipsBody(
      {
        fa: { note: '  ', tips: [{ title: ' چای ', body: ' ' }, { title: '', body: '' }] },
        en: { note: 'Hi {name}', tips: [] },
      },
      ['fa', 'en', 'ar'],
    );
    expect(body).toEqual({
      texts: {
        fa: { note: null, tips: [{ title: 'چای', body: null }] },
        en: { note: 'Hi {name}', tips: [] },
      },
    });
  });

  it('compares drafts ignoring outer spaces', () => {
    expect(sameDraft({ note: 'a ', tips: [{ title: 'b', body: '' }] }, { note: 'a', tips: [{ title: ' b', body: ' ' }] })).toBe(true);
    expect(sameDraft({ note: 'a', tips: [] }, { note: 'a', tips: [{ title: 'b', body: '' }] })).toBe(false);
  });

  it('fills the sample name and reorders', () => {
    expect(fillName('{name} و {name}', 'سارا')).toBe('سارا و سارا');
    expect(move(['a', 'b', 'c'], 2, -1)).toEqual(['a', 'c', 'b']);
    expect(move(['a', 'b'], 0, -1)).toEqual(['a', 'b']);
  });
});
