import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { AssistantSummaryPage } from '@/screens/assistant-summary';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/assistant/chat/[id]/summary` (nbl_v18_Summary) (B-N7-07). */
export default async function AssistantSummaryPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="assistantSummary">
      <Suspense>
        <AssistantSummaryPage id={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
      </Suspense>
    </RouteMessages>
  );
}
