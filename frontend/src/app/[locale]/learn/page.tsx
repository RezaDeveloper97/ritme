import { setRequestLocale } from 'next-intl/server';

import { LearnHubPage } from '@/screens/learn';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/learn` — «دوره‌های من» (nbl_Learn_Hub, B-N8-03). A hub: bottom nav («من»). */
export default async function LearnRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="learn">
      <LearnHubPage />
    </RouteMessages>
  );
}
