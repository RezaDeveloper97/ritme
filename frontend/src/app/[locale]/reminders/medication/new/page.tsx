import { setRequestLocale } from 'next-intl/server';

import { MedicationFormPage } from '@/screens/reminder-medication-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  // `for`: the owner a companion records for / whose record this is (B-N4-06).
  searchParams: Promise<{ from?: string; for?: string }>;
}

export default async function NewMedicationRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { from, for: forParam } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminderForm">
      <MedicationFormPage from={from} forParam={forParam} />
    </RouteMessages>
  );
}
