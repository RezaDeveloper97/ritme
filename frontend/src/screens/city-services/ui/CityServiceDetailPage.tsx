'use client';

import { useFormatter, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCityServiceListing } from '@/entities/city-service';
import { Link, useRouter } from '@/shared/i18n';
import { Card, EmptyState, Icon, type IconName, InfoNote, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';

import { CsError, CsSkeleton } from './ListSkeleton';

/** An http(s)-only guard for the admin-entered booking link (the server validates too). */
function safeHref(url: string | null): string | null {
  if (!url) return null;
  return /^https?:\/\//i.test(url) ? url : null;
}

/** A tel: href from a venue phone (digits and a leading +). */
function telHref(phone: string): string {
  return `tel:${phone.replace(/[^\d+]/g, '')}`;
}

/**
 * `/city-services/[id]` — one venue: photos, price, hours, address, phone and the booking action of its mode:
 * `link` opens the venue's own site, `request` opens the request form, `phone` calls the venue. The outgoing link
 * carries no user identifier.
 */
export function CityServiceDetailPage({ id }: { id: number }) {
  const t = useTranslations('cityServices');
  const fmt = useFormatter();
  const router = useRouter();
  const q = useCityServiceListing(id);
  const [photo, setPhoto] = useState(0);
  const back = () => router.push('/city-services');

  let body;
  if (q.isPending) {
    body = <CsSkeleton label={t('loading')} />;
  } else if (q.isError) {
    const missing = (q.error as { response?: { status?: number } } | null)?.response?.status === 404;
    body = missing ? (
      <Card>
        <EmptyState
          icon="home"
          title={t('detail.notFoundTitle')}
          body={t('detail.notFoundBody')}
          action={<PrimaryButton onClick={back}>{t('detail.backToList')}</PrimaryButton>}
        />
      </Card>
    ) : (
      <CsError title={t('loadError')} retry={t('retry')} busy={q.isFetching} onRetry={() => void q.refetch()} />
    );
  } else {
    const l = q.data;
    const href = safeHref(l.booking_url);
    const photos = l.photos.length > 0 ? l.photos : l.photo_url ? [l.photo_url] : [];
    const current = photos[Math.min(photo, photos.length - 1)];
    const facts: [IconName, string, string | null][] = [
      ['mapPin', t('detail.address'), [l.city.name, l.district, l.address].filter(Boolean).join('، ') || null],
      ['clock', t('detail.hours'), l.hours_text],
      ['bookmark', t('detail.price'), l.price_text],
    ];
    body = (
      <div className="csv-stack">
        <div className="csv-hero">
          {current ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={current} alt={t('detail.photo', { n: fmt.number(photo + 1), name: l.name })} className="csv-hero-img" />
          ) : (
            <span className="csv-hero-art" aria-hidden>
              <Icon name={l.category.icon} size={56} strokeWidth={1.3} />
            </span>
          )}
        </div>
        {photos.length > 1 ? (
          <ul className="csv-strip" aria-label={t('detail.title')}>
            {photos.map((p, i) => (
              <li key={p}>
                <button type="button" className={i === photo ? 'csv-strip-btn on' : 'csv-strip-btn'} aria-label={t('detail.photo', { n: fmt.number(i + 1), name: l.name })} aria-pressed={i === photo} onClick={() => setPhoto(i)}>
                  {/* eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image */}
                  <img src={p} alt="" loading="lazy" />
                </button>
              </li>
            ))}
          </ul>
        ) : null}
        <div>
          <h2 className="csv-name">{l.name}</h2>
          <p className="csv-card-meta">
            <span className="csv-badge">{l.category.name}</span>
          </p>
        </div>
        <Card>
          <dl className="csv-facts">
            {facts.map(([icon, label, value]) =>
              value ? (
                <div key={label} className="csv-fact">
                  <Icon name={icon} size={18} />
                  <dt>{label}</dt>
                  <dd>{value}</dd>
                </div>
              ) : null,
            )}
            {l.phone ? (
              <div className="csv-fact">
                <Icon name="phone" size={18} />
                <dt>{t('detail.phone')}</dt>
                <dd>
                  <a className="csv-tel" href={telHref(l.phone)} dir="ltr">
                    {l.phone}
                  </a>
                </dd>
              </div>
            ) : null}
          </dl>
        </Card>
        {l.description ? (
          <Card>
            <h3 className="csv-sub">{t('detail.about')}</h3>
            <p className="csv-text">{l.description}</p>
          </Card>
        ) : null}
        <div className="csv-cta">
          {l.booking_mode === 'link' && href ? (
            <>
              <a className="csv-go-btn" href={href} target="_blank" rel="noopener noreferrer">
                {t('detail.linkCta')}
                <Icon name="arrowR" size={16} />
              </a>
              <InfoNote icon="shield">{t('detail.linkNote')}</InfoNote>
            </>
          ) : null}
          {l.booking_mode === 'request' ? (
            <>
              <PrimaryButton block onClick={() => router.push(`/city-services/${l.id}/request`)}>
                {t('detail.requestCta')}
              </PrimaryButton>
              <InfoNote icon="shield">{t('detail.requestNote')}</InfoNote>
            </>
          ) : null}
          {l.booking_mode === 'phone' && l.phone ? (
            <>
              <a className="csv-go-btn" href={telHref(l.phone)}>
                <Icon name="phone" size={16} />
                {t('detail.phoneCta')}
              </a>
              <InfoNote icon="info">{t('detail.phoneNote')}</InfoNote>
            </>
          ) : null}
          <Link href="/profile/bookings" className="csv-link">
            <Icon name="calendar" size={18} />
            {t('list.bookings')}
          </Link>
          <SecondaryButton variant="text" onClick={back}>
            {t('detail.backToList')}
          </SecondaryButton>
        </div>
        <p className="csv-note">{t('note')}</p>
      </div>
    );
  }

  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={q.data?.name ?? t('detail.title')} onBack={back} backLabel={t('back')} />
        <div className="csv-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
