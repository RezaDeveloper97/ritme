import { describe, expect, it } from 'vitest';

import type { CheckupRecord } from '@/entities/checkup';

import { editRecordSheetArg, parseTypeParam, withLocalAttachment } from './view';

const rec = (id: number, hasAttachment: boolean): CheckupRecord => ({
  id,
  checkupTypeId: 3,
  checkupTitle: 'x',
  checkupKey: null,
  checkupIcon: null,
  checkupTone: 'neutral',
  doneOn: '2026-01-01',
  result: 'normal',
  findings: [],
  note: null,
  hasAttachment,
  nextDueOn: null,
});

describe('checkup-history view', () => {
  it('parses ?type=', () => {
    expect(parseTypeParam('12')).toBe(12);
    expect(parseTypeParam('0')).toBeNull();
    expect(parseTypeParam('a1')).toBeNull();
    expect(parseTypeParam(undefined)).toBeNull();
  });

  it('keeps only records whose file is on this device', () => {
    const list = [rec(1, true), rec(2, true), rec(3, false)];
    expect(withLocalAttachment(list, new Set([2])).map((r) => r.id)).toEqual([2]);
    expect(withLocalAttachment(list, undefined).map((r) => r.id)).toEqual([1, 2]);
  });

  it('builds the edit sheet arg', () => {
    expect(editRecordSheetArg({ checkupTypeId: 3, id: 9 })).toBe('3-9');
  });
});
