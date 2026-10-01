'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import {
  formatTomanThousands,
  usePlusTrial,
  useServerCountdown,
  type PlusTrialOffer,
  type PlusTrialSheet as TrialSheetData,
  type PlusTrialUsage,
} from '@/entities/plus';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  Icon,
  IconCircle,
  PrimaryButton,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  StatusPill,
  type IconName,
  type Tone,
} from '@/shared/ui';

type T = ReturnType<typeof useTranslations<'plus'>>;

const pad = (n: number) => String(n).padStart(2, '0');

/** The usage lines in the artboard's order; keys the design has no row for (voice log) are left out. */
const ROWS: Record<string, { slug: 'deepAnalysis' | 'labAi' | 'assistant' | 'pdfShare' | 'visitDiscount'; icon: IconName; tone: Tone }> = {
  'plus.deep_analysis': { slug: 'deepAnalysis', icon: 'chart', tone: 'brand' },
  'plus.lab_ai': { slug: 'labAi', icon: 'flask', tone: 'data' },
  'plus.assistant_unlimited': { slug: 'assistant', icon: 'sparkle', tone: 'warm' },
  'plus.pdf_share': { slug: 'pdfShare', icon: 'note', tone: 'bloom' },
  'plus.visit_discount': { slug: 'visitDiscount', icon: 'stetho', tone: 'danger' },
};

/** Whole days of the trial (the «۷ روز» of the copy), from its own dates. */
function trialDays(sheet: TrialSheetData, offer: PlusTrialOffer): number {
  const t = sheet.trial;
  const days = t ? Math.round((Date.parse(t.endsAt) - Date.parse(t.startedAt)) / 86_400_000) : NaN;
  return Number.isFinite(days) && days > 0 ? days : offer.daysLeft;
}

function UsageRow({ t, loc, usage }: { t: T; loc: Locale; usage: PlusTrialUsage }) {
  const row = ROWS[usage.key];
  if (!row) return null;
  const base = `trialSheet.features.${row.slug}` as const;
  let sub: string;
  if (row.slug === 'labAi') {
    sub = usage.plusLimit
      ? t('trialSheet.features.labAi.sub', { limit: formatNumber(usage.plusLimit, loc) })
      : t('trialSheet.features.labAi.subUnlimited');
  } else {
    sub = t(`${base}.sub`);
  }
  let value: ReactNode;
  if (row.slug === 'visitDiscount') {
    // Visits arrive with B-N7; until then the line has nothing to count.
    value = (
      <>
        <span aria-hidden>{t('trialSheet.notAvailable')}</span>
        <span className="sr-only">{t('trialSheet.notAvailableLabel')}</span>
      </>
    );
  } else if (usage.used <= 0) {
    value = t('trialSheet.notYet');
  } else {
    value = t(`trialSheet.features.${row.slug}.used`, { count: formatNumber(usage.used, loc) });
  }
  return (
    <li className="plus-tsheet-row">
      <IconCircle icon={row.icon} tone={row.tone} size="sm" />
      <span className="plus-tsheet-row-text">
        <b>{t(`${base}.title`)}</b>
        <span>{sub}</span>
      </span>
      <span className={clsx('plus-tsheet-row-value', `nb-tone-${row.tone}`)}>{value}</span>
    </li>
  );
}

