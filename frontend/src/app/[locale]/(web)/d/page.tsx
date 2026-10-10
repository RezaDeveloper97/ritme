import type { Metadata } from 'next';
import { setRequestLocale } from 'next-intl/server';

import { DoctorCodePage } from '@/screens/record-export';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** Never indexed or cached; no referrer leaves the page. */
export const metadata: Metadata = {
  robots: { index: false, follow: false, nocache: true, googleBot: { index: false, follow: false } },
  referrer: 'no-referrer',
};

/** Rendered per request (no static HTML with a stale state); `no-store` + `no-referrer` come from next.config headers. */
export const dynamic = 'force-dynamic';

/**
 * `/d` — public: the doctor types the patient's 24h code (CB-REC-05, D-71). `(web)` groups the public pages opened
 * by people who are not Ritme users; it adds no layout — `d` / `emergency` are public segments (no session, no
 * install prompt, no bottom nav).
 */
export default async function DoctorCodeRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="doctorCode">
      <DoctorCodePage />
    </RouteMessages>
  );
}
