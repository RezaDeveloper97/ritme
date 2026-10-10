import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { AssistantProfilePage } from '@/screens/assistant-profile';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/assistant/profile` — shared profile (nbl_v18_Profile) (B-N7-07). */
export default async function AssistantProfilePageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="assistantProfile">
      <Suspense>
        <AssistantProfilePage />
      </Suspense>
    </RouteMessages>
  );
}
