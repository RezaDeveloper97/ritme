'use client';

import { useLocale, useTranslations } from 'next-intl';

import { DEFAULT_RANGE, useAnalysisSummary } from '@/entities/analysis';
import { formatLabValue, MarkerStatePill, stateTone, trimNumber, useLabTrends } from '@/entities/lab';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';
import {
  EmptyState,
  Icon,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';
import { BottomNav } from '@/widgets/bottom-nav';
import { TrendLine } from '@/widgets/charts';

/**
 * `/analysis/labs` (An_Labs, B-N3-10). With verified labs (B-N6-06) it lists
 * every marker with its latest value, state against the sheet's range and its
 * trend line (from 2 labs), each opening the marker detail (B-N6-07); without
 * any, the empty state leads to the lab analysis (`/labs`). The hub's `labs`
 * section (cached from the hub) says whether this is a Plus lock for the reader.
 */
export function AnalysisLabsPage() {
  const t = useTranslations('analysis.reports');
  const tl = useTranslations('analysis.reports.labs');
  const tPlus = useTranslations('plus.gate');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const summary = useAnalysisSummary(DEFAULT_RANGE);
  const trends = useLabTrends();
  const locked = summary.data?.sections.labs.locked === true;
  const markers = trends.data?.markers ?? [];
  const minPoints = trends.data?.minPoints ?? 2;

  const plusNote = locked ? (
    <p id="alb-cta-note" className="alb-plus">
      <PlusBadge label={tPlus('label')} />
      <span>{tl('locked')}</span>
    </p>
  ) : null;

  return (
    <div className="view axc-page">
      <SkyLayer />
      <div className="scroll axc-scroll">
        <ScreenHeader
          title={tl('title')}
          subtitle={
            trends.data && trends.data.labsCount > 0
              ? tl('subLabs', { n: formatNumber(trends.data.labsCount, locale) })
              : tl('sub')
          }
          onBack={() => router.push('/analysis')}
          backLabel={t('back')}
          className="axc-hdr"
        />
        <div className="axc-content">
          {trends.isPending ? (
            <SkeletonGroup label={tl('title')}>
              <Skeleton shape="card" />
              <Skeleton shape="card" />
            </SkeletonGroup>
          ) : trends.isError ? (
            <section className="nb-card alb-card" role="alert">
              <p className="alb-error">{tl('loadError')}</p>
              <SecondaryButton icon="refresh" block={false} loading={trends.isFetching} onClick={() => void trends.refetch()}>
                {tl('retry')}
              </SecondaryButton>
            </section>
          ) : markers.length ? (
            <>
              <section className="nb-card alb-list" aria-labelledby="alb-markers">
                <h2 id="alb-markers" className="alb-list-title">
                  {tl('markersTitle')}
                </h2>
                <ul className="alb-rows">
                  {markers.map((s) => {
                    const values = s.trend.points.map((p) => p.value);
                    return (
                      <li key={s.key}>
                        <button
                          type="button"
                          className="alb-row"
                          onClick={() => router.push(`/labs/${s.latest.labId}/markers/${s.latest.markerId}`)}
                        >
                          <span className="alb-row-top">
                            <span className="alb-row-name">{s.name}</span>
                            <span className={`alb-row-value nb-tone-${stateTone(s.latest.state)}`}>
                              {formatLabValue(s.latest, locale)}
                              {s.unit ? <span className="alb-row-unit">{s.unit}</span> : null}
                            </span>
                            <MarkerStatePill state={s.latest.state} label={s.latest.stateLabel} />
                          </span>
                          {values.length >= minPoints ? (
                            <TrendLine
                              label={tl('trendLabel', {
                                name: s.name,
                                values: values.map((v) => formatDecimal(trimNumber(v), locale)).join('، '),
                              })}
                              values={values}
                              tone={stateTone(s.latest.state)}
                              height={56}
                            />
                          ) : null}
                          {s.trend.sentence ? <span className="alb-row-sub">{s.trend.sentence}</span> : null}
                        </button>
                      </li>
                    );
                  })}
                </ul>
              </section>
              <PrimaryButton icon="flask" onClick={() => router.push('/labs')}>
                {tl('allLabs')}
              </PrimaryButton>
              {plusNote}
            </>
          ) : (
            <section className="nb-card alb-card">
              <EmptyState
                icon="flask"
                title={tl('emptyTitle')}
                body={tl('emptyBody')}
                action={
                  <div className="alb-action">
                    <button type="button" className="alb-cta is-live" onClick={() => router.push('/labs')} aria-describedby={locked ? 'alb-cta-note' : undefined}>
                      <Icon name="flaskLh" size={18} strokeWidth={2} />
                      <span className="alb-cta-label">{tl('cta')}</span>
                    </button>
                    {plusNote}
                  </div>
                }
              />
            </section>
          )}
          <InfoNote>{tl('note')}</InfoNote>
        </div>
      </div>
      <BottomNav />
    </div>
  );
}
