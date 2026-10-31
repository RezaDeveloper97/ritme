import { describe, expect, it } from 'vitest';

import { feedLinkTarget } from '../lib/feed-link';
import { telHref } from '../lib/feed-view';
import { parseFeed } from './feed-schema';

const banner = {
  id: 3,
  rule_key: 'banner.ask_last_period',
  version: 1,
  variant: 'a',
  kind: 'engine',
  level: 'info',
  title: 'T',
  body: 'B',
  cta_text: 'Go',
  cta_deeplink: 'ritme://period/last',
  dismissible: true,
};

describe('parseFeed', () => {
  it('parses a full feed', () => {
    const feed = parseFeed({
      slots: {
        urgent_modal: [
          {
            ...banner,
            id: 9,
            level: 'urgent',
            dismissible: false,
            evidence: { seen: 's', confidence: 'c', contact: '115' },
          },
        ],
        status_banner: [banner],
        smart_tip: [{ ...banner, id: 4, cta_text: null, cta_deeplink: null }],
        daily_advice: [{ ...banner, id: 5 }, { ...banner, id: 6 }],
        daily_challenge: [],
        modal: [{ ...banner, id: 7 }],
      },
      alerts: { open: 2, urgent: 1 },
      rules_rev: 8,
      daily_challenge: { id: 11, title: 'Walk', description: null, category: 'exercise', duration_days: 7, day: 2, reason: 'ongoing' },
    });
    expect(feed?.slots.urgent_modal[0]?.evidence).toEqual({ seen: 's', confidence: 'c', contact: '115' });
    expect(feed?.slots.status_banner[0]?.ctaDeeplink).toBe('ritme://period/last');
    expect(feed?.slots.smart_tip[0]?.ctaText).toBeNull();
    expect(feed?.slots.daily_advice).toHaveLength(2);
    expect(feed?.alerts).toEqual({ open: 2, urgent: 1 });
    expect(feed?.dailyChallenge).toMatchObject({ id: 11, day: 2, category: 'exercise' });
  });

  it('empty slots parse to empty lists and no challenge', () => {
    const feed = parseFeed({ slots: { status_banner: [] }, alerts: { open: 0, urgent: 0 }, rules_rev: 1 });
    expect(feed?.slots.status_banner).toEqual([]);
    expect(feed?.slots.modal).toEqual([]);
    expect(feed?.dailyChallenge).toBeNull();
  });

  it('ignores unknown slots and fields, and drops malformed or empty items', () => {
    const feed = parseFeed({
      slots: {
        push: [banner],
        status_banner: [{ ...banner, surprise: 1 }, { nope: true }, { ...banner, id: 4, title: '', body: '' }],
      },
      alerts: { open: 0, urgent: 0 },
      rules_rev: 1,
    });
    expect(feed?.slots).not.toHaveProperty('push');
    expect(feed?.slots.status_banner.map((i) => i.id)).toEqual([3]);
  });

  it('returns null for a non-object payload', () => {
    expect(parseFeed(null)).toBeNull();
    expect(parseFeed('x')).toBeNull();
  });
});

describe('deep links and calls', () => {
  it('maps the seeded status banner links to in-page actions', () => {
    expect(feedLinkTarget('ritme://period/end')).toEqual({ kind: 'action', action: 'edit_end' });
    expect(feedLinkTarget('ritme://log/pregnancy-test')).toEqual({ kind: 'action', action: 'log_pregnancy_test' });
    expect(feedLinkTarget('ritme://period/last')).toEqual({ kind: 'action', action: 'edit_last_period' });
  });
  it('maps other app links to routes and refuses external ones', () => {
    expect(feedLinkTarget('ritme://checkups')).toEqual({ kind: 'route', path: '/checkups' });
    expect(feedLinkTarget('/reminders/medication')).toEqual({ kind: 'route', path: '/reminders/medication' });
    expect(feedLinkTarget('https://evil.example')).toBeNull();
    expect(feedLinkTarget('//evil.example')).toBeNull();
    expect(feedLinkTarget('javascript:alert(1)')).toBeNull();
    expect(feedLinkTarget(null)).toBeNull();
  });
  it('a contact becomes a tel link only when it is a number', () => {
    expect(telHref('115')).toBe('tel:115');
    expect(telHref('+98 21 1234 5678')).toBe('tel:+982112345678');
    expect(telHref('Dr. Rad')).toBeNull();
  });
});
