import { describe, expect, it } from 'vitest';

import { checkoutBlockerKey, maskPhone } from './checkout';

describe('checkout helpers', () => {
  it('masks the middle of a phone number', () => {
    expect(maskPhone('09123334567', 'en')).toBe('0912 ••• ••67');
  });
  it('leaves a short value alone', () => {
    expect(maskPhone('123', 'en')).toBe('123');
  });
  it('maps unknown blockers to unavailable', () => {
    expect(checkoutBlockerKey('address_invalid')).toBe('address_invalid');
    expect(checkoutBlockerKey('weird')).toBe('cart_unavailable');
  });
});
