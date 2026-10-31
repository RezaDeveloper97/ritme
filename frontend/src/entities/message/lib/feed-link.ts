/** In-page actions the home owns (they open its own editors); everything else is a route. */
export type FeedAction = 'edit_end' | 'log_pregnancy_test' | 'edit_last_period';

export type FeedLinkTarget = { kind: 'action'; action: FeedAction } | { kind: 'route'; path: string };

const ACTIONS: Record<string, FeedAction> = {
  'ritme://period/end': 'edit_end',
  'ritme://log/pregnancy-test': 'log_pregnancy_test',
  'ritme://period/last': 'edit_last_period',
};

/**
 * Maps a rule's `cta_deeplink` to the app. The three status-banner links open the home's own editors; any other
 * `ritme://a/b` becomes the route `/a/b`, and a plain in-app path is kept. External URLs and anything odd map to
 * `null` (no button) — a message must never send her out of the app.
 */
export function feedLinkTarget(deeplink: string | null | undefined): FeedLinkTarget | null {
  if (!deeplink) return null;
  const link = deeplink.trim();
  const action = ACTIONS[link.toLowerCase()];
  if (action) return { kind: 'action', action };
  let path: string | null = null;
  if (link.toLowerCase().startsWith('ritme://')) path = `/${link.slice('ritme://'.length)}`;
  else if (link.startsWith('/') && !link.startsWith('//')) path = link;
  if (!path || !/^\/[A-Za-z0-9\-_/]*(\?[A-Za-z0-9\-_=&,.:]*)?$/.test(path)) return null;
  return { kind: 'route', path };
}
