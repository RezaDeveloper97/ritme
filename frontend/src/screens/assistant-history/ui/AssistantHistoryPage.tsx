'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  BodySkeleton,
  ConversationRow,
  LoadError,
  scalarOf,
  useAssistantError,
  useAssistantProfile,
  useConversations,
  useDeleteConversations,
  type Conversation,
} from '@/entities/assistant';
import { Link, useRouter } from '@/shared/i18n';
import { Card, EmptyState, HeaderButton, Icon, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';

/**
 * `/assistant/history` — «سابقه سلامت من» (nbl_/nbd_v18_History): every conversation with per-row delete and
 * «delete all» (both confirm first — a hard delete, messages and photos are gone), plus the profile the assistant sees.
 */
export function AssistantHistoryPage() {
  const t = useTranslations('assistant');
  const router = useRouter();
  const errText = useAssistantError();
  const list = useConversations();
  const profile = useAssistantProfile();
  const del = useDeleteConversations();
  const [confirm, setConfirm] = useState<number | 'all' | null>(null);
  const error = del.one.error ?? del.all.error;

  const remove = (target: number | 'all') => {
    const done = { onSuccess: () => setConfirm(null) };
    if (target === 'all') del.all.mutate(undefined, done);
    else del.one.mutate(target, done);
  };

  let body;
  if (list.isPending) body = <BodySkeleton rows={4} />;
  else if (list.isError) body = <div className="as-body"><LoadError onRetry={() => void list.refetch()} busy={list.isFetching} /></div>;
  else {
    const items: Conversation[] = list.data.conversations;
    const shared = profile.data?.consent.granted ? profile.data.items.filter((i) => i.shared) : [];
    body = (
      <div className="as-body">
        {items.length === 0 ? (
          <Card><EmptyState icon="history" title={t('history.emptyTitle')} body={t('history.emptyBody')} action={<PrimaryButton onClick={() => router.push('/assistant')}>{t('hub.title')}</PrimaryButton>} /></Card>
        ) : (
          <div className="as-list">
            {items.map((c) => (
              <div key={c.id} className="as-hrow">
                <ConversationRow c={c} />
                <button type="button" className="as-icon-btn" aria-label={t('history.delete')} onClick={() => setConfirm(c.id)}>
                  <Icon name="trash" size={18} />
                </button>
              </div>
            ))}
          </div>
        )}

        {confirm !== null ? (
          <Card className="as-confirm" role="alertdialog" aria-label={t('history.confirmTitle')}>
            <h3 className="as-h3">{confirm === 'all' ? t('history.confirmAllTitle') : t('history.confirmTitle')}</h3>
            <p className="as-text">{t('history.confirmBody')}</p>
            {error ? <p className="as-error" role="alert">{errText(error)}</p> : null}
            <div className="as-actions">
              <PrimaryButton className="is-danger" loading={del.one.isPending || del.all.isPending} onClick={() => remove(confirm)}>{t('history.confirmYes')}</PrimaryButton>
              <SecondaryButton onClick={() => setConfirm(null)}>{t('common.cancel')}</SecondaryButton>
            </div>
          </Card>
        ) : null}

        {items.length > 0 && confirm === null ? (
          <SecondaryButton block icon="trash" onClick={() => setConfirm('all')}>{t('history.deleteAll')}</SecondaryButton>
        ) : null}

        <Card>
          <div className="as-know-head">
            <strong>{t('history.profile')}</strong>
            <Link href="/assistant/profile" className="as-link">{t('history.editData')}</Link>
          </div>
          {shared.length > 0 ? (
            <div className="as-tags">
              {shared.map((i) => (
                <span key={i.key} className="as-tag">{[t(`profile.keys.${i.key}` as 'profile.keys.age'), scalarOf(i)].filter((v) => v !== null && v !== '').join(' ')}</span>
              ))}
            </div>
          ) : (
            <p className="as-text">{t('hub.knowsNone')}</p>
          )}
        </Card>
      </div>
    );
  }

  return (
    <div className="view as-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('history.title')}
          subtitle={t('history.subtitle')}
          onBack={() => router.push('/assistant')}
          backLabel={t('common.back')}
          action={<HeaderButton label={t('profile.title')} icon="user" onClick={() => router.push('/assistant/profile')} />}
        />
        {body}
      </div>
    </div>
  );
}
