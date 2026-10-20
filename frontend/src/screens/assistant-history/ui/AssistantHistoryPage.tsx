'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

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
import { useModuleEnabled } from '@/entities/app-module';
import { DoctorAvatar, useBookings, useDoctorTitle, useModeLabel, whenLabel } from '@/entities/doctor';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { Card, EmptyState, HeaderButton, Icon, PrimaryButton, ScreenHeader, SecondaryButton, SegmentedTabs, SkyLayer, StatusPill } from '@/shared/ui';

import { type FeedItem, HISTORY_TABS, type HistoryTab, buildFeed } from '../model/feed';
import { useHistoryPdf } from './useHistoryPdf';

const PANEL_ID = 'as-hist-panel';

/**
 * `/assistant/history` — «سابقه سلامت من» (nbl_/nbd_v18_History): every conversation with per-row delete and
 * «delete all» (both confirm first — a hard delete, messages and photos are gone), plus the profile the assistant sees.
 */
export function AssistantHistoryPage() {
  const t = useTranslations('assistant');
  const dt = useTranslations('doctors');
  const router = useRouter();
  const errText = useAssistantError();
  const list = useConversations();
  const profile = useAssistantProfile();
  // «پزشک و ماما» switched off: no visits in the feed and no «پزشک» tab.
  const doctorsOn = useModuleEnabled('doctors');
  const bookings = useBookings('all', doctorsOn);
  const locale = useLocale() as Locale;
  const doctorTitle = useDoctorTitle();
  const modeLabel = useModeLabel();
  const [tab, setTab] = useState<HistoryTab>('all');
  const del = useDeleteConversations();
  const feed = useMemo(
    () => buildFeed(list.data?.conversations ?? [], doctorsOn ? (bookings.data?.items ?? []) : [], tab),
    [list.data, bookings.data, tab, doctorsOn],
  );
  const pdf = useHistoryPdf(feed);
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
    // A failed visits request must not hide the chats: the doctor tab says so, the rest still lists.
    const visitsFailed = bookings.isError && tab === 'doctor';
    const shared = profile.data?.consent.granted ? profile.data.items.filter((i) => i.shared) : [];
    body = (
      <div className="as-body">
        <SegmentedTabs
          tabs={HISTORY_TABS.filter((k) => doctorsOn || k !== 'doctor').map((k) => ({ value: k, label: t(`history.tabs.${k}`) }))}
          value={tab}
          onChange={setTab}
          label={t('history.tabsLabel')}
          panelId={() => PANEL_ID}
        />
        <div id={PANEL_ID} role="tabpanel" aria-label={t(`history.tabs.${tab}`)}>
          {visitsFailed ? (
            <LoadError onRetry={() => void bookings.refetch()} busy={bookings.isFetching} />
          ) : feed.length === 0 ? (
            <Card>
              {items.length === 0 && tab !== 'doctor' ? (
                <EmptyState icon="history" title={t('history.emptyTitle')} body={t('history.emptyBody')} action={<PrimaryButton onClick={() => router.push('/assistant')}>{t('hub.title')}</PrimaryButton>} />
              ) : (
                <EmptyState icon="history" title={t('history.emptyTab')} />
              )}
            </Card>
          ) : (
            <div className="as-list">
              {feed.map((it: FeedItem) =>
                it.kind === 'assistant' ? (
                  <div key={it.key} className="as-hrow">
                    <ConversationRow c={it.conversation} />
                    <button type="button" className="as-icon-btn" aria-label={t('history.delete')} onClick={() => setConfirm(it.conversation.id)}>
                      <Icon name="trash" size={18} />
                    </button>
                  </div>
                ) : (
                  <div key={it.key} className="as-hrow">
                    <Link href={`/services/bookings/${it.booking.id}`} className="as-row">
                      <DoctorAvatar name={it.booking.doctor.name} photoUrl={it.booking.doctor.photo_url} size="sm" />
                      <span className="as-row-main">
                        <strong>{doctorTitle(it.booking.doctor)}</strong>
                        <span>{[t('history.visit'), modeLabel(it.booking.mode), whenLabel(it.booking.date, it.booking.time, locale)].join(' · ')}</span>
                      </span>
                      <StatusPill tone={it.booking.status === 'cancelled' ? 'period' : it.booking.status === 'completed' ? 'brand' : 'data'}>
                        {dt(`status.${it.booking.status}`)}
                      </StatusPill>
                    </Link>
                    <span className="as-icon-gap" aria-hidden="true" />
                  </div>
                ),
              )}
            </div>
          )}
        </div>

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

        {feed.length > 0 ? (
          <div className="as-pdf">
            <SecondaryButton block icon="export" loading={pdf.state === 'working'} onClick={() => void pdf.run()}>
              {pdf.state === 'working' ? t('history.exporting') : t('history.exportPdf')}
            </SecondaryButton>
            {pdf.state === 'error' ? <p className="as-error" role="alert">{t('history.exportError')}</p> : <p className="as-text">{t('history.pdfNote')}</p>}
          </div>
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
          subtitle={t('history.subtitle2')}
          onBack={() => router.push('/assistant')}
          backLabel={t('common.back')}
          action={<HeaderButton label={t('profile.title')} icon="user" onClick={() => router.push('/assistant/profile')} />}
        />
        {body}
      </div>
    </div>
  );
}
