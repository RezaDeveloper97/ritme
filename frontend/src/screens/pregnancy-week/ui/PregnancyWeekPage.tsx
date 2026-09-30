'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type TouchEvent, useRef } from 'react';

import { clampV2Week, usePregnancyToday, usePregnancyWeek } from '@/entities/pregnancy';
import { useUpdateWeekState } from '@/features/track-pregnancy';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { swipeTarget } from '../model/swipe';
import { WeekSections } from './WeekSections';
import { WeekTabs } from './WeekTabs';

interface Props {
  /** Week from the URL; `null` (`/pregnancy/weeks`) opens the current week. */
  week: number | null;
}

/** «هفته‌به‌هفته» — `/pregnancy/weeks/[n]` (PregFull_Week). Swipe or tab to a neighbour week. */
export function PregnancyWeekPage({ week: requested }: Props) {
  const t = useTranslations('pregnancyV2');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const router = useRouter();
  const today = usePregnancyToday();
  const currentWeek = today.data ? clampV2Week(today.data.progress.week) : null;
  const week = requested != null ? clampV2Week(requested) : currentWeek;
  const query = usePregnancyWeek(week);
  const bookmark = useUpdateWeekState();
  const touch = useRef<{ x: number; y: number } | null>(null);
  const data = query.data;

  const onTouchStart = (e: TouchEvent) => {
    const p = e.touches[0];
    touch.current = p ? { x: p.clientX, y: p.clientY } : null;
  };
  const onTouchEnd = (e: TouchEvent) => {
    const start = touch.current;
    const p = e.changedTouches[0];
    touch.current = null;
    if (!start || !p || week == null) return;
    const next = swipeTarget(week, p.clientX - start.x, p.clientY - start.y, dir);
    if (next != null) router.replace(`/pregnancy/weeks/${next}`, { scroll: false });
  };

  const trimester = data?.trimester ?? null;
  const rangeText = data
    ? (data.rangeLabel ??
      (data.range
        ? t('common.dateRange', {
            from: formatLongDate(fromApiDate(data.range.from), locale),
            to: formatLongDate(fromApiDate(data.range.to), locale),
          })
        : null))
    : null;
  const subtitle = [trimester ? t(`common.trimester.${trimester as 1 | 2 | 3}`) : null, rangeText]
    .filter(Boolean)
    .join(' · ');

  let body: React.ReactNode;
  if ((query.isPending && week != null) || (week == null && today.isPending)) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="pgn-skel">
        <Skeleton shape="card" className="pgn-skel-hero" />
        <Skeleton shape="block" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || today.isError) {
    body = (
      <Card className="pgn-state" role="alert">
        <span className="pgn-state-disc" aria-hidden>
          <Icon name="warning" size={24} />
        </span>
        <p className="pgn-state-text">{t('common.loadError')}</p>
        <SecondaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
          {t('common.retry')}
        </SecondaryButton>
      </Card>
    );
  } else if (data) {
    body = <WeekSections data={data} />;
  } else {
    body = (
      <Card>
        <EmptyState
          icon="heart"
          title={t('common.notActive')}
          action={
            <Link href="/pregnancy/setup" className="nb-btn is-primary is-block">
              {t('today.setupCta')}
            </Link>
          }
        />
      </Card>
    );
  }

  return (
    <div className="view preg-page pgn-screen">
      <SkyLayer />
      <div className="scroll" onTouchStart={onTouchStart} onTouchEnd={onTouchEnd}>
        <ScreenHeader
          title={week != null ? t('week.headerTitle', { week: formatNumber(week, locale) }) : t('common.title')}
          subtitle={subtitle || undefined}
          onBack={() => router.push('/pregnancy')}
          backLabel={t('common.back')}
          action={
            data ? (
              <HeaderButton
                icon="bookmark"
                className={data.bookmarked ? 'pgn-bookmarked' : undefined}
                label={data.bookmarked ? t('week.unbookmark') : t('week.bookmark')}
                onClick={() => bookmark.mutate({ week: data.week, state: { bookmarked: !data.bookmarked } })}
              />
            ) : undefined
          }
        />
        <div className="pgn-body">
          {week != null && <WeekTabs week={week} />}
          {body}
        </div>
      </div>
      <BottomNav />
    </div>
  );
}
