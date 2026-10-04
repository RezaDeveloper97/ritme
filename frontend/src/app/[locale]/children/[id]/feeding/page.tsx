import { setRequestLocale } from 'next-intl/server';

import { LogFeedPage } from '@/screens/log-feed';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
  searchParams: Promise<{ section?: string | string[] }>;
}

/** `/children/[id]/feeding[?section=sleep|diapers]` — feed timer, baby sleep and diapers (nbl_Log_Feed, B-N5-07). */
export default async function ChildFeedingRoute({ params, searchParams }: Props) {
  const { locale, id } = await params;
  const { section } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="logFeed">
      <LogFeedPage id={Number(id)} section={typeof section === 'string' ? section : null} />
    </RouteMessages>
  );
}
