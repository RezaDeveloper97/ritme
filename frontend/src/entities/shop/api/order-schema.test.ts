import { describe, expect, it } from 'vitest';

import { shopCodeKey } from './order';
import { shopCartEnvelopeSchema, shopOrderEnvelopeSchema } from './order-schema';

describe('shopCartEnvelopeSchema', () => {
  it('parses an empty cart and falls back on an unknown line status', () => {
    const empty = shopCartEnvelopeSchema.parse({ cart: {} }).cart;
    expect(empty.groups).toEqual([]);
    expect(empty.discount).toBeNull();
    const cart = shopCartEnvelopeSchema.parse({
      cart: {
        groups: [
          {
            seller: { id: 1, name: 'S' },
            items: [{ variant_id: 3, product_id: 2, title: 'Body', status: 'weird' }],
          },
        ],
      },
    }).cart;
    expect(cart.groups[0].items[0].status).toBe('unavailable');
  });
});

describe('shopOrderEnvelopeSchema', () => {
  it('keeps the replayed flag and tolerates missing shipments', () => {
    const out = shopOrderEnvelopeSchema.parse({ order: { id: 5, code: '520841', status: 'placed', recipient: {} }, replayed: true });
    expect(out.replayed).toBe(true);
    expect(out.order.shipments).toEqual([]);
  });
});

describe('shopCodeKey', () => {
  it('maps known refusal codes and everything else to generic', () => {
    expect(shopCodeKey('slot_full')).toBe('slot_full');
    expect(shopCodeKey('nope')).toBe('generic');
    expect(shopCodeKey(undefined)).toBe('generic');
  });
});
