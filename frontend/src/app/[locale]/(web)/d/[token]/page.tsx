import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';

import { SharedReportPage } from '@/screens/record-export';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; token: string }>;
}

/** Never indexed or cached; no referrer leaves the page (the token is in the URL). */
export const metadata: Metadata = {
  robots: { index: false, follow: false, nocache: true, googleBot: { index: false, follow: false } },
  referrer: 'no-referrer',
};

export const dynamic = 'force-dynamic';

/**
 * `/d/[token]` — the short public doctor page (CB-REC-05): the same read-only paper as bloom's
 * `/shared/report/[token]` (which the server's QR path points at), for a 24h summary or a 7-day report token. After the
 * first open the token is replaced by `/d` in the address bar (a refresh lands on the code page).
 */
export default async function DoctorReportRoute({ params }: Props) {
  const { locale, token } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="sharedReport">
      <SharedReportPage token={token} scrubTo={`/${locale}/d`} />
    </RouteMessages>
  );
}
