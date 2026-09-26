import { describe, expect, it } from 'vitest';

import { markDoneArg, parseMarkDoneArg } from './arg';

describe('mark-done sheet arg', () => {
  it('round-trips new and edit targets', () => {
    expect(parseMarkDoneArg(markDoneArg(7))).toEqual({ typeId: 7, recordId: null });
    expect(parseMarkDoneArg(markDoneArg(7, 42))).toEqual({ typeId: 7, recordId: 42 });
  });
  it('rejects junk', () => {
    expect(parseMarkDoneArg(undefined)).toBeNull();
    expect(parseMarkDoneArg('0')).toBeNull();
    expect(parseMarkDoneArg('a-1')).toBeNull();
    expect(parseMarkDoneArg('3-0')).toBeNull();
  });
});
