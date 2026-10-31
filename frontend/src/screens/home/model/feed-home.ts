import { moduleOfRoute, type AppModules } from '@/entities/app-module';
import type { FeedItem, FeedLinkTarget, FeedState } from '@/entities/message';

/** Where the home banner's copy comes from. The feed wins; the built-in T-M9-42 copy is the fallback (M-14). */
export type BannerSource = 'feed' | 'built-in' | 'none';

export function bannerSource(feedBanner: FeedItem | null, builtInBanner: boolean): BannerSource {
  if (feedBanner) return 'feed';
  return builtInBanner ? 'built-in' : 'none';
}

/** Stable key of one served message, for "already dismissed this session". */
export function feedItemKey(item: Pick<FeedItem, 'slot' | 'id' | 'variant' | 'version'>): string {
  return `${item.slot}:${item.id}:${item.variant}:${item.version}`;
}

/** The first item of a slot that has not been dismissed locally. */
export function firstVisible(items: readonly FeedItem[], dismissed: ReadonlySet<string>): FeedItem | null {
  return items.find((i) => !dismissed.has(feedItemKey(i))) ?? null;
}

/**
 * A CTA to a route is hidden when the admin switched that route's module off (kill-switch). The home's own
 * in-page actions never depend on a module.
 */
export function usableLink(target: FeedLinkTarget | null, modules: AppModules): FeedLinkTarget | null {
  if (!target) return null;
  if (target.kind === 'route') {
    const owner = moduleOfRoute(target.path.split('?')[0] ?? target.path);
    if (owner !== null && modules[owner] === false) return null;
  }
  return target;
}

/** The legacy `/messages/daily` is asked only once the feed has settled without an answer. */
export function legacyDailyEnabled(state: FeedState): boolean {
  return state === 'legacy';
}

/** The urgent modal stays up until the server stops serving it (acknowledged). */
export function urgentToShow(items: readonly FeedItem[]): FeedItem | null {
  return items[0] ?? null;
}
