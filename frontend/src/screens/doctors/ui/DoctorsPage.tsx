'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  BodySkeleton,
  DoctorAvatar,
  LoadError,
  ModeTag,
  dayOffset,
  formatClock,
  formatToman,
  useDoctorFilters,
  useDoctorTitle,
  useDoctors,
  type DoctorCard,
  type VisitMode,
} from '@/entities/doctor';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber, formatWeekdayDayMonth, fromApiDate } from '@/shared/lib/date';
import { Card, EmptyState, HeaderButton, Icon, PillChip, ScreenHeader, SegmentedTabs, SkyLayer } from '@/shared/ui';

type ModeTab = 'all' | VisitMode;
const MODE_TABS: readonly ModeTab[] = ['all', 'video', 'phone', 'in_person'];

/**
 * `/services/doctors` — «پزشکان و ماماها» (nbl_/nbd_v17_Doctors): mode tabs, filter chips (today, specialty, city,
 * insurer — labels from the catalog filters endpoint), the doctor cards with their first free time, paging.
 */
export function DoctorsPage() {
  const t = useTranslations('doctors');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const title = useDoctorTitle();
  const [tab, setTab] = useState<ModeTab>('all');
  const [today, setToday] = useState(false);
  const [specialty, setSpecialty] = useState<string>();
  const [city, setCity] = useState<string>();
  const [insurance, setInsurance] = useState<string>();
  const [page, setPage] = useState(1);
  const filters = useDoctorFilters();
  const q = useDoctors({ mode: tab === 'all' ? undefined : tab, today, specialty, city, insurance, page });

  const reset = <T,>(set: (v: T) => void) => (v: T) => {
    set(v);
    setPage(1);
  };
  const chips = (list: { code: string; title: string | null }[] | undefined, value: string | undefined, set: (v: string | undefined) => void, label: string) =>
    list && list.length > 0 ? (
      <>
        {list.map((o) => (
          <PillChip key={o.code} pressed={value === o.code} onPressedChange={(on) => reset(set)(on ? o.code : undefined)} aria-label={`${label}: ${o.title ?? o.code}`}>
            {o.title ?? o.code}
          </PillChip>
        ))}
      </>
    ) : null;

  const when = (d: DoctorCard) => {
    if (!d.next_slot) return t('card.noSlot');
    const off = dayOffset(d.next_slot.date, new Date());
    const day = off <= 0 ? t('card.today') : off === 1 ? t('card.tomorrow') : off === 2 ? t('card.dayAfter') : formatWeekdayDayMonth(fromApiDate(d.next_slot.date), locale);
    return t('card.firstSlot', { when: `${day} ${formatClock(d.next_slot.time, locale)}` });
  };

  let body;
  if (q.isPending) body = <BodySkeleton rows={4} />;
  else if (q.isError) body = <div className="dr-body"><LoadError onRetry={() => void q.refetch()} busy={q.isFetching} /></div>;
  else if (q.data.items.length === 0)
    body = (
      <Card className="dr-body">
        <EmptyState icon="stetho" title={t('list.emptyTitle')} body={t('list.emptyBody')} />
      </Card>
    );
  else {
    body = (
      <div className="dr-body">
        {q.data.items.map((d) => (
          <Link key={d.id} href={`/services/doctors/${d.id}`} className="dr-card">
            <DoctorAvatar name={d.name} photoUrl={d.photo_url} />
            <div className="dr-card-main">
              <div className="dr-card-top">
                <h2 className="dr-card-name">{title(d)}</h2>
                {d.rating !== null ? (
                  <span className="dr-rating">
                    <Icon name="star" size={13} fill="currentColor" strokeWidth={0} />
                    {formatNumber(d.rating.toFixed(1), locale)}
                  </span>
                ) : null}
              </div>
              <p className="dr-card-sub">
                {[d.headline ?? d.specialty.title, d.experience_years ? t('card.years', { count: formatNumber(d.experience_years, locale) }) : null].filter(Boolean).join(' · ')}
              </p>
              <div className="dr-tags">
                {d.modes.map((m) => (
                  <ModeTag key={m} mode={m} />
                ))}
              </div>
              <div className="dr-card-foot">
                <span className="dr-when">{when(d)}</span>
                <span className="dr-from">{t('card.from', { price: formatToman(d.price_from_rials, locale) })}</span>
              </div>
            </div>
          </Link>
        ))}
        {q.data.meta.last_page > 1 ? (
          <nav className="dr-pager" aria-label={t('list.pages')}>
            <button type="button" className="dr-link" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              {t('list.prev')}
            </button>
            <span>{t('list.page', { page: formatNumber(page, locale), last: formatNumber(q.data.meta.last_page, locale) })}</span>
            <button type="button" className="dr-link" disabled={page >= q.data.meta.last_page} onClick={() => setPage((p) => p + 1)}>
              {t('list.next')}
            </button>
          </nav>
        ) : null}
        <Link href="/assistant" className="dr-ask">
          <span className="dr-ask-icon"><Icon name="sparkle" size={20} /></span>
          <span className="dr-ask-text">
            <strong>{t('list.askTitle')}</strong>
            <span>{t('list.askBody')}</span>
          </span>
          <span className="dr-link">{t('list.ask')}</span>
        </Link>
      </div>
    );
  }

  return (
    <div className="view dr-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('list.title')}
          subtitle={q.data ? t('list.count', { count: formatNumber(q.data.meta.total, locale) }) : undefined}
          onBack={() => router.push('/services')}
          backLabel={t('common.back')}
          action={<HeaderButton label={t('bookings.title')} icon="calendar" onClick={() => router.push('/services/bookings')} />}
        />
        <div className="dr-filters">
          <SegmentedTabs
            tabs={MODE_TABS.map((m) => ({ value: m, label: t(m === 'all' ? 'list.all' : `mode.${m}`) }))}
            value={tab}
            onChange={reset(setTab)}
            label={t('list.modeLabel')}
          />
          <div className="dr-chiprow">
            <PillChip pressed={today} onPressedChange={reset(setToday)}>
              {t('list.today')}
            </PillChip>
            {chips(filters.data?.specialties, specialty, setSpecialty, t('list.specialty'))}
            {chips(filters.data?.cities, city, setCity, t('list.city'))}
            {chips(filters.data?.insurers, insurance, setInsurance, t('list.insurer'))}
          </div>
        </div>
        {body}
        <div className="page-tail" />
      </div>
    </div>
  );
}
