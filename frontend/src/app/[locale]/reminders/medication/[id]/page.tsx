import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { MedicationFormPage } from '@/screens/reminder-medication-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
  searchParams: Promise<{ from?: string }>;
}

export default async function EditMedicationRoute({ params, searchParams }: Props) {
  const { locale, id } = await params;
  const { from } = await searchParams;
  setRequestLocale(locale);
  const medicationId = Number(id);
  if (!/^\d+$/.test(id) || !Number.isSafeInteger(medicationId) || medicationId <= 0) notFound();
  return (
    <RouteMessages route="reminders">
      <MedicationFormPage id={medicationId} from={from} />
    </RouteMessages>
  );
}
