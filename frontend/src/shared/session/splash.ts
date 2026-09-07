/**
 * Where the splash screen sends a visitor: the welcome intro the first time,
 * signup after that.
 *
 * Three places need this answer — the splash screen itself, the middleware
 * (which skips the splash inside the Android shell) and the locale root (which
 * would otherwise redirect into a splash that only redirects again) — so it
 * lives in one directive-free module they can all reach, edge included.
 */
export function postSplashRoute(locale: string, introSeen: boolean): string {
  return `/${locale}/${introSeen ? 'signup' : 'welcome'}`;
}
