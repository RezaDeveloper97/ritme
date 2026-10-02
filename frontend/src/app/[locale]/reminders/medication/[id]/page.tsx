import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';

import { MedicationFormPage } from '@/screens/reminder-medication-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
  // `for`: the owner a companion records for / whose record this is (B-N4-06).
  searchParams: Promise<{ from?: string; for?: string }>;
}

export default async function EditMedicationRoute({ params, searchParams }: Props) {
  const { locale, id } = await params;
  const { from, for: forParam } = await searchParams;
  setRequestLocale(locale);
  const medicationId = Number(id);
  if (!/^\d+$/.test(id) || !Number.isSafeInteger(medicationId) || medicationId <= 0) notFound();
  return (
    <RouteMessages route="reminderForm">
      <MedicationFormPage id={medicationId} from={from} forParam={forParam} />
    </RouteMessages>
  );
}
