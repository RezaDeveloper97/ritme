import { describe, expect, it } from 'vitest';

import { shopCodeKey } from './order';
import { shopCartEnvelopeSchema, shopOrderEnvelopeSchema, shopReviewEnvelopeSchema } from './order-schema';

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

describe('order item review fields (CB-SHOP-12 F-5)', () => {
  it('reads reviewable / review_status and falls back when absent or unknown', () => {
    const order = shopOrderEnvelopeSchema.parse({
      order: {
        id: 5, code: '520841', status: 'delivered', recipient: {},
        shipments: [{ id: 1, status: 'delivered', items: [
          { id: 7, title: 'Body', reviewable: true, review_status: null },
          { id: 8, title: 'Sleepsuit', reviewable: false, review_status: 'pending' },
          { id: 9, title: 'Pads', review_status: 'weird' },
        ] }],
      },
    }).order;
    const [a, b, c] = order.shipments[0].items;
    expect([a.reviewable, a.review_status]).toEqual([true, null]);
    expect([b.reviewable, b.review_status]).toEqual([false, 'pending']);
    expect([c.reviewable, c.review_status]).toEqual([false, null]);
  });

  it('parses the created review', () => {
    const r = shopReviewEnvelopeSchema.parse({ review: { id: 1, product_id: 2, order_item_id: 7, rating: 4, body: null, status: 'pending', created_at: null } }).review;
    expect(r.status).toBe('pending');
  });
});

describe('shopCodeKey', () => {
  it('maps known refusal codes and everything else to generic', () => {
    expect(shopCodeKey('already_reviewed')).toBe('already_reviewed');
    expect(shopCodeKey('slot_full')).toBe('slot_full');
    expect(shopCodeKey('nope')).toBe('generic');
    expect(shopCodeKey(undefined)).toBe('generic');
  });
});
