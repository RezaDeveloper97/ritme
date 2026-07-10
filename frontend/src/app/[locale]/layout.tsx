import type { Metadata } from 'next';
import localFont from 'next/font/local';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { ReactNode } from 'react';

import { getDirection, isLocale, routing } from '@/shared/i18n';
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
