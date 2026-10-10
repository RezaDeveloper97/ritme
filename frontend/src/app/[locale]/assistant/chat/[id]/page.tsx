import { setRequestLocale } from 'next-intl/server';
import { Suspense } from 'react';

import { AssistantChatPage } from '@/screens/assistant-chat';

import { RouteMessages } from '../../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/assistant/chat/[id]` — the chat (nbl_v18_AIChat) (B-N7-07). */
export default async function AssistantChatPageRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="assistantChat">
      <Suspense>
        <AssistantChatPage id={/^\d{1,12}$/.test(id) ? Number(id) : 0} />
      </Suspense>
    </RouteMessages>
  );
}
