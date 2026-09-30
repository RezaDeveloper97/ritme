import type { Metadata, Viewport } from 'next';
import localFont from 'next/font/local';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { ReactNode } from 'react';

import { OnboardingCalendarSync } from '@/entities/user';
import { AppLockGate, appLockInitScript } from '@/features/app-lock';
import { SessionRefresher } from '@/features/auth';
import { BackGuard } from '@/shared/back-guard';
import {
  BUNDLED_LOCALES,
  DirectionProvider,
  getDirection,
  isSupportedLocale,
  pickNamespaces,
} from '@/shared/i18n';
import { NoZoom, ViewportHeight } from '@/shared/lib/viewport';
import { InstallPrompt, UpdateGate } from '@/shared/pwa';
import { SessionGuard } from '@/shared/session';
import { ThemeApplier, themeInitScript } from '@/shared/theme';

import '../globals.css';
import { SHELL_NAMESPACES } from '../message-scopes';
import { AppProviders } from '../providers';
import { SheetHost } from '../sheets/SheetHost';

// One variable woff2 (wght 100–900) instead of six static weights: every page
// preloaded all six (6 × ~21 KB = 128 KB, 31 % of the page weight — perf
// baseline §1.3/§3 #4). This file is the official Vazirmatn v33.003
// `Vazirmatn[wght].ttf` subset to exactly the same 366 code points the static
// files carried (the Arabic-script subset; Latin falls back to the system
// font as before): `python3 -m fontTools.subset 'Vazirmatn[wght].ttf'
// --unicodes-file=<cmap of the old 400> --layout-features='*' --flavor=woff2`.
// 45 KB, one preload, and every weight in use renders exactly, no synthesis.
const vazirmatn = localFont({
  src: '../fonts/Vazirmatn-Variable.woff2',
  weight: '100 900',
  style: 'normal',
  variable: '--font-vazirmatn',
  display: 'swap',
});

// iOS launch screens (`apple-touch-startup-image`), one per iPhone portrait
// size × light/dark. iOS only uses an image whose media query matches the
// device exactly, so every entry names width, height and DPR. The images are
// the --page canvas of each theme with the logo centred (public/splash/).
// [CSS width, CSS height, device pixel ratio]
const IPHONE_SCREENS: ReadonlyArray<readonly [number, number, number]> = [
  [440, 956, 3], // 16 Pro Max
  [430, 932, 3], // 14 Pro Max, 15 Plus / Pro Max, 16 Plus
  [402, 874, 3], // 16 Pro
  [393, 852, 3], // 14 Pro, 15, 15 Pro, 16
  [428, 926, 3], // 12 / 13 Pro Max, 14 Plus
  [390, 844, 3], // 12, 13, 14, 12 / 13 Pro
  [375, 812, 3], // X, XS, 11 Pro, 12 / 13 mini
  [414, 896, 3], // XS Max, 11 Pro Max
  [414, 896, 2], // XR, 11
  [414, 736, 3], // 6+ / 7+ / 8 Plus
  [375, 667, 2], // SE 2nd/3rd gen, 6 / 7 / 8
];

const startupImages = IPHONE_SCREENS.flatMap(([w, h, dpr]) =>
  (['light', 'dark'] as const).map((scheme) => ({
    url: `/splash/${scheme}-${w * dpr}x${h * dpr}.png`,
    media:
      `(device-width: ${w}px) and (device-height: ${h}px) and ` +
      `(-webkit-device-pixel-ratio: ${dpr}) and (orientation: portrait) and ` +
      `(prefers-color-scheme: ${scheme})`,
  })),
);

export const metadata: Metadata = {
  title: 'ریتمی',
  description: 'ریتمی — همراه سلامت زنان',
  applicationName: 'ریتمی',
  manifest: '/manifest.webmanifest',
  appleWebApp: {
    capable: true,
    title: 'ریتمی',
    // Next always emits statusBarStyle ('default' when unset) — that is the
    // light value; chromeInitScript puts the dark one in front of it.
    statusBarStyle: 'default',
    startupImage: startupImages,
  },
  // Next 15.5 renders only `mobile-web-app-capable` for `appleWebApp.capable`;
  // iOS < 16.4 needs the apple- prefixed one to open the home-screen app
  // standalone.
  other: {
    'apple-mobile-web-app-capable': 'yes',
  },
};

