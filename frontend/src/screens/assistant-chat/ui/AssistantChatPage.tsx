'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import {
  BodySkeleton,
  ConsentCard,
  Disclaimer,

  LoadError,
  takeDraft,
  useAssistantConsent,
  useAssistantError,
  useConversation,
  type ConversationPage,
} from '@/entities/assistant';
import { ChatThread, Composer, itemsOf, useChat, type ChatItem } from '@/features/assistant-chat';
import { getApiErrorStatus } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { Card, EmptyState, HeaderButton, Icon, IconCircle, PrimaryButton, ScreenHeader, SkyLayer } from '@/shared/ui';

function ChatBody({ page }: { page: ConversationPage }) {
  const t = useTranslations('assistant');
  const router = useRouter();
  const errText = useAssistantError();
  const consent = useAssistantConsent('ai_assistant');
  const { conversation: c, department: d } = page;
  const initial: ChatItem[] =
    page.messages.length > 0
      ? itemsOf(page.messages)
      : [{ key: 'intro', role: 'assistant', text: d.intro ?? d.tagline ?? '', card: null, chips: d.quick_replies, triage: null, hasPhoto: false, state: 'complete' }];
  const chat = useChat(c.id, initial, c.emergency);
  const needsConsent = chat.needsConsent || (consent.data ? consent.data.needs_consent && !consent.data.granted : false);

  // A first message stashed by the hub / lab link is sent once, as soon as the consent allows it.
  const sentDraft = useRef(false);
  const canSend = consent.isSuccess && !needsConsent;
  useEffect(() => {
    if (!canSend || sentDraft.current) return;
    sentDraft.current = true;
    const draft = takeDraft(c.id);
    if (draft) void chat.send(draft);
  }, [canSend, c.id, chat]);

  const send = (text: string, photoId: number | null = null) => void chat.send(text, photoId);
  const summaryReady = page.messages.length >= 2 || chat.items.some((i) => i.role === 'assistant' && i.text && i.key !== 'intro');

  return (
    <div className="view as-view as-chat">
      <ScreenHeader
        title={d.name}
        subtitle={t('chat.subtitle')}
        onBack={() => router.push(`/assistant/${d.code}`)}
        backLabel={t('common.back')}
        action={<HeaderButton label={t('chat.summary')} icon="fileDoc" onClick={() => router.push(`/assistant/chat/${c.id}/summary`)} className={summaryReady ? '' : 'is-off'} />}
      />
      <div className="scroll as-chat-scroll">
        <ChatThread items={chat.items} busy={chat.busy} onChip={(x) => send(x)} />
        {chat.denial ? (
          <Card className="as-paywall" role="alert">
            <IconCircle icon="crown" tone="warm" size="md" />
            <p className="as-text">{chat.denial.kind === 'locked' ? t('paywall.locked') : t('paywall.quota', { limit: chat.denial.limit ?? 5 })}</p>
            <PrimaryButton block onClick={() => router.push('/plus')}>{t('paywall.cta')}</PrimaryButton>
          </Card>
        ) : null}
        {chat.errorCode && !chat.denial ? (
          <div className="as-fail" role="alert">
            <Icon name="warning" size={16} />
            <span>{errText(null, chat.errorCode === 'network' ? 'ai_failed' : chat.errorCode)}</span>
            {chat.failedText ? <button type="button" className="as-link" onClick={() => send(chat.failedText ?? '')}>{t('common.retry')}</button> : null}
          </div>
        ) : null}
        {needsConsent ? <ConsentCard code="ai_assistant" /> : null}
        <Disclaimer />
      </div>
      <div className="as-dock">
        <Composer
          busy={chat.busy}
          disabled={needsConsent || !!chat.denial}
          restore={chat.denial || chat.needsConsent ? chat.failedText : null}
          onSend={send}
          onStop={chat.stop}
        />
      </div>
    </div>
  );
}

/** `/assistant/chat/[id]` — the chat (nbl_/nbd_v18_AIChat): SSE streaming answers, chips, photo, triage pill, 115 card. */
export function AssistantChatPage({ id }: { id: number }) {
  const t = useTranslations('assistant');
  const router = useRouter();
  const q = useConversation(id);
  if (q.isPending)
    return (
      <div className="view as-view"><SkyLayer /><div className="scroll"><BodySkeleton rows={4} /></div></div>
    );
  if (q.isError || !q.data)
    return (
      <div className="view as-view">
        <SkyLayer />
        <div className="scroll"><div className="as-body">
          {getApiErrorStatus(q.error) === 404 ? (
            <Card><EmptyState icon="chat" title={t('chat.notFound')} action={<PrimaryButton onClick={() => router.push('/assistant')}>{t('hub.title')}</PrimaryButton>} /></Card>
          ) : (
            <LoadError onRetry={() => void q.refetch()} busy={q.isFetching} />
          )}
        </div></div>
      </div>
    );
  return <ChatBody key={q.data.conversation.id} page={q.data} />;
}

