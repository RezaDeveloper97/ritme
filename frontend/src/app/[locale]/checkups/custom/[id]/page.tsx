import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { CustomCheckupFormPage } from '@/screens/checkup-custom-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

export default async function EditCustomCheckupRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const checkupId = Number(id);
  if (!/^\d+$/.test(id) || !Number.isSafeInteger(checkupId) || checkupId <= 0) notFound();
  return (
    <RouteMessages route="checkups">
      <CustomCheckupFormPage id={checkupId} />
    </RouteMessages>
  );
}
