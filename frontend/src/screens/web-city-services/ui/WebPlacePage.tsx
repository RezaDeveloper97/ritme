'use client';

import clsx from 'clsx';
import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type CityServiceDetail, useCityServiceListing, useCityServiceSlots } from '@/entities/city-service';
import { Link, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { isAuthenticated } from '@/shared/session';
import { Card, EmptyState, Icon, Skeleton, SkeletonGroup } from '@/shared/ui';

import { APP_DIRECTORY, bookable, minPrice, neshanHref, PLACES_HOME, safeHttp, slotParts, telHref } from '../model/place';
import { DirectoryWebShell } from './DirectoryWebShell';
import { PlaceBody, usePrice } from './PlaceBody';

/**
 * CB-DIR-11 — `W_Dir_Place`: breadcrumb, gallery, details and a sticky booking panel. Public; the server hands over the
 * listing (`initial`) so the first HTML already carries the content for search engines. Booking: logged out → login
 * link; logged in → the app's own request flow (`/city-services/[id]/request`). No owner, application or contact
 * data of anyone but the venue is ever on this page.
 */
export function WebPlacePage({ id, initial }: { id: number; initial?: CityServiceDetail | null }) {
  const t = useTranslations('cityServices');
  const w = useTranslations('webDirectory');
  const q = useCityServiceListing(id, { anon: true, initialData: initial ?? undefined });

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="wcp-skel">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    const missing = (q.error as { response?: { status?: number } } | null)?.response?.status === 404;
    body = (
      <Card>
        <EmptyState
          icon={missing ? 'home' : 'warning'}
          title={missing ? w('place.notFoundTitle') : t('loadError')}
          body={missing ? w('place.notFoundBody') : undefined}
          action={
            <Link href={PLACES_HOME} className="wcs-link-btn">
              {w('place.toSearch')}
            </Link>
          }
        />
      </Card>
    );
  } else {
    body = <PlaceContent l={q.data} />;
  }

  return (
    <DirectoryWebShell appHref={`${APP_DIRECTORY}/${id}`}>
      <div className="wcp">{body}</div>
    </DirectoryWebShell>
  );
}

function PlaceContent({ l }: { l: CityServiceDetail }) {
  const t = useTranslations('cityServices');
  const w = useTranslations('webDirectory');
  const fmt = useFormatter();
  const locale = useLocale() as Locale;
  const [photo, setPhoto] = useState(0);
  const photos = l.photos.length > 0 ? l.photos : l.photo_url ? [l.photo_url] : [];
  const current = photos[Math.min(photo, photos.length - 1)];
  const dec = (v: number) => formatDecimal(v, locale);
  const num = (v: number) => formatNumber(v, locale);
  const where = [l.district, l.city.name].filter(Boolean).join('، ');
  const site = safeHttp(l.website_url);

  const actions: { key: string; icon: 'phone' | 'mapPin' | 'globe'; label: string; href: string; external: boolean }[] = [];
  if (l.phone) actions.push({ key: 'call', icon: 'phone', label: t('place.call'), href: telHref(l.phone), external: false });
  if (l.lat !== null && l.lng !== null) actions.push({ key: 'dir', icon: 'mapPin', label: t('place.directions'), href: neshanHref(l.lat, l.lng), external: true });
  if (site) actions.push({ key: 'web', icon: 'globe', label: t('place.website'), href: site, external: true });

  return (
    <article>
      <nav aria-label={w('place.crumbLabel')} className="wcp-crumbs">
        <Link href={PLACES_HOME}>{w('place.crumbRoot')}</Link>
        <span aria-hidden>/</span>
        <span>{l.city.name}</span>
        <span aria-hidden>/</span>
        <span>{l.category.name}</span>
        <span aria-hidden>/</span>
        <span aria-current="page">{l.name}</span>
      </nav>

      <header className="wcp-head">
        <div>
          <p className="csv-card-meta cdp-meta">
            <span className="csv-badge">{l.category.name}</span>
            {l.verified ? (
              <span className="csv-badge cdp-verified">
                <Icon name="shield" size={12} />
                {t('home.verified')}
              </span>
            ) : null}
          </p>
          <h1 className="wcp-h1">{l.name}</h1>
          <p className="csv-card-meta cdp-meta">
            {l.rating_avg !== null ? (
              <span className="cdh-rating">
                <Icon name="star" size={14} />
                {dec(l.rating_avg)} <span className="cdp-dim">({t('place.reviewCount', { n: num(l.rating_count) })})</span>
              </span>
            ) : null}
            {where ? <span>{where}</span> : null}
            {l.open_now !== null ? <span className={l.open_now ? 'cdh-open' : 'cdp-closed'}>{l.open_now ? t('home.openNow') : t('place.closedNow')}</span> : null}
          </p>
        </div>
        {actions.length > 0 ? (
          <ul className="cdp-actions wcp-actions">
            {actions.map((a) => (
              <li key={a.key}>
                {/* External links open with noopener noreferrer; only http(s) reaches here. */}
                <a className="cdp-action" href={a.href} {...(a.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}>
                  <Icon name={a.icon} size={18} />
                  {a.label}
                </a>
              </li>
            ))}
          </ul>
        ) : null}
      </header>

      <section aria-label={w('place.gallery')} className="wcp-gallery">
        <div className="wcp-hero">
          {current ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={current} alt={t('detail.photo', { n: fmt.number(photo + 1), name: l.name })} />
          ) : (
            <span className="csv-hero-art" aria-hidden>
              <Icon name={l.category.icon} size={72} strokeWidth={1.3} />
            </span>
          )}
        </div>
        {photos.length > 1 ? (
          <ul className="wcp-thumbs">
            {photos.slice(0, 5).map((p, i) => (
              <li key={`${i}-${p}`}>
                <button
                  type="button"
                  className={clsx('csv-strip-btn', i === photo && 'on')}
                  aria-label={t('detail.photo', { n: fmt.number(i + 1), name: l.name })}
                  aria-pressed={i === photo}
                  onClick={() => setPhoto(i)}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image */}
                  <img src={p} alt="" loading="lazy" />
                </button>
              </li>
            ))}
            {photos.length > 5 ? <li className="wcp-more">{w('place.allPhotos', { n: num(photos.length) })}</li> : null}
          </ul>
        ) : null}
      </section>

      <div className="wcp-cols">
        <div className="wcp-main">
          <PlaceBody l={l} />
        </div>
        <aside className="wcp-panel" aria-label={w('place.bookTitle')}>
          <BookingPanel l={l} />
        </aside>
      </div>
    </article>
  );
}

