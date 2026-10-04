import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { IvfMedFormPage } from '@/screens/ivf-meds';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/ivf/meds/{id}` — edit an IVF medicine (CB-IVF-03; form, no nav). */
export default async function EditIvfMedRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  if (!/^\d{1,18}$/.test(id)) notFound();
  return (
    <RouteMessages route="ivfMedForm">
      <IvfMedFormPage medId={Number(id)} />
    </RouteMessages>
  );
}
