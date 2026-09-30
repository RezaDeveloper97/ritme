'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type MouseEvent, useEffect } from 'react';

import {
  checkupIcon,
  pruneCheckupAttachmentsSoon,
  useCheckupHome,
  type CheckupItem,
  type CheckupSummary,
} from '@/entities/checkup';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { withHandoff } from '@/shared/lib/handoff';
import { Icon, Skeleton } from '@/shared/ui';

import {
  BOOK_HREF,
  bookPrefill,
  countsLine,
  guideHref,
  highlightKind,
  highlightMeta,
  ringFraction,
} from '../model/counts';

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
        className="stroke-(--brand)"
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
  const router = useRouter();
  const kind = highlightKind(item);
  const meta = highlightMeta(item, t('separator'));
  // «ثبت نوبت»: the title rides a one-time prefill, not the URL (audit M3-M7 #3).
  const book = (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault();
    router.push(withHandoff(BOOK_HREF, bookPrefill(item)));
  };
  return (
    <div
      className={clsx(
        'ckc-row flex items-center gap-3 bg-(--ck-soft)',
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
        {meta && <span className="block truncate text-[12px] font-semibold text-(--text-3)">{meta}</span>}
      </span>
      <Link
        href={kind === 'book' ? BOOK_HREF : guideHref(item)}
        onClick={kind === 'book' ? book : undefined}
        className="ckc-action flex min-h-11 shrink-0 items-center bg-(--checkup-row-action) text-[12.5px] font-extrabold text-(--ck-ink)"
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

  // Drop report files whose record is gone, e.g. a deleted custom checkup (audit M3-M7 #1).
  useEffect(() => {
    void pruneCheckupAttachmentsSoon();
  }, []);

  if (query.isPending && query.fetchStatus !== 'idle') {
    return (
      <section className="trm-sec" aria-busy="true" aria-label={t('loading')}>
        <div className="trm-card">
          <Header t={t} />
          <Skeleton shape="block" className="trm-skel" />
          <Skeleton shape="block" className="trm-skel is-short" />
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
        {/* v14_Main: ring at the start, title + counts beside it, «همه» at the end. */}
        <div className="ckc-head">
          <Ring summary={data.summary} t={t} locale={locale} />
          <div className="ckc-head-text">
            <h2 className="trm-title">{t('card.title')}</h2>
            <p className="ckc-counts">{counts}</p>
          </div>
          <Link href="/checkups" className="trm-all ckc-all">
            {t('card.all')}
          </Link>
        </div>
        {data.highlights.length > 0 && (
          <div className="mt-2.5 flex flex-col gap-2">
            {data.highlights.map((item) => (
              <HighlightRow key={item.id} item={item} t={t} />
            ))}
          </div>
        )}
        <p className="mt-2.5 mb-0 text-start text-[11.5px] leading-5 text-(--ink-3)">{t('card.disclaimer')}</p>
      </div>
    </section>
  );
}
