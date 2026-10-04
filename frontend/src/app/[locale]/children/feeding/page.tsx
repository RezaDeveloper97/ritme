import { setRequestLocale } from 'next-intl/server';

import { LogFeedPage } from '@/screens/log-feed';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ section?: string | string[] }>;
}

/** `/children/feeding[?section=]` — the postpartum log sheet's baby tiles: opens the first own child's feeding screen (B-N5-07). */
export default async function FeedingResolveRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { section } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="logFeed">
      <LogFeedPage section={typeof section === 'string' ? section : null} />
    </RouteMessages>
  );
}
