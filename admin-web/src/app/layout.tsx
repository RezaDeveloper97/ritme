import type { Metadata } from 'next';
import localFont from 'next/font/local';
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages, getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { uiDirection } from '@/shared/i18n';
import { themeInitScript } from '@/shared/theme';

import './globals.css';
import { Providers } from './providers';

// Same variable Vazirmatn as the user app (frontend/src/app/fonts). Arabic-script
// subset; Latin falls back to the system font.
const vazirmatn = localFont({
  src: './fonts/Vazirmatn-Variable.woff2',
  weight: '100 900',
  style: 'normal',
  variable: '--font-vazirmatn',
  display: 'swap',
});

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('common');
  return {
    title: { default: `${t('adminPanel')} — ${t('appName')}`, template: `%s — ${t('appName')}` },
    // The admin must never be indexed.
    robots: { index: false, follow: false },
  };
}

export default async function RootLayout({ children }: { children: ReactNode }) {
  const locale = await getLocale();
  const messages = await getMessages();
  return (
    <html lang={locale} dir={uiDirection(locale)} className={vazirmatn.variable} suppressHydrationWarning>
      <head>
        {/* Before first paint: no light flash for a dark-mode admin. */}
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
