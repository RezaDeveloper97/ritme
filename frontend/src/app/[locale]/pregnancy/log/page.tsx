import { setRequestLocale } from 'next-intl/server';

import { PregnancyLogRoute } from '@/screens/pregnancy-log';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ tab?: string; date?: string }>;
}

export default async function PregnancyLogPageRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { tab, date } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="pregnancyLog">
      <PregnancyLogRoute tab={tab} date={date} />
    </RouteMessages>
  );
}
