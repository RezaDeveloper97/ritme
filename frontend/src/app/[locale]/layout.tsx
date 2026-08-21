import type { Metadata, Viewport } from 'next';
import localFont from 'next/font/local';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { ReactNode } from 'react';

import { OnboardingCalendarSync } from '@/entities/user';
import {
  BUNDLED_LOCALES,
  DirectionProvider,
  getDirection,
  isSupportedLocale,
} from '@/shared/i18n';
import { NoZoom, ViewportHeight } from '@/shared/lib/viewport';
import { InstallPrompt, UpdateGate } from '@/shared/pwa';
import { SessionGuard } from '@/shared/session';
import { ThemeApplier, themeInitScript } from '@/shared/theme';

import '../globals.css';
import { AppProviders } from '../providers';
import { SheetHost } from '../sheets/SheetHost';

const vazirmatn = localFont({
  src: [
    { path: '../fonts/Vazirmatn-400.woff2', weight: '400', style: 'normal' },
    { path: '../fonts/Vazirmatn-500.woff2', weight: '500', style: 'normal' },
    { path: '../fonts/Vazirmatn-600.woff2', weight: '600', style: 'normal' },
    { path: '../fonts/Vazirmatn-700.woff2', weight: '700', style: 'normal' },
    { path: '../fonts/Vazirmatn-800.woff2', weight: '800', style: 'normal' },
    { path: '../fonts/Vazirmatn-900.woff2', weight: '900', style: 'normal' },
  ],
  variable: '--font-vazirmatn',
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'ریتمی',
  description: 'ریتمی — همراه سلامت زنان',
  applicationName: 'ریتمی',
  manifest: '/manifest.webmanifest',
  appleWebApp: {
    capable: true,
    title: 'ریتمی',
    statusBarStyle: 'default',
  },
};

// theme-color meta cannot reference CSS variables; these hex values mirror
// --page in globals.css :root / [data-theme="dark"] (baselined exception).
export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
  minimumScale: 1,
  userScalable: false,
  viewportFit: 'cover',
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#F2ECFF' },
    { media: '(prefers-color-scheme: dark)', color: '#131022' },
  ],
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
        <NextIntlClientProvider locale={locale} messages={messages}>
          <DirectionProvider direction={direction}>
            <AppProviders>
            <ThemeApplier />
            <SessionGuard />
            <OnboardingCalendarSync />
            <ViewportHeight />
            <NoZoom />
            <UpdateGate />
            <InstallPrompt />
            <div className="stage">
              <div className="app-shell">
                {children}
                {/* Mounted beside the screen, not inside it: every secondary
                    screen is a sheet over whatever is showing, and it has to
                    outlive the screen that opened it (see app/sheets). */}
                <SheetHost />
              </div>
            </div>
            </AppProviders>
          </DirectionProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
