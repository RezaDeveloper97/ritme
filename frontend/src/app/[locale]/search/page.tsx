import { setRequestLocale } from 'next-intl/server';

import { SearchPage } from '@/screens/search';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/search` — global search from the Today header (CB-NAV-02, nbd_Nav_Search). A flow: no bottom nav. */
export default async function SearchRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="search">
      <SearchPage />
    </RouteMessages>
  );
}
