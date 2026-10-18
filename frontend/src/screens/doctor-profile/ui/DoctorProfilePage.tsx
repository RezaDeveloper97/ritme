'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  BodySkeleton,
  DoctorAvatar,
  LoadError,
  MODE_ICON,
  SlotPicker,
  formatClock,
  formatToman,
  useDoctor,
  useDoctorReviews,
  useDoctorTitle,
  useModeLabel,
  type PickedSlot,
  type VisitMode,
  type VisitType,
} from '@/entities/doctor';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SkyLayer } from '@/shared/ui';

const MODE_TONE: Record<VisitMode, string> = { video: 'nb-tone-data', phone: 'nb-tone-brand', in_person: 'nb-tone-warm' };

/**
 * `/services/doctors/[id]` — profile (nbl_/nbd_v17_DoctorProfile): header card with stats, visit types, the week
 * strip with free times, reviews, and the sticky «رزرو» bar that continues to review & book.
 */
export function DoctorProfilePage({ doctorId }: { doctorId: number }) {
  const t = useTranslations('doctors');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const title = useDoctorTitle();
  const modeLabel = useModeLabel();
  const q = useDoctor(doctorId);
  const [mode, setMode] = useState<VisitMode | null>(null);
  const [slot, setSlot] = useState<PickedSlot | null>(null);
  const [allReviews, setAllReviews] = useState(false);
  const reviews = useDoctorReviews(doctorId, allReviews);
  const back = () => router.push('/services/doctors');

  const shell = (children: React.ReactNode, cta?: React.ReactNode) => (
    <div className="view dr-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={q.data ? title(q.data) : t('profile.title')} onBack={back} backLabel={t('common.back')} />
        {children}
        <div className="page-tail" />
      </div>
      {cta}
    </div>
  );

  if (q.isPending) return shell(<BodySkeleton rows={4} />);
  if (q.isError) {
    const notFound = getApiErrorStatus(q.error) === 404;
    return shell(
      <div className="dr-body">
        {notFound ? (
          <Card>
            <EmptyState icon="stetho" title={t('profile.notFound')} action={<PrimaryButton onClick={back}>{t('profile.backToList')}</PrimaryButton>} />
          </Card>
        ) : (
          <LoadError onRetry={() => void q.refetch()} busy={q.isFetching} />
        )}
      </div>,
    );
  }

  const d = q.data;
  const active: VisitType | undefined = d.visit_types.find((v) => v.mode === mode) ?? d.visit_types[0];
  const activeMode = active?.mode ?? null;
  const chosen = mode ?? activeMode;
  const items = allReviews && reviews.data ? reviews.data.items : d.reviews.items;
  const stat = (value: string | null, label: string, tone: string) =>
    value ? (
      <div className={`dr-stat ${tone}`}>
        <span className="dr-stat-label">{label}</span>
        <span className="dr-stat-value">{value}</span>
      </div>
    ) : null;

  const book = () => {
    if (!chosen || !slot) return;
    router.push(`/services/doctors/${d.id}/book?mode=${chosen}&date=${slot.date}&time=${encodeURIComponent(slot.time)}`);
  };

  return shell(
    <div className="dr-body">
      <Card as="section" className="dr-profile">
        <div className="dr-profile-top">
          <DoctorAvatar name={d.name} photoUrl={d.photo_url} size="lg" />
          <div className="dr-profile-id">
            <h2 className="dr-card-name">{title(d)}</h2>
            <p className="dr-card-sub">{[d.headline ?? d.specialty.title, d.licence_no ? t('profile.licence', { no: formatNumber(d.licence_no, locale) }) : null].filter(Boolean).join(' · ')}</p>
            <p className="dr-card-sub">
              {d.rating !== null ? (
                <span className="dr-rating">
                  <Icon name="star" size={13} fill="currentColor" strokeWidth={0} />
                  {formatNumber(d.rating.toFixed(1), locale)}
                </span>
              ) : null}{' '}
              {t('profile.reviewsCount', { count: formatNumber(d.reviews_count, locale) })}
              {d.experience_years ? ` · ${t('card.years', { count: formatNumber(d.experience_years, locale) })}` : ''}
            </p>
          </div>
        </div>
        <div className="dr-stats">
          {stat(d.stats.visits_count > 0 ? t('profile.visitsValue', { count: formatNumber(d.stats.visits_count, locale) }) : null, t('profile.visits'), 'nb-tone-brand')}
          {stat(d.stats.response_minutes !== null ? t('profile.responseValue', { count: formatNumber(d.stats.response_minutes, locale) }) : null, t('profile.response'), 'nb-tone-data')}
          {stat(d.stats.satisfaction_percent !== null ? t('profile.satisfactionValue', { count: formatNumber(d.stats.satisfaction_percent, locale) }) : null, t('profile.satisfaction'), 'nb-tone-warm')}
        </div>
        {d.bio ? <p className="dr-bio">{d.bio}</p> : null}
      </Card>

      <h2 className="dr-h2">{t('profile.visitTypes')}</h2>
      {d.visit_types.length === 0 ? (
        <Card>
          <EmptyState icon="calendar" title={t('profile.noVisitTypes')} />
        </Card>
      ) : (
        <div role="radiogroup" aria-label={t('profile.visitTypes')} className="dr-types">
          {d.visit_types.map((v) => {
            const on = v.mode === chosen;
            return (
              <button
                key={v.mode}
                type="button"
                role="radio"
                aria-checked={on}
                className={`dr-type ${MODE_TONE[v.mode]}${on ? ' is-on' : ''}`}
                onClick={() => {
                  setMode(v.mode);
                  setSlot(null);
                }}
              >
                <span className="dr-type-icon">
                  <Icon name={MODE_ICON[v.mode]} size={20} />
                </span>
                <span className="dr-type-text">
                  <strong>{v.mode === 'in_person' && v.address ? t('profile.inPerson', { address: v.address }) : modeLabel(v.mode)}</strong>
                  <span>{[t('profile.minutes', { count: formatNumber(v.duration_minutes, locale) }), v.note].filter(Boolean).join(' · ')}</span>
                </span>
                <span className="dr-type-price">{formatToman(v.price_rials, locale)}</span>
                <span className="dr-radio" aria-hidden />
              </button>
            );
          })}
        </div>
      )}

      {chosen ? <SlotPicker key={chosen} doctorId={d.id} mode={chosen} value={slot} onChange={setSlot} /> : null}

      <Card as="section" aria-label={t('profile.reviews')} className="dr-reviews">
        <div className="dr-slot-head">
          <h2 className="dr-h2">{t('profile.reviews')}</h2>
          {!allReviews && d.reviews.count > d.reviews.items.length ? (
            <button type="button" className="dr-link" onClick={() => setAllReviews(true)}>
              {t('profile.all')}
            </button>
          ) : null}
        </div>
        {items.length === 0 ? (
          <p className="dr-slots-sub">{t('profile.noReviews')}</p>
        ) : (
          items.map((r) => (
            <div key={r.id} className="dr-review">
              <span className="dr-review-avatar" aria-hidden>{r.author_initial ?? '?'}</span>
              <div className="dr-review-main">
                <span className="dr-rating">
                  <Icon name="star" size={13} fill="currentColor" strokeWidth={0} />
                  {formatNumber(r.rating, locale)}
                </span>
                {r.body ? <p>{r.body}</p> : null}
              </div>
            </div>
          ))
        )}
        {allReviews && reviews.isPending ? <p className="dr-slots-sub">{t('common.loading')}</p> : null}
      </Card>
    </div>,
    <div className="dr-cta">
      <PrimaryButton icon={chosen ? MODE_ICON[chosen] : undefined} disabled={!slot || !chosen} onClick={book}>
        {slot && chosen ? t('profile.book', { mode: modeLabel(chosen), when: formatClock(slot.time, locale) }) : t('profile.pickSlot')}
      </PrimaryButton>
    </div>,
  );
}
