import { setRequestLocale } from 'next-intl/server';

import { RemindersPage } from '@/screens/reminders';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ tab?: string }>;
}

export default async function RemindersRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { tab } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminders">
      <RemindersPage initialTab={tab} />
    </RouteMessages>
  );
}
