'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useState } from 'react';

import {
  KIND_LOOK,
  TIMELINE_KINDS,
  type TimelineItem,
  type TimelineKind,
  type TimelineMonth,
  claimBadge,
  stayDays,
  useRecordTimeline,
} from '@/entities/health-record';
import { UploadDocumentSheet } from '@/features/record-documents';
import { type Locale, useDirection, useRouter } from '@/shared/i18n';
import { calendarSystem, formatDayMonth, formatMonthLabel, formatNumber, fromApiDate, toParts } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

const isKind = (v: string | null): v is TimelineKind => (TIMELINE_KINDS as readonly string[]).includes(v ?? '');

/**
 * `/record/timeline` «سوابق و اسناد» (CB-REC-04, nbl_Rec_Timeline): kind chips, the owner's documents and lab sheets
 * grouped by Jalali month (server-side), cursor paging, claim badges from each row's links only. Back header, no nav.
 */
export function RecordTimelinePage() {
  const t = useTranslations('record.timeline');
  const tk = useTranslations('record.kinds');
  const router = useRouter();
  const params = useSearchParams();
  const initial = params.get('kind');
  const [kind, setKind] = useState<TimelineKind>(isKind(initial) ? initial : 'all');
  const [uploadOpen, setUploadOpen] = useState(false);
  const query = useRecordTimeline(kind);
  const months = mergeMonths(query.data?.pages.flatMap((p) => p.months) ?? []);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="rec-tl-skel">
        <Skeleton shape="line" width="short" />
        <Skeleton shape="card" className="rec-tl-skel-card" />
        <Skeleton shape="line" width="short" />
        <Skeleton shape="card" className="rec-tl-skel-card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="warning"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (months.length === 0) {
    body = (
      <EmptyState
        icon="fileDoc"
        title={t('empty')}
        body={t('emptyBody')}
        action={
          <PrimaryButton icon="plus" block={false} onClick={() => setUploadOpen(true)}>
            {t('add')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = (
      <>
        {months.map((m) => (
          <MonthGroup key={m.key} month={m} />
        ))}
        {query.hasNextPage ? (
          <SecondaryButton icon="chevronDown" loading={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>
            {t('more')}
          </SecondaryButton>
        ) : null}
      </>
    );
  }

  return (
    <div className="view rec-tl-page">
      <SkyLayer />
      <div className="scroll rec-tl-scroll">
        <ScreenHeader
          title={t('title')}
          onBack={() => router.push('/record')}
          backLabel={t('back')}
          action={<HeaderButton icon="plus" label={t('add')} onClick={() => setUploadOpen(true)} />}
        />
        <ChipGroup label={t('filters')} className="rec-tl-chips">
          {TIMELINE_KINDS.map((k) => (
            <PillChip key={k} pressed={kind === k} onPressedChange={() => setKind(k)}>
              {tk(k)}
            </PillChip>
          ))}
        </ChipGroup>
        {body}
        {months.length > 0 ? <InfoNote>{t('note')}</InfoNote> : null}
      </div>
      <UploadDocumentSheet
        open={uploadOpen}
        onClose={() => setUploadOpen(false)}
        onCreated={(doc) => {
          setUploadOpen(false);
          router.push(`/record/documents/${doc.id}`);
        }}
      />
    </div>
  );
}

/** Pages never split a day, but a month can span two pages — join them. */
function mergeMonths(months: TimelineMonth[]): TimelineMonth[] {
  const out: TimelineMonth[] = [];
  for (const m of months) {
    const last = out[out.length - 1];
    if (last && last.key === m.key) out[out.length - 1] = { ...last, items: [...last.items, ...m.items] };
    else out.push(m);
  }
  return out;
}

function MonthGroup({ month }: { month: TimelineMonth }) {
  const locale = useLocale() as Locale;
  const label =
    calendarSystem(locale) === 'jalali'
      ? formatMonthLabel(month.jalaliYear, month.jalaliMonth, locale)
      : (() => {
          const p = toParts(fromApiDate(month.start), locale);
          return formatMonthLabel(p.year, p.month, locale);
        })();
  const id = `rec-tl-${month.key}`;
  return (
    <section className="rec-tl-month" aria-labelledby={id}>
      <h2 id={id} className="rec-tl-month-title">
        {label}
      </h2>
      <Card padding="none" className="rec-tl-card">
        <ul className="rec-tl-list">
          {month.items.map((item) => (
            <li key={`${item.type}-${item.id}`}>
              <Row item={item} />
            </li>
          ))}
        </ul>
      </Card>
    </section>
  );
}

function Row({ item }: { item: TimelineItem }) {
  const t = useTranslations('record.timeline');
  const tk = useTranslations('record.kinds');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  const look = KIND_LOOK[item.kind];
  const num = (v: number) => formatNumber(v, locale);

  const sub: string[] = [item.dateKnown ? formatDayMonth(fromApiDate(item.date), locale) : t('noDate')];
  if (item.lab) sub.push(item.lab.allNormal ? t('allNormal') : t('markers', { n: num(item.lab.markerCount) }));
  else if (item.kind === 'hospital') {
    const days = stayDays(item.date, item.endedOn);
    if (days) sub.push(t('stay', { n: num(days) }));
  }
  if (item.centre) sub.push(item.centre);
  else if (item.doctor) sub.push(item.doctor);

  const claim = claimBadge(item.links);
  let badge = null;
  if (claim === 'waiting') badge = <StatusPill tone="period">{t('claimWaiting')}</StatusPill>;
  else if (claim === 'attached') badge = <StatusPill tone="data">{t('claimAttached')}</StatusPill>;
  else if (item.reviewState === 'needs_review') badge = <StatusPill tone="warm">{t('needsReview')}</StatusPill>;
  else if (item.reviewState === 'pending') badge = <StatusPill tone="brand">{t('reading')}</StatusPill>;

  const open = () => router.push(item.type === 'lab' ? `/labs/${item.id}` : `/record/documents/${item.id}`);
  return (
    <button type="button" className="rec-tl-row" onClick={open}>
      <IconCircle icon={look.icon} tone={look.tone} size="md" />
      <span className="rec-tl-text">
        <span className="rec-tl-title">{item.title || tk(item.kind)}</span>
        <span className="rec-tl-sub">{sub.join(' · ')}</span>
      </span>
      {badge ?? <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="rec-tl-chev" />}
    </button>
  );
}
