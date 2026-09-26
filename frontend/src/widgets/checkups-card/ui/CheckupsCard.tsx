'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { checkupIcon, useCheckupHome, type CheckupItem, type CheckupSummary } from '@/entities/checkup';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { bookHref, countsLine, guideHref, highlightKind, ringFraction } from '../model/counts';

type T = ReturnType<typeof useTranslations<'checkups'>>;

const RING_SIZE = 60;
const RING_STROKE = 6;
const RING_R = (RING_SIZE - RING_STROKE) / 2;
const RING_C = 2 * Math.PI * RING_R;

function Header({ t }: { t: T }) {
  return (
    <div className="trm-head">
      <h2 className="trm-title">{t('card.title')}</h2>
      <Link href="/checkups" className="trm-all">
        {t('card.all')}
      </Link>
    </div>
  );
}

function Ring({ summary, t, locale }: { summary: CheckupSummary; t: T; locale: Locale }) {
  const upToDate = formatNumber(summary.upToDate, locale);
  const total = formatNumber(summary.total, locale);
  const arc = ringFraction(summary) * RING_C;
  return (
    <svg
      role="img"
      aria-label={t('card.ringLabel', { upToDate, total })}
      width={RING_SIZE}
      height={RING_SIZE}
      viewBox={`0 0 ${RING_SIZE} ${RING_SIZE}`}
      className="shrink-0"
    >
      <circle
        cx={RING_SIZE / 2}
        cy={RING_SIZE / 2}
        r={RING_R}
        fill="none"
        strokeWidth={RING_STROKE}
        className="stroke-(--track)"
      />
      <circle
        cx={RING_SIZE / 2}
        cy={RING_SIZE / 2}
        r={RING_R}
        fill="none"
        strokeWidth={RING_STROKE}
        strokeLinecap="round"
        strokeDasharray={`${arc} ${RING_C}`}
        transform={`rotate(-90 ${RING_SIZE / 2} ${RING_SIZE / 2})`}
        className="stroke-(--success)"
      />
      <text
        x="50%"
        y="50%"
        dominantBaseline="central"
        textAnchor="middle"
        direction="ltr"
        className="fill-(--ink) font-['Lalezar'] text-[17px]"
        aria-hidden
      >
        {t('card.ring', { upToDate, total })}
      </text>
    </svg>
  );
}

function HighlightRow({ item, t }: { item: CheckupItem; t: T }) {
  const kind = highlightKind(item);
  const meta = item.timingLabel ?? item.nextDueLabel ?? item.subtitle;
  return (
    <div
      className={clsx(
        'mt-2.5 flex items-center gap-3 rounded-2xl bg-(--ck-soft) px-3 py-2.5',
        kind === 'book' ? 'ck-tone-amber' : 'ck-tone-rose',
      )}
    >
      <span
        className="flex size-9 shrink-0 items-center justify-center rounded-full bg-(--surface) text-(--ck-ink)"
        aria-hidden
      >
        <Icon name={checkupIcon(item.icon, { category: item.category })} size={18} strokeWidth={1.8} />
      </span>
      <span className="min-w-0 flex-1 text-start">
        <span className="block truncate text-[13.5px] font-extrabold text-(--ink)">{item.title}</span>
        {meta && <span className="block truncate text-[12px] font-semibold text-(--ck-ink)">{meta}</span>}
      </span>
      <Link
        href={kind === 'book' ? bookHref(item) : guideHref(item)}
        className="flex min-h-11 shrink-0 items-center rounded-xl bg-(--checkup-row-action) px-3 text-[12.5px] font-extrabold text-(--ck-ink)"
      >
        {t(kind === 'book' ? 'card.book' : 'card.guide')}
      </Link>
    </div>
  );
}

/**
 * «چکاپ‌های دوره‌ای» — the cycle home card (artboard `nbl_v14_Main` /
 * `nbd_v14_Main`). Hidden while there is nothing to show (`total = 0` →
 * the parser returns null) and on error; the pregnancy home never mounts it.
 */
export function CheckupsCard() {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const query = useCheckupHome();

  if (query.isPending && query.fetchStatus !== 'idle') {
    return (
      <section className="trm-sec" aria-busy="true" aria-label={t('loading')}>
        <div className="trm-card">
          <Header t={t} />
          <span className="skeleton-line trm-skel" />
          <span className="skeleton-line trm-skel is-short" />
        </div>
      </section>
    );
  }

  const data = query.data;
  if (query.isError || !data) return null;

  const counts = countsLine(
    data.summary,
    (key, count) => t(`card.${key}`, { count: formatNumber(count, locale) }),
    t('separator'),
  );

  return (
    <section className="trm-sec">
      <div className="trm-card">
        <Header t={t} />
        <div className="mt-1 flex items-center gap-3">
          <Ring summary={data.summary} t={t} locale={locale} />
          <p className="m-0 min-w-0 flex-1 text-start text-[12.5px] leading-6 font-semibold text-(--muted)">
            {counts}
          </p>
        </div>
        {data.highlights.map((item) => (
          <HighlightRow key={item.id} item={item} t={t} />
        ))}
        <p className="mt-2.5 mb-0 text-start text-[11.5px] leading-5 text-(--ink-3)">{t('card.disclaimer')}</p>
      </div>
    </section>
  );
}
