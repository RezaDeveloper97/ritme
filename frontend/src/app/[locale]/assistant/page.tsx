import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { AssistantHubPage } from '@/screens/assistant';

import { RouteMessages } from '../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/assistant` — the assistant hub (nbl_v18_AssistantHub) (B-N7-07). */
export default async function AssistantHubPageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="assistant">
      <Suspense>
        <AssistantHubPage />
      </Suspense>
    </RouteMessages>
  );
}
