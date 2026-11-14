import { describe, expect, it } from 'vitest';

import type { CreditPack } from '../model/types';

import { defaultPack, packFromParam } from './packs';

const pack = (id: number, isHighlighted = false): CreditPack => ({ id, code: `p${id}`, title: 't', badge: null, credits: id * 50, price: id * 1000, isHighlighted });

describe('credit packs', () => {
  it('starts on the highlighted pack, else the first', () => {
    expect(defaultPack([pack(1), pack(2, true), pack(3)])?.id).toBe(2);
    expect(defaultPack([pack(1), pack(3)])?.id).toBe(1);
    expect(defaultPack([])).toBeNull();
  });
  it('reads ?pack= strictly', () => {
    const packs = [pack(1), pack(2)];
    expect(packFromParam(packs, '2')?.id).toBe(2);
    expect(packFromParam(packs, '9')).toBeNull();
    expect(packFromParam(packs, '2abc')).toBeNull();
    expect(packFromParam(packs, null)).toBeNull();
  });
});
