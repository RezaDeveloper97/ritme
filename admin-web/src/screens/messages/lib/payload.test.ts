import { describe, expect, it } from 'vitest';

import { fieldKind, fromDraft, previewOf, toDraft } from './payload';

const payload = { title: 'سلام', body: 'x'.repeat(61), dos: ['آب بنوشید', 'بخوابید'] };

describe('smart-message payload editor', () => {
  it('picks the control by shape and length', () => {
    expect(fieldKind(payload.title)).toBe('line');
    expect(fieldKind(payload.body)).toBe('text');
    expect(fieldKind(payload.dos)).toBe('list');
  });

  it('round-trips lists as lines, trimming and dropping blanks', () => {
    const draft = toDraft(payload);
    expect(draft.dos).toBe('آب بنوشید\nبخوابید');
    expect(fromDraft(payload, { ...draft, dos: ' a \n\n b\r\n' })).toEqual({ title: 'سلام', body: payload.body, dos: ['a', 'b'] });
  });

  it('previews the first value', () => {
    expect(previewOf({ dos: ['a', 'b'] }, '، ')).toBe('a، b');
    expect(previewOf({}, ', ')).toBe('');
  });
});
