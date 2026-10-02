'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import type { AnalysisSection } from '@/entities/analysis';
import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import {
  EmptyState,
  LineChart,
  PrimaryButton,
  ProgressRing,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  StatusPill,
  type Tone,
} from '@/shared/ui';

import { useTtcHub } from '../api/ttc';
import { barShares, lhCells, tryingShare } from '../model/ttc';
import type { TtcBbt, TtcHub, TtcLh, TtcLuteal, TtcMucus, TtcRegularity, TtcTiming } from '../model/types';
import { TtcCard } from './TtcCard';

type T = ReturnType<typeof useTranslations<'analysis.ttc.ui'>>;

function useFmt(): { t: T; num: (n: number) => string; temp: (n: number) => string } {
  const t = useTranslations('analysis.ttc.ui');
  const loc = useLocale() as Locale;
  return { t, num: (n) => formatNumber(n, loc), temp: (n) => formatDecimal(n.toFixed(2), loc) };
}

const FERTILITY = '/analysis/fertility';

function Note({ children }: { children: string }) {
  return <p className="an-card-note">{children}</p>;
}

/** «۴ سیکل اقدام»: summary + referral advice, the trying ring (share of the referral threshold). */
function TryingCard({ trying }: { trying: TtcHub['trying'] }) {
  const { t, num } = useFmt();
  const active = trying.since != null;
  return (
    <section className={clsx('ttc-an-hero', trying.referral.due && 'is-due')} aria-labelledby="ttc-an-trying">
      <div className="ttc-an-hero-text">
        <h2 id="ttc-an-trying" className="ttc-an-hero-title">
          {active ? t('trying.title', { n: trying.cycles }) : t('trying.startTitle')}
        </h2>
        <p className="ttc-an-hero-body">
          {trying.summary.text} {trying.advice.text}
        </p>
      </div>
      {active ? (
        <ProgressRing
          value={tryingShare(trying.months, trying.referral.thresholdMonths)}
          label={t('trying.title', { n: trying.cycles })}
          valueText={t('trying.title', { n: trying.cycles })}
          size={96}
          thickness={9}
          tone="warm"
          className="ttc-an-ring"
        >
          <span className="ttc-an-ring-num">{num(trying.cycles)}</span>
          <span className="ttc-an-ring-unit">{t('trying.unit')}</span>
        </ProgressRing>
      ) : null}
    </section>
  );
}

function BbtCard({ section }: { section: AnalysisSection<TtcBbt> }) {
  const { t, num, temp } = useFmt();
  const b = section.data;
  const values = b?.points.map((p) => p.value) ?? [];
  return (
    <TtcCard title={t('bbt.title')} sub={t('bbt.sub')} href={FERTILITY}>
      {section.ready && b ? (
        <>
          <LineChart
            label={t('bbt.chart', { n: num(values.length), min: temp(Math.min(...values)), max: temp(Math.max(...values)) })}
            series={[{ values, tone: 'brand' }]}
            highlightIndex={values.length - 1}
            height={96}
            className="ttc-an-spark"
          />
          {b.confirmed && b.shiftDay != null ? (
            <StatusPill tone="data" icon="check" className="ttc-an-pill">
              {t('bbt.confirmed', { day: num(b.shiftDay) })}
            </StatusPill>
          ) : (
            <StatusPill tone="neutral" className="ttc-an-pill">
              {t('bbt.waiting')}
            </StatusPill>
          )}
        </>
      ) : (
        <Note>{t('bbt.notReady')}</Note>
      )}
    </TtcCard>
  );
}

function LhCard({ section }: { section: AnalysisSection<TtcLh> }) {
  const { t, num } = useFmt();
  const l = section.data;
  const cells = l ? lhCells(l.tests, l.positiveDay) : [];
  return (
    <TtcCard title={t('lh.title')} href={FERTILITY}>
      {cells.length ? (
        <span
          className="ttc-an-lh"
          role="img"
          aria-label={t('lh.strip', {
            list: (l?.tests ?? []).map((x) => t('lh.item', { day: num(x.day), value: t(`lh.values.${x.value}`) })).join('، '),
          })}
        >
          {cells.map((c) => (
            <span key={c.day} className={clsx('ttc-an-lh-cell', `is-${c.value ?? 'none'}`)} aria-hidden />
          ))}
        </span>
      ) : null}
      <Note>{l?.text.text ?? ''}</Note>
    </TtcCard>
  );
}

const TEASER_DAYS = Array.from({ length: 28 }, (_, i) => i + 1);

function TimingCard({ section }: { section: AnalysisSection<TtcTiming> }) {
  const { t, num } = useFmt();
  const d = section.data;
  return (
    <TtcCard title={t('timing.title')} plus href={FERTILITY} locked={section.locked}>
      {section.locked ? (
        <span className="ttc-an-teaser" aria-hidden>
          <span className="an-num">{t('timing.of', { n: '—', total: '—' })}</span>
          <span className="an-strip-dots ttc-an-dots">
            {TEASER_DAYS.map((n) => (
              <span key={n} className="an-dot" />
            ))}
          </span>
        </span>
      ) : section.ready && d ? (
        <>
          <span className="an-stat">
            <span className="an-num">{t('timing.of', { n: num(d.inWindow), total: num(d.windowDays) })}</span>
            <span className="an-stat-unit">{t('timing.unit')}</span>
          </span>
          <span
            className="an-strip-dots ttc-an-dots"
            role="img"
            aria-label={t('timing.chart', { from: num(d.windowFrom), to: num(d.windowTo), n: num(d.inWindow) })}
            style={{ ['--an-days' as string]: d.days.length }}
          >
            {d.days.map((x) => (
              <span
                key={x.day}
                className={clsx('an-dot', `is-${x.phase}`, x.future && 'is-future', x.intercourse && 'ttc-an-sex')}
              />
            ))}
          </span>
          {d.windowSource ? <span className="an-caption">{t(`timing.source.${d.windowSource}`)}</span> : null}
        </>
      ) : (
        <Note>{t('timing.notReady')}</Note>
      )}
    </TtcCard>
  );
}

