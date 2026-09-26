import { setRequestLocale } from 'next-intl/server';

import { MedicationFormPage } from '@/screens/reminder-medication-form';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ from?: string }>;
}

export default async function NewMedicationRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { from } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="reminders">
      <MedicationFormPage from={from} />
    </RouteMessages>
  );
}
