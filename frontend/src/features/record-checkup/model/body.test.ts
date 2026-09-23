import { describe, expect, it } from 'vitest';

import { selfExamResult, toCheckupRecordBody, toggleFinding } from './body';

describe('toCheckupRecordBody', () => {
  it('maps the MarkDone form to the API body', () => {
    expect(
      toCheckupRecordBody(
        { doneOn: '2026-09-16', result: 'normal', note: '  جواب خوب بود ', nextDueOn: null },
        true,
      ),
    ).toEqual({
      done_on: '2026-09-16',
      result: 'normal',
      note: 'جواب خوب بود',
      next_due_on: null,
      has_attachment: true,
    });
  });

  it('keeps a PUT partial and normalizes blanks / duplicate findings', () => {
    expect(toCheckupRecordBody({ note: '   ' })).toEqual({ note: null });
    expect(toCheckupRecordBody({ findings: ['lump', 'skin', 'lump'] })).toEqual({ findings: ['lump', 'skin'] });
    expect(toCheckupRecordBody({ nextDueOn: '' })).toEqual({ next_due_on: null });
    expect(toCheckupRecordBody({}, false)).toEqual({ has_attachment: false });
  });
});

describe('self-exam findings', () => {
  const exclusive = ['none'];

  it('the exclusive option clears the others and vice versa', () => {
    expect(toggleFinding(['lump', 'skin'], 'none', exclusive)).toEqual(['none']);
    expect(toggleFinding(['none'], 'lump', exclusive)).toEqual(['lump']);
    expect(toggleFinding(['lump'], 'skin', exclusive)).toEqual(['lump', 'skin']);
    expect(toggleFinding(['lump', 'skin'], 'lump', exclusive)).toEqual(['skin']);
  });

  it('any real finding means follow_up', () => {
    expect(selfExamResult([], exclusive)).toBe('normal');
    expect(selfExamResult(['none'], exclusive)).toBe('normal');
    expect(selfExamResult(['pain'], exclusive)).toBe('follow_up');
  });
});