/** Booking panel: pick a service + slot, then the venue's own action by booking mode. */
function BookingPanel({ l }: { l: CityServiceDetail }) {
  const t = useTranslations('cityServices');
  const w = useTranslations('webDirectory');
  const locale = useLocale() as Locale;
  const price = usePrice();
  const mounted = useMounted();
  const authed = mounted && isAuthenticated();
  const [serviceId, setServiceId] = useState<number | null>(null);
  const [slotId, setSlotId] = useState<number | null>(null);
  const slots = useCityServiceSlots(l.id, serviceId);
  const shown = serviceId !== null ? (slots.data ?? []) : l.next_slots;
  const chosen = l.services.find((s) => s.id === serviceId)?.price_toman ?? minPrice(l.services);
  const href = safeHttp(l.booking_url);
  const slotLabel = (stamp: string) => {
    const p = slotParts(stamp);
    if (!p) return stamp;
    const d = fromApiDate(p.day);
    return `${formatWeekday(d, locale)} ${formatDayMonth(d, locale)} ${formatNumber(p.time, locale)}`;
  };
  const requestHref = (() => {
    const qs = new URLSearchParams();
    if (serviceId !== null) qs.set('service', String(serviceId));
    if (slotId !== null) qs.set('slot', String(slotId));
    const s = qs.toString();
    return `${APP_DIRECTORY}/${l.id}/request${s ? `?${s}` : ''}`;
  })();

  let action;
  if (l.booking_mode === 'link' && href) {
    action = (
      <a className="csv-go-btn wcp-go" href={href} target="_blank" rel="noopener noreferrer">
        {t('detail.linkCta')}
        <Icon name="arrowR" size={16} />
      </a>
    );
  } else if (l.booking_mode === 'phone' && l.phone) {
    action = (
      <a className="csv-go-btn wcp-go" href={telHref(l.phone)}>
        <Icon name="phone" size={16} />
        {t('detail.phoneCta')}
      </a>
    );
  } else if (l.booking_mode === 'request' || l.booking_mode === 'online') {
    action = authed ? (
      <Link href={requestHref} className="csv-go-btn wcp-go">
        {slotId !== null ? t('place.continue') : l.booking_mode === 'online' ? t('place.chooseTime') : t('detail.requestCta')}
      </Link>
    ) : (
      <div className="wcp-login">
        <strong>{w('place.loginTitle')}</strong>
        <p>{w('place.loginBody')}</p>
        <Link href="/signup" className="csv-go-btn wcp-go">
          {w('place.login')}
        </Link>
      </div>
    );
  } else {
    action = null;
  }

  return (
    <Card>
      <h2 className="wcs-h2">{w('place.bookTitle')}</h2>
      {chosen !== null && chosen !== undefined ? (
        <p className="wcp-price">
          <small>{serviceId !== null ? t('place.price') : t('place.from')}</small>
          <strong>{price(chosen)}</strong>
        </p>
      ) : null}
      {l.services.length > 0 ? (
        <div role="radiogroup" aria-label={t('place.servicesTitle')} className="wcp-svcs">
          {l.services.map((s) => (
            <button
              key={s.id}
              type="button"
              role="radio"
              aria-checked={serviceId === s.id}
              className={clsx('wcp-svc', serviceId === s.id && 'on')}
              onClick={() => {
                setServiceId(s.id);
                setSlotId(null);
              }}
            >
              <span>{s.name}</span>
              {s.price_toman !== null ? <small>{price(s.price_toman)}</small> : null}
            </button>
          ))}
        </div>
      ) : null}
      {shown.length > 0 ? (
        <>
          <h3 className="csv-sub">{t('place.slotsTitle')}</h3>
          <ul className="cdp-slots">
            {shown.map((s) => (
              <li key={s.id}>
                <button type="button" className={clsx('cdp-slot', slotId === s.id && 'on')} aria-pressed={slotId === s.id} disabled={!bookable(s)} onClick={() => setSlotId(s.id)}>
                  {slotLabel(s.starts_at)}
                  {!bookable(s) ? <small>{t('place.full')}</small> : null}
                </button>
              </li>
            ))}
          </ul>
        </>
      ) : null}
      {action}
      <p className="cdp-hint">{l.booking_mode === 'link' ? t('detail.linkNote') : l.booking_mode === 'phone' ? t('detail.phoneNote') : w('place.bookingNote')}</p>
    </Card>
  );
}
