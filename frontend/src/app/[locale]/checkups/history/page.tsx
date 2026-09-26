import { setRequestLocale } from 'next-intl/server';

import { CheckupHistoryPage, parseTypeParam } from '@/screens/checkup-history';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ type?: string }>;
}

export default async function CheckupHistoryRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { type } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="checkups">
      <CheckupHistoryPage type={parseTypeParam(type)} />
    </RouteMessages>
  );
}
