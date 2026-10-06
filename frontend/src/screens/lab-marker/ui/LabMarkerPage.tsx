'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { formatLabValue, formatReference, MarkerStatePill, RangeBar, stateTone, trimNumber, useLabMarker } from '@/entities/lab';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDecimal, formatLongDate, formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  UrgentCard,
} from '@/shared/ui';

import { TrendChart } from './TrendChart';

function Shell({ header, children }: { header: ReactNode; children: ReactNode }) {
  return (
    <div className="view lab-screen">
      <SkyLayer />
      <div className="scroll lab-scroll">
        {header}
        <div className="lab-body">{children}</div>
      </div>
    </div>
  );
}

/**
 * `/labs/[id]/markers/[mid]` (nbl_Lab_Marker): the value against its range,
 * the trend across the user's labs, what the marker is, related factors,
 * personal context notes, when to see a doctor and the disclaimer. Catalog
 * and AI texts are plain text.
 */
export function LabMarkerPage({ id, markerId }: { id: number; markerId: number }) {
  const t = useTranslations('labs');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0 && Number.isFinite(markerId) && markerId > 0;
  const q = useLabMarker(valid ? id : 0, valid ? markerId : 0);
  const back = () => router.push(valid ? `/labs/${id}` : '/labs');
  const d = q.data;
  const header = (
    <ScreenHeader
      title={d?.marker.name ?? t('marker.title')}
      subtitle={d?.marker.subtitle ?? undefined}
      onBack={back}
      backLabel={t('common.back')}
    />
  );

  if (!valid || getApiErrorStatus(q.error) === 404) {
    return (
      <Shell header={header}>
        <EmptyState
          icon="flask"
          title={t('common.notFoundTitle')}
          body={t('common.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/labs')}>{t('common.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }
  if (!mounted || q.isPending) {
    return (
      <Shell header={header}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="block" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }
  if (q.isError || !d) {
    return (
      <Shell header={header}>
        <Card className="lab-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="lab-state-text">{t('common.loadError')}</p>
          <SecondaryButton icon="refresh" block={false} loading={q.isFetching} onClick={() => void q.refetch()}>
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const m = d.marker;
  const tone = stateTone(m.state);
  const value = formatLabValue(m, locale);
  const ref = formatReference(m.reference, locale);
  const dateLabel = (iso: string) => {
    const p = toParts(fromApiDate(iso), locale);
    return `${monthName(p.month, locale)} ${formatNumber(String(p.year % 100).padStart(2, '0'), locale)}`;
  };
  // «۱۵ تا ۱۵۰» reads unambiguously in both directions, unlike «۱۵–۱۵۰» inside RTL text (nbl_Lab_Marker)
  const between =
    m.reference.low !== null && m.reference.high !== null
      ? t('marker.between', {
          low: formatDecimal(trimNumber(m.reference.low), locale),
          high: formatDecimal(trimNumber(m.reference.high), locale),
        })
      : (ref ?? '');
  const refLine =
    m.reference.source === 'catalog' ? t('marker.typicalRange', { ref: between }) : t('marker.sheetRange', { ref: between });

  return (
    <Shell header={header}>
      {d.redFlag ? (
        <UrgentCard variant={d.redFlag.severity === 'urgent' ? 'card' : 'note'} icon="warning" title={d.redFlag.name}>
          {d.redFlag.message}
        </UrgentCard>
      ) : null}

      <Card className="lab-mk-hero">
        <div className="lab-mk-top">
          <MarkerStatePill state={m.state} label={m.stateLabel} />
          <span className="lab-mk-date">{formatLongDate(fromApiDate(d.lab.date), locale)}</span>
        </div>
        <p className={`lab-mk-value nb-tone-${tone}`}>
          <span className="lab-mk-num">{value}</span>
          {m.unit ? <span className="lab-mk-unit">{m.unit}</span> : null}
        </p>
        <RangeBar
          size="lg"
          value={m.value}
          low={m.reference.low}
          high={m.reference.high}
          tone={tone}
          label={ref ? t('result.rangeLabel', { value, ref }) : value}
        />
        {ref ? <p className="lab-mk-ref">{refLine}</p> : null}
      </Card>

      <Card as="section" className="lab-res-card" aria-labelledby="lab-mk-trend">
        <h2 id="lab-mk-trend" className="lab-card-title">
          {t('marker.trend')}
        </h2>
        {d.trend.points.length >= 2 ? (
          <TrendChart points={d.trend.points} low={m.reference.low} high={m.reference.high} dateLabel={dateLabel} />
        ) : null}
        {d.trend.sentence ? <p className="lab-mk-sentence">{d.trend.sentence}</p> : null}
      </Card>

      {d.about?.body ? (
        <Card as="section" className="lab-res-card" aria-labelledby="lab-mk-about">
          <h2 id="lab-mk-about" className="lab-card-title">
            {t('marker.what', { name: d.about.name || m.name })}
          </h2>
          <p className="lab-prose">{d.about.body}</p>
        </Card>
      ) : null}

      {d.factors.length || d.contextNotes.length ? (
        <Card as="section" className="lab-res-card" aria-labelledby="lab-mk-factors">
          <h2 id="lab-mk-factors" className="lab-card-title">
            {t('marker.factors')}
          </h2>
          {d.factors.length ? (
            <ul className="lab-checks">
              {d.factors.map((f, i) => (
                <li key={i} className="lab-check is-strong">
                  <Icon name="check" size={14} strokeWidth={2.4} className="lab-check-icon" />
                  <span>{f}</span>
                </li>
              ))}
            </ul>
          ) : null}
          {d.contextNotes.length ? (
            <div className="lab-mk-notes">
              <p className="lab-mk-notes-title">{t('marker.forYou')}</p>
              <ul className="lab-checks">
                {d.contextNotes.map((n, i) => (
                  <li key={i} className="lab-check">
                    <Icon name="user" size={14} strokeWidth={2} className="lab-check-icon" />
                    <span>{n}</span>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </Card>
      ) : null}

      {d.seeDoctor ? (
        <Card as="section" className="lab-res-card" aria-labelledby="lab-mk-doctor">
          <h2 id="lab-mk-doctor" className="lab-card-title">
            {t('marker.doctor')}
          </h2>
          <p className="lab-prose">{d.seeDoctor}</p>
        </Card>
      ) : null}

      <InfoNote>{d.disclaimer || t('common.disclaimer')}</InfoNote>
    </Shell>
  );
}