// Browser/OS chrome that has to match the resolved theme but cannot read a
// CSS variable: `theme-color` and the iOS status-bar style. These metas are
// created here, after themeInitScript has set data-theme. Never rewrite a meta
// Next rendered: React 19 hydrates head <meta>s by matching their exact
// attributes, so a rewritten one gets a stale twin appended after hydration
// (that was the second `theme-color`, audit I-6).
// - theme-color is not rendered by Next at all; it takes the live --page, so no
//   hex lives here, and the theme store rewrites it on a theme switch.
// - Status bar: light keeps Next's 'default' (dark text on a light bar). Dark
//   gets black-translucent — white text over the app's own dark top strip,
//   which the shell pads with env(safe-area-inset-top) — inserted *first* in
//   <head> so a first-match lookup sees it. Never black-translucent in light:
//   its text is always white and would vanish over the white shell. iOS reads
//   the style at launch (or when the web clip is created), not live.
// Plain string, no imports: it runs inline before first paint.
const chromeInitScript = `(function(){try{
var r=document.documentElement,h=document.head;
var c=getComputedStyle(r).getPropertyValue('--page').trim();
if(c){var m=document.createElement('meta');m.name='theme-color';m.content=c;h.appendChild(m);}
if(r.dataset.theme==='dark'){var s=document.createElement('meta');s.name='apple-mobile-web-app-status-bar-style';s.content='black-translucent';h.insertBefore(s,h.firstChild);}
}catch(e){}})();`;

// theme-color is deliberately absent: chromeInitScript creates it from --page.
export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
  minimumScale: 1,
  userScalable: false,
  viewportFit: 'cover',
};

// Only the compiled-in locales are pre-rendered. A language an admin adds
// later isn't known at build time, so its pages render on demand — which is
// the whole point of resolving the locale list at runtime (CLAUDE.md §6).
export function generateStaticParams() {
  return BUNDLED_LOCALES.map((locale) => ({ locale }));
}

interface LocaleLayoutProps {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}

export default async function LocaleLayout({ children, params }: LocaleLayoutProps) {
  const { locale } = await params;

  if (!(await isSupportedLocale(locale))) {
    notFound();
  }
  setRequestLocale(locale);

  const [messages, direction] = await Promise.all([
    getMessages(),
    getDirection(locale),
  ]);

  return (
    <html lang={locale} dir={direction} suppressHydrationWarning>
      <body className={vazirmatn.className}>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
        <script dangerouslySetInnerHTML={{ __html: chromeInitScript }} />
        {/* B-N1-12: hides the shell before first paint while an app lock is set. */}
        <script dangerouslySetInnerHTML={{ __html: appLockInitScript }} />
        {/* Only what the shell itself renders (PWA prompts, sheets); each
            route adds its own namespaces with <RouteMessages> — see
            app/message-scopes.ts. */}
        <NextIntlClientProvider
          locale={locale}
          messages={pickNamespaces(messages, SHELL_NAMESPACES)}
        >
          <DirectionProvider direction={direction}>
            <AppProviders>
            <ThemeApplier />
            <BackGuard />
            <SessionGuard />
            <SessionRefresher />
            <OnboardingCalendarSync />
            <ViewportHeight />
            <NoZoom />
            <UpdateGate />
            <InstallPrompt />
            <div className="stage">
              <div className="app-shell">
                {/* B-N1-12: while the app lock is closed neither the screen
                    nor any sheet renders — whatever the route or history. */}
                <AppLockGate>
                  {children}
                  {/* Mounted beside the screen, not inside it: every secondary
                      screen is a sheet over whatever is showing, and it has to
                      outlive the screen that opened it (see app/sheets). */}
                  <SheetHost />
                </AppLockGate>
              </div>
            </div>
            </AppProviders>
          </DirectionProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
