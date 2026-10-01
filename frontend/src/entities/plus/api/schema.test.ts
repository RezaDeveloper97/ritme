import { describe, expect, it } from 'vitest';

import { catalogSchema, checkoutSchema, quoteEnvelopeSchema, statusSchema } from './schema';

/** Boundary contract for the B-N2-04 `/plus/*` payloads (backend-go/internal/plus/view.go). */
describe('plus schemas', () => {
  it('maps the plan catalogue', () => {
    const catalog = catalogSchema.parse({
      currency: 'IRR',
      vat_rate_bps: 1000,
      trial_days: 7,
      plans: [
        {
          id: 2,
          code: 'plus_3m',
          title: '۳ ماهه',
          badge: 'محبوب‌ترین',
          duration_months: 3,
          price: 2370000,
          monthly_price: 790000,
          savings_percent: 20,
          is_highlighted: true,
        },
      ],
    });
    expect(catalog.trialDays).toBe(7);
    expect(catalog.plans[0]).toMatchObject({ durationMonths: 3, monthlyPrice: 790000, isHighlighted: true, badge: 'محبوب‌ترین' });
  });

  it('maps a free status and a subscribed one', () => {
    const free = statusSchema.parse({
      tier: 'free',
      is_plus: false,
      subscription: null,
      trial: null,
      trial_available: true,
      period_start: '2026-10-01',
      resets_at: '2026-11-01T00:00:00+03:30',
      entitlements: [{ key: 'plus.voice_log', allowed: false, unlimited: false, limit: 0, used: 0, remaining: 0 }],
    });
    expect(free).toMatchObject({ tier: 'free', isPlus: false, subscription: null, trialAvailable: true });

    const plus = statusSchema.parse({
      tier: 'plus',
      is_plus: true,
      subscription: {
        id: 1,
        plan: { id: 2, code: 'plus_3m', title: '3 months', duration_months: 3 },
        status: 'active',
        source: 'purchase',
        starts_at: '2026-10-01T10:00:00+03:30',
        ends_at: '2027-01-01T10:00:00+03:30',
        days_left: 92,
        auto_renew: true,
        canceled_at: null,
      },
      trial: null,
      trial_available: false,
    });
    expect(plus.subscription).toMatchObject({ daysLeft: 92, autoRenew: true, plan: { durationMonths: 3 } });
    expect(plus.entitlements).toEqual([]);
  });

  it('reads a quote with and without the B-N2-06 fields', () => {
    const base = {
      plan: { id: 2, code: 'plus_3m', title: '۳ ماهه', duration_months: 3 },
      currency: 'IRR',
      subtotal: 2370000,
      discount: 237000,
      discount_code: 'RITME10',
      vat_rate_bps: 1000,
      vat: 213300,
      total: 2346300,
    };
    expect(quoteEnvelopeSchema.parse({ quote: base }).discountSource).toBe('code');
    const offer = quoteEnvelopeSchema.parse({
      quote: { ...base, discount_code: null, discount_source: 'trial_offer', trial_offer: { percent: 10 } },
    });
    expect(offer.discountSource).toBe('trial_offer');
    expect(offer.total).toBe(2346300);
  });

  it('maps a checkout with a gateway redirect and a settled one', () => {
    const invoice = {
      reference: 'abc',
      status: 'pending',
      plan: null,
      duration_months: 1,
      currency: 'IRR',
      subtotal: 990000,
      discount: 0,
      discount_code: null,
      vat_rate_bps: 1000,
      vat: 99000,
      total: 1089000,
      gateway: 'fake',
      created_at: '2026-10-01T10:00:00+03:30',
      expires_at: '2026-10-01T10:30:00+03:30',
      paid_at: null,
      receipt: null,
    };
    const pending = checkoutSchema.parse({
      invoice,
      payment: { gateway: 'fake', authority: 'FAKE-abc', redirect_url: 'http://127.0.0.1:8020/api/v1/payments/fake/pay/FAKE-abc' },
    });
    expect(pending.payment?.redirectUrl).toContain('/payments/fake/pay/');
    expect(checkoutSchema.parse({ invoice: { ...invoice, status: 'paid' }, payment: null }).payment).toBeNull();
  });
});
