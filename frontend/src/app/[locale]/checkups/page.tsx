import { setRequestLocale } from 'next-intl/server';

import { CheckupsPage } from '@/screens/checkups';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ filter?: string }>;
}

export default async function CheckupsRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { filter } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="checkups">
      <CheckupsPage initialFilter={filter} />
    </RouteMessages>
  );
}
