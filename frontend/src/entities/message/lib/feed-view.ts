import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

import type { FeedItem, FeedSlot, MessageFeed } from '../model/feed';

/** Items of a slot (empty when the feed is absent). */
export function slotItems(feed: MessageFeed | null | undefined, slot: FeedSlot): FeedItem[] {
  return feed?.slots[slot] ?? [];
}

/** The winner of a single-capacity slot. */
export function slotWinner(feed: MessageFeed | null | undefined, slot: FeedSlot): FeedItem | null {
  return slotItems(feed, slot)[0] ?? null;
}

/** `daily_advice` shows up to three. */
export function dailyAdvice(feed: MessageFeed | null | undefined): FeedItem[] {
  return slotItems(feed, 'daily_advice').slice(0, 3);
}

/**
 * The server renders numbers (`{days_late}`) with Latin digits; show them in the app's digits (Persian in fa).
 * Only used on feed text, so a phone number in a call button goes through {@link telHref} instead.
 */
export function localizeFeedText(text: string, locale: Locale): string {
  return formatNumber(text, locale);
}

/** The feed state the home branches on. */
export type FeedState = 'loading' | 'ready' | 'legacy';

/**
 * `legacy` = the feed did not answer (Laravel 404, network error, offline): built-in copy and the legacy
 * endpoints, exactly as before the feed. The legacy `/messages/daily` is asked only once the feed has settled
 * without an answer.
 */
export function feedState(q: { data: MessageFeed | null | undefined; isPending: boolean; fetchStatus: string }): FeedState {
  if (q.data) return 'ready';
  if (q.isPending && q.fetchStatus === 'fetching') return 'loading';
  return 'legacy';
}

/** `tel:` target of an urgent contact when it is a phone number (115, 1480, +98…); otherwise null. */
export function telHref(contact: string): string | null {
  const digits = contact.replace(/[\s\-()]/g, '');
  return /^\+?\d{3,15}$/.test(digits) ? `tel:${digits}` : null;
}
