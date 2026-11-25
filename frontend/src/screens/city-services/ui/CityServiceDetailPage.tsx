'use client';

import { useFormatter, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCityServiceListing, useCityServiceSlots } from '@/entities/city-service';
import { Link, useRouter } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { Card, EmptyState, Icon, type IconName, InfoNote, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';

import { minPrice } from '../model/place';
import { CsError, CsSkeleton } from './ListSkeleton';
import { PlaceAbout, PlaceHead, PlaceInfo, PlaceReviews, PlaceServices, PlaceSlots, usePrice } from './PlaceSections';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/csv.css';
import '@/shared/ui/styles/cdp.css';

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
  // CB-DIR-08: the chosen service / slot ride to the request flow as query params (DIR-09 rebuilds that flow).
  const [serviceId, setServiceId] = useState<number | null>(null);
  const [slotId, setSlotId] = useState<number | null>(null);
  const svcSlots = useCityServiceSlots(id, serviceId);
  const price = usePrice();
  const back = () => router.back('/city-services');

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
          <PlaceHead l={l} />
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
        {/* CB-DIR-08: place page sections (board nbl_Dir_Place) */}
        <PlaceAbout l={l} />
        <PlaceServices
          services={l.services}
          serviceId={serviceId}
          onPick={(sid) => {
            setServiceId(sid);
            setSlotId(null);
          }}
        />
        <PlaceSlots
          slots={serviceId !== null ? (svcSlots.data ?? []) : l.next_slots}
          slotId={slotId}
          loading={serviceId !== null && svcSlots.isPending}
          onPick={setSlotId}
        />
        <PlaceReviews l={l} />
        <PlaceInfo l={l} />
        <div className="csv-cta">
          <InfoNote icon={l.booking_mode === 'link' ? 'shield' : 'info'}>
            {t(l.booking_mode === 'link' ? 'detail.linkNote' : l.booking_mode === 'phone' ? 'detail.phoneNote' : 'detail.requestNote')}
          </InfoNote>
          <Link href="/profile/bookings" className="csv-link" {...pilot('city-detail.bookings')}>
            <Icon name="calendar" size={18} />
            {t('list.bookings')}
          </Link>
          <SecondaryButton variant="text" onClick={back}>
            {t('detail.backToList')}
          </SecondaryButton>
        </div>
        <p className="csv-note">{t('note')}</p>
        {/* CB-DIR-08: sticky choose-time bar — the booking action of the venue's mode (link / phone / request+online). */}
        <div className="cdp-bar-sticky">
          <div className="cdp-bar-price">
            {(() => {
              const chosen = l.services.find((s) => s.id === serviceId)?.price_toman ?? minPrice(l.services);
              return chosen !== null && chosen !== undefined ? (
                <>
                  <small>{serviceId !== null ? t('place.price') : t('place.from')}</small>
                  <strong>{price(chosen)}</strong>
                </>
              ) : null;
            })()}
          </div>
          {l.booking_mode === 'link' && href ? (
            <a className="csv-go-btn" href={href} target="_blank" rel="noopener noreferrer">
              {t('detail.linkCta')}
              <Icon name="arrowR" size={16} />
            </a>
          ) : l.booking_mode === 'phone' && l.phone ? (
            <a className="csv-go-btn" href={telHref(l.phone)}>
              <Icon name="phone" size={16} />
              {t('detail.phoneCta')}
            </a>
          ) : l.booking_mode === 'request' || l.booking_mode === 'online' ? (
            <PrimaryButton
              {...pilot('city-detail.continue')}
              onClick={() => {
                const qs = new URLSearchParams();
                if (serviceId !== null) qs.set('service', String(serviceId));
                if (slotId !== null) qs.set('slot', String(slotId));
                const q2 = qs.toString();
                router.push(`/city-services/${l.id}/request${q2 ? `?${q2}` : ''}`);
              }}
            >
              {slotId !== null ? t('place.continue') : l.booking_mode === 'online' ? t('place.chooseTime') : t('detail.requestCta')}
            </PrimaryButton>
          ) : null}
        </div>
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