function OfferBody({ sheet, offer, since, t, loc }: { sheet: TrialSheetData; offer: PlusTrialOffer; since: number; t: T; loc: Locale }) {
  const router = useRouter();
  const left = useServerCountdown(offer.secondsLeft, since);
  const days = trialDays(sheet, offer);
  const daysText = formatNumber(days, loc);
  const percent = formatNumber(offer.percent, loc);
  const half = offer.percent === 50;
  const plan = offer.plan;
  const cells = [
    { value: left.days, label: t('trialBanner.days') },
    { value: left.hours, label: t('trialBanner.hours') },
    { value: left.minutes, label: t('trialBanner.minutes') },
  ];

  return (
    <div className="plus-tsheet">
      <Card as="section" className="plus-tsheet-hero" aria-labelledby="plus-tsheet-title">
        <Icon name="crown" size={30} strokeWidth={1.8} className="plus-tsheet-crown" />
        <h2 id="plus-tsheet-title" className="plus-tsheet-title">{t('trialSheet.title')}</h2>
        <p className="plus-tsheet-lead">
          {t.rich('trialSheet.lead', {
            days: daysText,
            discount: half ? t('trialSheet.half') : t('trialSheet.percentCheaper', { percent }),
            hl: (chunks) => <b className="plus-tsheet-hl">{chunks}</b>,
          })}
        </p>
        <div className="plus-tsheet-clock" role="timer" aria-label={t('trialSheet.countdownLabel')}>
          {cells.map((c) => (
            <span key={c.label} className="plus-tsheet-cell">
              <b>{formatNumber(pad(c.value), loc)}</b>
              <span>{c.label}</span>
            </span>
          ))}
        </div>
        <span className="plus-tsheet-caption">{t('trialSheet.countdownCaption')}</span>
      </Card>

      <Card as="section" className="plus-tsheet-usage" aria-labelledby="plus-tsheet-usage">
        <h3 id="plus-tsheet-usage" className="plus-tsheet-h">{t('trialSheet.usageTitle')}</h3>
        <ul className="plus-tsheet-rows">
          {sheet.usage.features.map((u) => (
            <UsageRow key={u.key} t={t} loc={loc} usage={u} />
          ))}
        </ul>
      </Card>

      <Card as="section" className="plus-tsheet-plan" aria-labelledby="plus-tsheet-plan">
        <div className="plus-tsheet-plan-head">
          <h3 id="plus-tsheet-plan" className="plus-tsheet-plan-title">
            {plan.badge ? t('trialSheet.planTitle', { plan: plan.title, badge: plan.badge }) : plan.title}
          </h3>
          <StatusPill tone="warm" className="plus-tsheet-pill">{t('trialSheet.percentOff', { percent })}</StatusPill>
        </div>
        <div className="plus-tsheet-prices">
          {plan.offerPrice !== null && plan.offerPrice < plan.price ? (
            <s className="plus-tsheet-was">{t('trialSheet.was', { amount: formatTomanThousands(plan.price, loc) })}</s>
          ) : null}
          <span className="plus-tsheet-now">
            {t('trialSheet.now', { amount: formatTomanThousands(plan.offerPrice ?? plan.price, loc) })}
          </span>
        </div>
        <p className="plus-tsheet-note">
          {t('trialSheet.perMonth', {
            amount: formatTomanThousands(plan.offerMonthlyPrice ?? plan.monthlyPrice, loc),
            days: daysText,
          })}
        </p>
      </Card>

      {/* Checkout prices the offer server-side (B-N2-06) — the plan id is all it needs. */}
      <PrimaryButton onClick={() => router.push(`/plus/checkout?plan=${plan.id}`)}>
        {half ? t('trialSheet.ctaHalf') : t('trialSheet.ctaPercent', { percent })}
      </PrimaryButton>
      <Link href="/plus" className="plus-tsheet-compare">{t('trialSheet.compare')}</Link>

      <FreeNote t={t} />
      <p className="plus-tsheet-foot">{t('trialSheet.keepData', { days: daysText })}</p>
    </div>
  );
}

function FreeNote({ t }: { t: T }) {
  return (
    <Card as="section" className="plus-tsheet-free">
      <Icon name="heart" size={20} strokeWidth={1.8} className="plus-tsheet-free-icon" />
      <span>
        <b>{t('trialSheet.freeTitle')}</b>
        <span>{t('trialSheet.freeBody')}</span>
      </span>
    </Card>
  );
}

function DaysPill({ t, loc, days }: { t: T; loc: Locale; days: number | null }) {
  return (
    <>
      <span className="sr-only">{t('trialSheet.label')}</span>
      {days !== null ? (
        <span className="plus-tsheet-days">{t('trialSheet.daysLeft', { days: formatNumber(days, loc) })}</span>
      ) : null}
    </>
  );
}

/**
 * «پنجره فرصت پلاس» (B-N2-08, nbl_/nbd_Prem_TrialSheet): what the trial
 * gave so far (`GET /plus/trial` usage — «هنوز نه» for unused, «—» for the
 * visit discount until visits exist), the offer on the featured plan with a
 * live countdown, a checkout CTA at the offer and the free-vs-plus note. A
 * caller-owned inline sheet: fetched only while open.
 */
export function PlusTrialSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useTranslations('plus');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const query = usePlusTrial(open);
  const sheet = query.data;
  const offer = sheet?.offer ?? null;
  const days = offer ? Math.floor(offer.secondsLeft / 86_400) : null;

  let body: ReactNode;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="plus-tsheet">
        <Skeleton shape="card" className="plus-tsheet-skel-hero" />
        <Skeleton shape="card" className="plus-tsheet-skel-list" />
        <Skeleton shape="card" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (query.isError || !sheet) {
    body = (
      <EmptyState
        icon="warning"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <SecondaryButton loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </SecondaryButton>
        }
      />
    );
  } else if (!offer) {
    // The trial (and with it the offer) ended, or she subscribed meanwhile.
    body = (
      <div className="plus-tsheet">
        <EmptyState
          icon="crown"
          title={t('trialSheet.endedTitle')}
          body={t('trialSheet.endedBody')}
          action={<PrimaryButton onClick={() => router.push('/plus/plans')}>{t('trialSheet.endedCta')}</PrimaryButton>}
        />
        <FreeNote t={t} />
      </div>
    );
  } else {
    body = <OfferBody key={query.dataUpdatedAt} sheet={sheet} offer={offer} since={query.dataUpdatedAt} t={t} loc={loc} />;
  }

  return (
    <AppSheet open={open} onClose={onClose} size="full" className="plus-tsheet-panel" title={<DaysPill t={t} loc={loc} days={days} />}>
      {body}
    </AppSheet>
  );
}
