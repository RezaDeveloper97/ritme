'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { useModuleEnabled } from '@/entities/app-module';
import {
  BodySkeleton,
  EmergencyCard,
  LoadError,
  TRIAGE_LEVELS,
  TriagePill,
  useAssistantError,
  useConversation,
  usePrepareSummary,
  useWhen,
} from '@/entities/assistant';
import { pilot } from '@/shared/pilot';
import { useRouter } from '@/shared/i18n';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer, InfoNote } from '@/shared/ui';
import '@/shared/ui/styles/as.css';

/**
 * `/assistant/chat/[id]/summary` — «خلاصه گفت‌وگو» (nbl_/nbd_v18_Summary): the triage scale, what she said, self-care
 * suggestions, and the way to hand the summary to a doctor. The summary document is (re)written once per visit when
 * the conversation has changed since it was last written.
 */
export function AssistantSummaryPage({ id }: { id: number }) {
  const t = useTranslations('assistant');
  const router = useRouter();
  const when = useWhen();
  const errText = useAssistantError();
  const doctorsOn = useModuleEnabled('doctors');
  const q = useConversation(id);
  const prepare = usePrepareSummary(id);
  const tried = useRef(false);
  const page = q.data;
  const needs = !!page && page.conversation.message_count > 0 && (!page.summary || !page.summary.up_to_date);

  useEffect(() => {
    if (needs && !tried.current) {
      tried.current = true;
      prepare.mutate();
    }
  }, [needs, prepare]);

  let body;
  if (q.isPending) body = <BodySkeleton rows={3} />;
  else if (q.isError || !page) body = <div className="as-body"><LoadError onRetry={() => void q.refetch()} busy={q.isFetching} /></div>;
  else if (page.conversation.message_count === 0)
    body = (
      <div className="as-body">
        <Card><EmptyState icon="note" title={t('summary.emptyTitle')} body={t('summary.emptyBody')} action={<PrimaryButton onClick={() => router.push(`/assistant/chat/${id}`)}>{t('summary.backToChat')}</PrimaryButton>} /></Card>
      </div>
    );
  else if (needs && prepare.isPending) body = <BodySkeleton rows={3} />;
  else if (!page.summary)
    body = (
      <div className="as-body">
        <Card>
          <EmptyState
            icon="warning"
            title={t('summary.failedTitle')}
            body={prepare.isError ? errText(prepare.error) : undefined}
            action={<PrimaryButton icon="refresh" loading={prepare.isPending} onClick={() => prepare.mutate()}>{t('common.retry')}</PrimaryButton>}
          />
        </Card>
      </div>
    );
  else {
    const level = page.summary.triage_level ?? page.conversation.triage_level;
    const photos = page.messages.filter((m) => m.photo).length;
    body = (
      <div className="as-body">
        {level === 'emergency' || page.conversation.emergency ? <EmergencyCard /> : null}
        <Card className="as-triage-card">
          <div className="as-h3-row">
            <h3 className="as-h3">{t('summary.suggestion')}</h3>
            {level ? <TriagePill level={level} /> : null}
          </div>
          <ol className="as-scale" aria-label={t('summary.scale')}>
            {TRIAGE_LEVELS.map((l) => (
              <li key={l} className={clsx('as-scale-step', l === level && 'is-on', `is-${l}`)} aria-current={l === level ? 'step' : undefined}>
                {t(`triage.${l}` as 'triage.soon')}
              </li>
            ))}
          </ol>
        </Card>
        <Card>
          <h3 className="as-h3">{t('summary.said')}</h3>
          <p className="as-summary-text">{page.summary.text}</p>
          {photos > 0 ? (
            <p className="as-fine"><Icon name="image" size={14} /> {t('summary.photos', { count: photos })}</p>
          ) : null}
        </Card>
        {!page.summary.up_to_date ? <InfoNote icon="info">{t('summary.stale')}</InfoNote> : null}
        {doctorsOn ? (
          <Card className="as-handoff-cta">
            <h3 className="as-h3">{t('summary.connect')}</h3>
            <p className="as-text">{t('summary.consentNote')}</p>
            <PrimaryButton block icon="stetho" onClick={() => router.push(`/assistant/chat/${id}/handoff`)} {...pilot('assistant.open:handoff')}>{t('summary.send')}</PrimaryButton>
          </Card>
        ) : null}
        <SecondaryButton block onClick={() => router.push(`/assistant/chat/${id}`)}>{t('summary.backToChat')}</SecondaryButton>
        <InfoNote icon="shield">{t('summary.footnote')}</InfoNote>
      </div>
    );
  }

  return (
    <div className="view as-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('summary.title')}
          subtitle={page ? [page.conversation.department.name, when(page.conversation.last_message_at)].filter(Boolean).join(' · ') : undefined}
          onBack={() => router.push(`/assistant/chat/${id}`)}
          backLabel={t('common.back')}
        />
        {body}
      </div>
    </div>
  );
}
