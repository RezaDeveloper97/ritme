import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { AssistantHandoffPage } from '@/screens/assistant-handoff';

import { RouteMessages } from '../../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/assistant/chat/[id]/handoff` (nbl_v18_Handoff) (B-N7-07). */
export default async function AssistantHandoffPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="assistantHandoff">
      <Suspense>
        <AssistantHandoffPage id={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
      </Suspense>
    </RouteMessages>
  );
}
