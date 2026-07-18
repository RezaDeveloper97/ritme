import { setRequestLocale } from 'next-intl/server';

import { PhaseDetailsPage } from '@/screens/phase-details';

interface Props {
  params: Promise<{ locale: string }>;
}

// Param-less on purpose: the phase is read from the user's live cycle data on
// the client, never from the URL (§11 — cycle phase is health data).
export default async function CyclePhaseRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <PhaseDetailsPage />;
}
