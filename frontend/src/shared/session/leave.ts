/**
 * Marker on `<html>` while a signed-out document is on its way out. Hides the
 * page (globals.css), so if Back brings this document back from the bfcache it
 * shows nothing of the previous session until `SessionGuard` has re-checked.
 */
export const SIGNED_OUT_ATTR = 'data-signed-out';

/**
 * Leaves the signed-in app for `href` as a full document replace — no React
 * tree or query cache survives — and hides the page first, so the snapshot a
 * later Back might restore is blank, not the previous user's data.
 */
export function leaveSignedOut(href: string): void {
  document.documentElement.setAttribute(SIGNED_OUT_ATTR, '');
  window.location.replace(href);
}
