import { setRequestLocale } from 'next-intl/server';

import { FertilityLogPage } from '@/screens/fertility-log';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ date?: string; focus?: string }>;
}

export default async function FertilityLogRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { date, focus } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="fertilityLog">
      <FertilityLogPage date={date} focus={focus} />
    </RouteMessages>
  );
}
