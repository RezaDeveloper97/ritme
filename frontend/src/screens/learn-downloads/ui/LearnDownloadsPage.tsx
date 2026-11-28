'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { KindThumb, useOfflinePurge } from '@/entities/course';
import { pilot } from '@/shared/pilot';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { deleteOffline, useOfflineBlobUrl, useOfflineList, type OfflineItem } from '@/shared/pwa/offline';
import { AppSheet } from '@/shared/sheet';
import { Card, EmptyState, HeaderButton, Icon, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { meterPercent, sizeParts } from '../model/size';
import '@/shared/ui/styles/lrn.css';

function useSize() {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  return (bytes: number) => {
    const { unit, n } = sizeParts(bytes);
    return t(`downloads.size.${unit}`, { n: formatNumber(n, locale) });
  };
}

/**
 * `/learn/downloads` — «دانلودها» (nbl_/nbd_Learn_Downloads): storage meter,
 * the downloaded lessons (play in-app from the decrypted local copy, delete in
 * edit mode) and the privacy note. Works offline: everything comes from
 * IndexedDB, and the service worker keeps this document. When online, access
 * that ended is purged first (`useOfflinePurge`).
 */
export function LearnDownloadsPage() {
  const t = useTranslations('learn');
  const router = useRouter();
  const size = useSize();
  const { status, items, usage, reload } = useOfflineList();
  const [editing, setEditing] = useState(false);
  const [playing, setPlaying] = useState<OfflineItem | null>(null);
  useOfflinePurge();

  let body;
  if (status === 'loading') {
    body = (
      <SkeletonGroup label={t('common.loading')} className="lrn-dl">
        <Skeleton shape="card" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (status === 'error' || status === 'unsupported') {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t(status === 'error' ? 'downloads.loadError' : 'downloads.unsupported')}
          action={
            status === 'error' ? (
              <PrimaryButton icon="refresh" onClick={() => void reload()}>
                {t('common.retry')}
              </PrimaryButton>
            ) : undefined
          }
        />
      </Card>
    );
  } else {
    const pct = meterPercent(usage.used, usage.quota);
    body = (
      <div className="lrn-dl">
        <Card className="lrn-dl-meter">
          <div className="lrn-dl-meter-row">
            <span>{t('downloads.used')}</span>
            <span className="lrn-dl-meter-val">
              {usage.quota ? t('downloads.usedOf', { used: size(usage.used), quota: size(usage.quota) }) : size(usage.used)}
            </span>
          </div>
          <div
            className="lrn-dl-bar"
            role="progressbar"
            aria-label={t('downloads.used')}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={pct}
          >
            <span className="lrn-dl-bar-fill" style={{ width: `${pct}%` }} />
          </div>
        </Card>

        {items.length === 0 ? (
          <Card>
            <EmptyState icon="download" className="nb-tone-bloom" title={t('downloads.emptyTitle')} body={t('downloads.emptyBody')} />
          </Card>
        ) : (
          <section className="lrn-dl-list" aria-label={t('downloads.listLabel')}>
            {items.map((item) => (
              <div key={item.lessonId} className="lrn-dl-row">
                <button
                  type="button"
                  className="lrn-dl-main"
                  onClick={() => setPlaying(item)}
                  {...pilot(`learn.play:${item.lessonId}`)}
                  aria-label={t('downloads.play', { title: item.title })}
                >
                  <KindThumb kind={item.kind} size="sm" />
                  <span className="lrn-dl-text">
                    <b>{item.title}</b>
                    <span>{t(item.kind === 'pdf' ? 'downloads.metaPdf' : 'downloads.meta', { course: item.courseTitle, size: size(item.size) })}</span>
                  </span>
                </button>
                {editing ? (
                  <button
                    type="button"
                    className="lrn-dl-act is-danger"
                    onClick={() => void deleteOffline(item.lessonId)}
                    aria-label={t('downloads.remove', { title: item.title })}
                  >
                    <Icon name="trash" size={20} />
                  </button>
                ) : (
                  <span className="lrn-dl-act is-ok" aria-hidden>
                    <Icon name="check" size={18} />
                  </span>
                )}
              </div>
            ))}
          </section>
        )}

        <Card className="lrn-dl-note">
          <Icon name="shield" size={20} />
          <p>{t('downloads.note')}</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="view lrn-view">
      <div className="scroll">
        <ScreenHeader
          title={t('downloads.title')}
          subtitle={t('downloads.subtitle')}
          onBack={() => router.back('/learn')}
          backLabel={t('common.back')}
          action={
            items.length ? (
              <HeaderButton
                label={t(editing ? 'downloads.done' : 'downloads.edit')}
                icon={editing ? 'check' : 'pencil'}
                variant="soft"
                onClick={() => setEditing((v) => !v)}
                pilot={pilot('learn.edit-downloads')}
              />
            ) : undefined
          }
        />
        {body}
      </div>
      <BottomNav />
      <AppSheet open={!!playing} onClose={() => setPlaying(null)} title={playing?.title} className="lrn-dl-sheet">
        {playing ? <OfflinePlayer item={playing} /> : null}
      </AppSheet>
    </div>
  );
}

/** Plays the decrypted local copy from a `blob:` URL — never a file URL. */
function OfflinePlayer({ item }: { item: OfflineItem }) {
  const t = useTranslations('learn');
  const { url, failed } = useOfflineBlobUrl(item.lessonId);
  if (failed) return <p className="lrn-media-msg">{t('download.errors.corrupt')}</p>;
  if (!url) return <span className="lrn-media-msg" aria-busy>{t('common.loading')}</span>;
  if (item.kind === 'pdf') return <iframe className="lrn-pdf" src={url} title={item.title} />;
  if (item.kind === 'audio') {
    return <audio className="lrn-dl-audio" src={url} controls autoPlay controlsList="nodownload" onContextMenu={(e) => e.preventDefault()} />;
  }
  return (
    <video
      className="lrn-dl-video"
      src={url}
      controls
      autoPlay
      playsInline
      controlsList="nodownload"
      onContextMenu={(e) => e.preventDefault()}
    />
  );
}
