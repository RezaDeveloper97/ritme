import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { CheckupDetailPage } from '@/screens/checkup-detail';
import { MarkDoneToast } from '@/screens/checkup-mark-done';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

export default async function CheckupDetailRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const checkupId = Number(id);
  if (!/^\d+$/.test(id) || !Number.isSafeInteger(checkupId) || checkupId <= 0) notFound();
  return (
    <RouteMessages route="checkups">
      <CheckupDetailPage id={checkupId} />
      <MarkDoneToast />
    </RouteMessages>
  );
}
