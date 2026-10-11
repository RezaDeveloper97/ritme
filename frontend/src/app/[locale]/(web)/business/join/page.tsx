import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';

import { WebJoinPage } from '@/screens/web-city-services-business';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** The wizard and the done page hold an applicant's own data: never indexed, no referrer. */
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'webBusiness' });
  return { title: t('meta.joinTitle'), robots: { index: false, follow: false, nocache: true }, referrer: 'no-referrer' };
}

/**
 * `/business/join` — the desktop join wizard, and the done page once the application is sent (CB-DIR-12). `business`
 * is a public segment, so there is no middleware guard: the page itself asks a visitor without a session to log in.
 */
export default async function BusinessJoinRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="webBusiness">
      <WebJoinPage />
    </RouteMessages>
  );
}
