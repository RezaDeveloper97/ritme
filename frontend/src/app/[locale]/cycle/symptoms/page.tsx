import { setRequestLocale } from 'next-intl/server';

import { CycleSymptomsPage } from '@/screens/cycle';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/cycle/symptoms` — symptom pattern over the typical cycle (B-N1-08). */
export default async function CycleSymptomsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="cycleSymptoms">
      <CycleSymptomsPage />
    </RouteMessages>
  );
}
