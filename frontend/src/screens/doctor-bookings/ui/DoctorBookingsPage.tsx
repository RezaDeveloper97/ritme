'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { BodySkeleton, DoctorAvatar, LoadError, ModeTag, useBookings, useDoctorTitle, whenLabel, type Booking } from '@/entities/doctor';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { Card, EmptyState, PrimaryButton, ScreenHeader, SegmentedTabs, SkyLayer, StatusPill } from '@/shared/ui';

type Scope = 'upcoming' | 'past';
const SCOPES: readonly Scope[] = ['upcoming', 'past'];
const TONE: Record<Booking['status'], 'data' | 'brand' | 'warm' | 'period'> = {
  confirmed: 'data',
  held: 'warm',
  completed: 'brand',
  cancelled: 'period',
  expired: 'period',
  failed: 'period',
};

/** `/services/bookings` — my bookings (upcoming | past); a row opens the booking where it is cancelled / moved. */
export function DoctorBookingsPage() {
  const t = useTranslations('doctors');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const title = useDoctorTitle();
  const [scope, setScope] = useState<Scope>('upcoming');
  const q = useBookings(scope);

  let body;
  if (q.isPending) body = <BodySkeleton rows={3} />;
  else if (q.isError) body = <div className="dr-body"><LoadError onRetry={() => void q.refetch()} busy={q.isFetching} /></div>;
  else if (q.data.items.length === 0)
    body = (
      <Card className="dr-body">
        <EmptyState
          icon="calendar"
          title={t(scope === 'upcoming' ? 'bookings.emptyUpcoming' : 'bookings.emptyPast')}
          action={scope === 'upcoming' ? <PrimaryButton onClick={() => router.push('/services/doctors')}>{t('bookings.find')}</PrimaryButton> : undefined}
        />
      </Card>
    );
  else
    body = (
      <div className="dr-body">
        {q.data.items.map((b) => (
          <Link key={b.id} href={`/services/bookings/${b.id}`} className="dr-card">
            <DoctorAvatar name={b.doctor.name} photoUrl={b.doctor.photo_url} size="md" />
            <div className="dr-card-main">
              <div className="dr-card-top">
                <h2 className="dr-card-name">{title(b.doctor)}</h2>
                <StatusPill tone={TONE[b.status]}>{t(`status.${b.status}`)}</StatusPill>
              </div>
              <p className="dr-card-sub">{whenLabel(b.date, b.time, locale)}</p>
              <div className="dr-tags">
                <ModeTag mode={b.mode} />
              </div>
            </div>
          </Link>
        ))}
      </div>
    );

  return (
    <div className="view dr-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('bookings.title')} onBack={() => router.push('/services/doctors')} backLabel={t('common.back')} />
        <div className="dr-filters">
          <SegmentedTabs tabs={SCOPES.map((s) => ({ value: s, label: t(`bookings.${s}`) }))} value={scope} onChange={setScope} label={t('bookings.title')} />
        </div>
        {body}
        <div className="page-tail" />
      </div>
    </div>
  );
}
