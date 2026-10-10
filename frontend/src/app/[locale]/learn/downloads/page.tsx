import { setRequestLocale } from 'next-intl/server';

import { LearnDownloadsPage } from '@/screens/learn-downloads';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/**
 * `/learn/downloads` — «دانلودها» (nbl_Learn_Downloads, B-N8-04). The service
 * worker keeps the last copy of this document so downloaded lessons play offline.
 */
export default async function LearnDownloadsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="learnDownloads">
      <LearnDownloadsPage />
    </RouteMessages>
  );
}
