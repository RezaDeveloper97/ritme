import { describe, expect, it } from 'vitest';

import { navFor } from './nav';

// CB-SHOP-05: the shop's order queue and returns hold recipient PII → support only; the catalog stays content.
const keys = (role: 'marketing' | 'support', permissions: Record<string, 'view' | 'manage'>) =>
  navFor(role, { dev: false, permissions }).flatMap((g) => g.items.map((i) => i.key));

describe('shop nav (CB-SHOP-05)', () => {
  it('shows the order queue and returns to support, not to content-only roles', () => {
    const marketing = keys('marketing', { content: 'manage' });
    expect(marketing).toEqual(expect.arrayContaining(['shop', 'shopLayette']));
    expect(marketing).not.toContain('shopOrders');
    expect(marketing).not.toContain('shopReturns');

    const support = keys('support', { support: 'manage' });
    expect(support).toEqual(expect.arrayContaining(['shopOrders', 'shopReturns']));
    expect(support).not.toContain('shop');
  });
});
