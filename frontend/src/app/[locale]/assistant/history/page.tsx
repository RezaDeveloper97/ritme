import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { AssistantHistoryPage } from '@/screens/assistant-history';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/assistant/history` — history (nbl_v18_History) (B-N7-07). */
export default async function AssistantHistoryPageRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="assistantHistory">
      <Suspense>
        <AssistantHistoryPage />
      </Suspense>
    </RouteMessages>
  );
}
