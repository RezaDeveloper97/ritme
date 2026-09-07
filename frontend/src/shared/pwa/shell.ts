/**
 * Recognises the Android shell (`android-shell/`), which hosts the site in its
 * own WebView and stamps `RitmeApp/<version>` onto the UA.
 *
 * Deliberately a plain, directive-free module: the middleware matches on the
 * request header at the edge and `InstallPrompt` on `navigator.userAgent` in
 * the browser, and both must use the same pattern. (A `'use client'` module
 * would hand the edge a client-reference proxy instead of the function — see
 * the note in `shared/session/cookie.ts`.)
 */
const SHELL_UA = / RitmeApp\//;

export function isShellUserAgent(userAgent: string | null | undefined): boolean {
  return !!userAgent && SHELL_UA.test(userAgent);
}