function MucusCard({ section }: { section: AnalysisSection<TtcMucus> }) {
  const { t } = useFmt();
  return (
    <TtcCard title={t('mucus.title')} plus href={FERTILITY} locked={section.locked}>
      {section.locked ? (
        <span className="ttc-an-teaser" aria-hidden>
          <Note>{t('empty.body')}</Note>
        </span>
      ) : (
        <Note>{section.data?.text.text ?? ''}</Note>
      )}
    </TtcCard>
  );
}

const LUTEAL_TONE: Record<string, Tone> = { normal: 'data', short: 'warm', long: 'warm' };

function LutealCard({ section }: { section: AnalysisSection<TtcLuteal> }) {
  const { t, num } = useFmt();
  const l = section.data;
  return (
    <TtcCard title={t('luteal.title')} plus href={FERTILITY} locked={section.locked}>
      {section.locked ? (
        <span className="an-stat ttc-an-teaser" aria-hidden>
          <span className="an-num">—</span>
          <span className="an-stat-unit">{t('luteal.unit')}</span>
        </span>
      ) : section.ready && l?.days != null ? (
        <span className="an-stat">
          <span className="an-num">{num(l.days)}</span>
          <span className="an-stat-unit">{t('luteal.unit')}</span>
          {l.status ? (
            <StatusPill tone={LUTEAL_TONE[l.status]} className="an-stat-pill">
              {t(`luteal.status.${l.status}`)}
            </StatusPill>
          ) : null}
        </span>
      ) : (
        <Note>{t('luteal.notReady')}</Note>
      )}
    </TtcCard>
  );
}

function RegularityCard({ section }: { section: AnalysisSection<TtcRegularity> }) {
  const { t, num } = useFmt();
  const r = section.data;
  const bars = r?.cycles ?? [];
  const shares = barShares(bars.map((b) => b.length));
  return (
    <TtcCard title={t('regularity.title')} href="/analysis/cycle">
      {bars.length ? (
        <span
          className="an-bars ttc-an-bars"
          role="img"
          aria-label={t('regularity.chart', { list: bars.map((b) => num(b.length)).join('، ') })}
        >
          <span className="an-bars-row" aria-hidden>
            {bars.map((b, i) => (
              <span
                key={b.start}
                className={clsx('an-bar', i === bars.length - 1 && 'is-last', b.current && 'is-current')}
                style={{ blockSize: `${Math.round(shares[i] * 100)}%` }}
              />
            ))}
          </span>
          <span className="an-bars-labels" aria-hidden>
            {bars.map((b) => (
              <span key={b.start}>{num(b.index)}</span>
            ))}
          </span>
        </span>
      ) : null}
      {r?.status && r.variation != null && r.status !== 'not_enough_data' ? (
        <span className="an-caption">{t(`regularity.status.${r.status}`, { days: r.variation })}</span>
      ) : (
        <Note>{section.ready ? t('regularity.status.not_enough_data') : t('regularity.notReady')}</Note>
      )}
    </TtcCard>
  );
}

function HubSkeleton({ label }: { label: string }) {
  return (
    <SkeletonGroup label={label} className="an-skel">
      <Skeleton shape="block" className="an-skel-finding" />
      <Skeleton shape="card" />
      <Skeleton shape="card" />
      <Skeleton shape="card" />
    </SkeletonGroup>
  );
}

/**
 * The TTC analysis hub (An_Hub_TTC, B-N3-11), slotted into `/analysis` for
 * life-stage `ttc`: cycles trying + referral advice, BBT confirmation, LH,
 * intercourse timing / mucus / luteal phase (Plus) and cycle regularity. The
 * BBT, LH and Plus cards open the fertility detail (`/analysis/fertility`).
 */
export function AnalysisTtcHub() {
  const t = useTranslations('analysis.ttc.ui');
  const tHub = useTranslations('analysis.hub');
  const mounted = useMounted();
  const query = useTtcHub();
  const hub = query.data;

  let body;
  if (!mounted || (query.isPending && query.fetchStatus !== 'idle')) {
    body = <HubSkeleton label={tHub('loading')} />;
  } else if (query.isError || !hub) {
    body = (
      <EmptyState
        icon="warning"
        title={tHub('error.title')}
        body={tHub('error.body')}
        action={
          <SecondaryButton icon="refresh" onClick={() => void query.refetch()} loading={query.isFetching}>
            {tHub('error.retry')}
          </SecondaryButton>
        }
      />
    );
  } else if (!hub.cycles.length) {
    body = (
      <EmptyState
        icon="chart"
        title={t('empty.title')}
        body={t('empty.body')}
        action={
          <PrimaryButton icon="plus" block={false} onClick={() => openSheet('log')}>
            {t('empty.cta')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = (
      <div className="an-content">
        <TryingCard trying={hub.trying} />
        <BbtCard section={hub.bbt} />
        <LhCard section={hub.lh} />
        <TimingCard section={hub.timing} />
        <MucusCard section={hub.mucus} />
        <LutealCard section={hub.luteal} />
        <RegularityCard section={hub.regularity} />
      </div>
    );
  }

  return (
    <div className="an-hub">
      <header className="an-head">
        <h1 className="an-title">{tHub('title')}</h1>
        <StatusPill tone="warm" className="ttc-an-mode">
          {t('modeChip')}
        </StatusPill>
      </header>
      {body}
    </div>
  );
}
