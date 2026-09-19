import { setRequestLocale } from 'next-intl/server';

import { PregnancyLogPage } from '@/screens/pregnancy-log';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ tab?: string }>;
}

export default async function PregnancyLogRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { tab } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyLog">
      <PregnancyLogPage initialTab={tab} />
    </RouteMessages>
  );
}
