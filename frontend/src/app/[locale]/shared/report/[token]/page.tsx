import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';

import { SharedReportPage } from '@/screens/record-export';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; token: string }>;
}

/** Never indexed, never cached by search engines, no referrer leaves the page (the token is in the URL). */
export const metadata: Metadata = {
  robots: { index: false, follow: false, nocache: true, googleBot: { index: false, follow: false } },
  referrer: 'no-referrer',
};

/**
 * `/shared/report/[token]` — the public, read-only doctor report behind a 7-day share link (bloom B-N6-04, D-65).
 * Public segment (no session); the report is fetched client-side from GET /shared-reports/{token}.
 */
export default async function SharedReportRoute({ params }: Props) {
  const { locale, token } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="sharedReport">
      <SharedReportPage token={token} />
    </RouteMessages>
  );
}
