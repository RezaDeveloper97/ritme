import { describe, expect, it } from 'vitest';

import { shopCatalogListSchema, shopCatalogProductSchema } from './schema';

const card = { id: 1, title: 'x', category: 'c', price: 1000 };

describe('catalog list / product parsers (CB-SHOP-07)', () => {
  it('fills defaults for a sparse list payload', () => {
    const l = shopCatalogListSchema.parse({ category: null, products: [card], meta: {} });
    expect(l.products[0]?.in_stock).toBe(true);
    expect(l.filters.sizes).toEqual([]);
    expect(l.meta.last_page).toBe(1);
  });

  it('parses a product with variants, seller terms and reviews', () => {
    const p = shopCatalogProductSchema.parse({
      ...card,
      variants: [{ id: 5, size: 'S', price: 1000, stock: 0, in_stock: false }],
      seller: { name: 'a', ship_min_days: 2, ship_max_days: 4 },
      reviews: { rating_avg: 4.5, rating_count: 2, items: [{ id: 1, rating: 5 }] },
    });
    expect(p.variants[0]?.color).toBeNull();
    expect(p.seller?.return_days).toBe(0);
    expect(p.reviews.items).toHaveLength(1);
    expect(p.bought_together).toEqual([]);
  });
});
