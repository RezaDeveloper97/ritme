import { describe, expect, it } from 'vitest';

import { ALL_MODULES_ON } from '@/entities/app-module';
import { feedLinkTarget, feedState, parseFeed, type FeedItem } from '@/entities/message';

import { bannerSource, feedItemKey, firstVisible, legacyDailyEnabled, usableLink } from './feed-home';

const item = (over: Partial<FeedItem> = {}): FeedItem => ({
  slot: 'status_banner',
  id: 3,
  ruleKey: 'banner.overdue',
  version: 1,
  variant: 'a',
  kind: 'engine',
  level: 'info',
  title: 't',
  body: 'b',
  ctaText: null,
  ctaDeeplink: null,
  dismissible: true,
  evidence: null,
  ...over,
});

describe('banner source precedence', () => {
  it('feed copy wins over the built-in banner', () => {
    expect(bannerSource(item(), true)).toBe('feed');
    expect(bannerSource(item(), false)).toBe('feed');
  });
  it('falls back to the built-in copy, then to nothing', () => {
    expect(bannerSource(null, true)).toBe('built-in');
    expect(bannerSource(null, false)).toBe('none');
  });
});

describe('legacy fallback', () => {
  it('feed 404 (parsed to null and settled) → legacy path, daily asked', () => {
    const state = feedState({ data: null, isPending: false, fetchStatus: 'idle' });
    expect(state).toBe('legacy');
    expect(legacyDailyEnabled(state)).toBe(true);
  });
  it('offline / network error (no data, settled) → legacy path', () => {
    expect(legacyDailyEnabled(feedState({ data: undefined, isPending: false, fetchStatus: 'idle' }))).toBe(true);
  });
  it('while the feed is in flight the legacy daily waits', () => {
    const state = feedState({ data: undefined, isPending: true, fetchStatus: 'fetching' });
    expect(state).toBe('loading');
    expect(legacyDailyEnabled(state)).toBe(false);
  });
  it('an answered feed stops the legacy daily call', () => {
    const feed = parseFeed({ slots: {}, alerts: { open: 0, urgent: 0 }, rules_rev: 1 });
    const state = feedState({ data: feed, isPending: false, fetchStatus: 'idle' });
    expect(state).toBe('ready');
    expect(legacyDailyEnabled(state)).toBe(false);
  });
});

describe('dismissal and links', () => {
  it('a dismissed item is skipped for the session', () => {
    const a = item({ id: 1 });
    const b = item({ id: 2 });
    expect(firstVisible([a, b], new Set([feedItemKey(a)]))).toBe(b);
    expect(firstVisible([a], new Set([feedItemKey(a)]))).toBeNull();
  });
  it('hides a route CTA whose module is off, keeps in-page actions', () => {
    const route = feedLinkTarget('ritme://todo');
    expect(usableLink(route, ALL_MODULES_ON)).toEqual({ kind: 'route', path: '/todo' });
    expect(usableLink(route, { ...ALL_MODULES_ON, todo: false })).toBeNull();
    const act = feedLinkTarget('ritme://period/end');
    expect(usableLink(act, { ...ALL_MODULES_ON, todo: false })).toEqual({ kind: 'action', action: 'edit_end' });
  });
});
