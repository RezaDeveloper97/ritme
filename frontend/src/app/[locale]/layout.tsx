import type { Metadata, Viewport } from 'next';
import localFont from 'next/font/local';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { ReactNode } from 'react';

import { OnboardingCalendarSync } from '@/entities/user';
import { getDirection, isLocale, routing } from '@/shared/i18n';
import { NoZoom, ViewportHeight } from '@/shared/lib/viewport';
import { InstallPrompt, UpdateGate } from '@/shared/pwa';
import { ThemeApplier, themeInitScript } from '@/shared/theme';

import '../globals.css';
import { AppProviders } from '../providers';

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

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

interface LocaleLayoutProps {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}

export default async function LocaleLayout({ children, params }: LocaleLayoutProps) {
  const { locale } = await params;

  if (!isLocale(locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const messages = await getMessages();

  return (
    <html lang={locale} dir={getDirection(locale)} suppressHydrationWarning>
      <body className={vazirmatn.className}>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
        <NextIntlClientProvider locale={locale} messages={messages}>
          <AppProviders>
            <ThemeApplier />
            <OnboardingCalendarSync />
            <ViewportHeight />
            <NoZoom />
            <UpdateGate />
            <InstallPrompt />
            <div className="stage">
              <div className="app-shell">
                {children}
              </div>
            </div>
          </AppProviders>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
