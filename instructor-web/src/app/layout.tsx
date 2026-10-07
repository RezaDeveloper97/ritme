import type { Metadata, Viewport } from 'next';
import localFont from 'next/font/local';
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages, getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { DIRECTION } from '@/shared/i18n';
import { themeInitScript } from '@/shared/theme';

import './globals.css';
import { Providers } from './providers';

// Same fonts as the user app (frontend/src/app/fonts): Vazirmatn for UI, Lalezar
// for display titles and numerals (Night & Bloom tokens.md §6).
const vazirmatn = localFont({
  src: './fonts/Vazirmatn-Variable.woff2',
  weight: '100 900',
  style: 'normal',
  variable: '--font-vazirmatn',
  display: 'swap',
});
const lalezar = localFont({
  src: './fonts/Lalezar-Subset.woff2',
  weight: '400',
  style: 'normal',
  variable: '--font-lalezar',
  display: 'swap',
});

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('common');
  return {
    title: { default: `${t('panelName')} — ${t('appName')}`, template: `%s — ${t('appName')}` },
    // A private panel: never indexed.
    robots: { index: false, follow: false },
  };
}

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  viewportFit: 'cover',
};

export default async function RootLayout({ children }: { children: ReactNode }) {
  const locale = await getLocale();
  const messages = await getMessages();
  return (
    <html
      lang={locale}
      dir={DIRECTION}
      className={`${vazirmatn.variable} ${lalezar.variable}`}
      suppressHydrationWarning
    >
      <head>
        {/* Before first paint: no light flash for a dark-mode instructor. */}
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body>
        <NextIntlClientProvider locale={locale} messages={messages} timeZone="Asia/Tehran">
          <Providers>{children}</Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
