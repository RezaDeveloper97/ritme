import { setRequestLocale } from 'next-intl/server';

import { SplashPage } from '@/screens/auth-splash';
import { INTRO_COOKIE } from '@/shared/session';

interface Props { params: Promise<{ locale: string }> }

/**
 * How long the fallback waits before leaving the splash. Longer than the
 * screen's own 2.2s timer, so a healthy page always routes itself first and
 * this never fires.
 */
const FALLBACK_DELAY_MS = 3500;

/**
 * The splash used to advance only from a `setTimeout` inside the hydrated
 * screen, which made it the one screen that could hang forever: the HTML is
 * prerendered and paints instantly, so a bundle that never finished
 * downloading left the user staring at the spinner with no error and no way
 * out (Cafe Bazaar rejected 1.1.0 for exactly this).
 *
 * The escape hatch is this inline script: it ships inside the HTML, so it runs
 * without a single chunk having loaded, and it picks the same destination the
 * screen would by reading the same cookie. Reading the cookie here rather than
 * on the server keeps the route a static prerender — going dynamic would trade
 * a hung splash for a slower one.
 */
const fallbackScript = (locale: string) => `(function(){` +
  `var seen=document.cookie.split(';').some(function(c){return c.trim().indexOf('${INTRO_COOKIE}=1')===0});` +
  `var t=setTimeout(function(){location.replace('/${locale}/'+(seen?'signup':'welcome'))},${FALLBACK_DELAY_MS});` +
  `window.__ritmeSplashFallback=function(){clearTimeout(t)}` +
  `})()`;

export default async function SplashRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);

  return (
    <>
      {/* Cancelled by the screen the moment it hydrates. */}
      <script dangerouslySetInnerHTML={{ __html: fallbackScript(locale) }} />
      <SplashPage />
    </>
  );
}
