'use client';

import { useLocale, useTranslations } from 'next-intl';

import { bookingWhen, type ServiceSection, type ServiceTile, type UpcomingBooking } from '@/entities/service-hub';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber, monthName, toParts } from '@/shared/lib/date';
import { Icon, IconCircle, ListGroup, ListRow, SectionTitle, StatusPill } from '@/shared/ui';

/*
 * The «خدمات» hub sections (B-N7-01, v17_Main). Copy comes from the API (admin catalog) with the bundled strings as
 * fallback. A destination with status `soon` is plain text plus a «به‌زودی» pill — never a link, never a button.
 */

function SoonPill() {
  const t = useTranslations('services');
  return <StatusPill tone="neutral">{t('soon')}</StatusPill>;
}

export function SearchLink({ section }: { section: ServiceSection }) {
  const t = useTranslations('services');
  const placeholder = section.title ?? t('search.placeholder');
  const body = (
    <>
      <Icon name="search" size={18} className="svc-search-icon" />
      <span className="svc-search-text">{placeholder}</span>
    </>
  );
  return (
    <div className="svc-sec">
      {section.href ? (
        <Link href={section.href} className="svc-search" aria-label={`${t('search.label')}: ${placeholder}`}>
          {body}
        </Link>
      ) : (
        <div className="svc-search is-soon" aria-label={t('soonLabel', { title: t('search.label') })} role="note">
          {body}
        </div>
      )}
    </div>
  );
}

export function BookingCard({ booking, locale }: { booking: UpcomingBooking; locale: Locale }) {
  const t = useTranslations('services');
  const when = bookingWhen(booking);
  const parts = toParts(when.date, locale);
  const title = t('booking.title', { kind: t(`booking.kind.${booking.kind}`), name: booking.providerName });
  return (
    <section className="svc-sec" aria-label={t('booking.label')}>
      <div className="svc-booking">
        <div className="svc-booking-date" aria-hidden>
          <span className="svc-booking-day">{formatNumber(parts.day, locale)}</span>
          <span className="svc-booking-month">{monthName(parts.month, locale)}</span>
        </div>
        <div className="svc-booking-text">
          <p className="svc-booking-title">{title}</p>
          <p className="svc-booking-when">
            {t('booking.when', { days: when.daysAway, time: formatNumber(when.time, locale) })}
          </p>
        </div>
        {booking.href ? (
          <Link
            href={booking.href}
            className="svc-booking-cta"
            aria-label={t('booking.detailsLabel', { name: booking.providerName })}
          >
            {t('booking.details')}
          </Link>
        ) : null}
      </div>
    </section>
  );
}

function Tile({ tile, className }: { tile: ServiceTile; className: string }) {
  const t = useTranslations('services');
  const title = tile.title ?? tile.code;
  const sub = tile.count !== null && tile.count > 0 ? t('care.docCount', { count: tile.count }) : tile.subtitle;
  const live = tile.status === 'live' && tile.href;
  return (
    <li className={live ? `${className} is-live` : className}>
      <div className="svc-tile-top">
        <IconCircle icon={tile.icon} tone={tile.tone} />
        {live ? null : <SoonPill />}
      </div>
      <p className="svc-tile-title">
        {live ? (
          <Link href={tile.href as string} className="svc-tile-hit">
            {title}
          </Link>
        ) : (
          title
        )}
      </p>
      {sub ? <p className="svc-tile-sub">{sub}</p> : null}
    </li>
  );
}

export function CareGrid({ section, tiles }: { section: ServiceSection; tiles: ServiceTile[] }) {
  const t = useTranslations('services');
  return (
    <section className="svc-sec" aria-labelledby="svc-care">
      <SectionTitle id="svc-care" title={section.title ?? t('care.title')} />
      <ul className="svc-grid">
        {tiles.map((tile) => (
          <Tile key={tile.code} tile={tile} className="svc-tile" />
        ))}
      </ul>
    </section>
  );
}

export function CheckupsLink({ section }: { section: ServiceSection }) {
  const t = useTranslations('services');
  const label = section.title ?? t('checkups.title');
  return (
    <div className="svc-sec">
      {section.href ? (
        <Link href={section.href} className="svc-inline-link">
          <Icon name="clock" size={18} />
          <span>{label}</span>
        </Link>
      ) : (
        <p className="svc-inline-link is-soon">
          <Icon name="clock" size={18} />
          <span>{label}</span>
          <SoonPill />
        </p>
      )}
    </div>
  );
}

export function ProgramsRow({ section, tiles }: { section: ServiceSection; tiles: ServiceTile[] }) {
  const t = useTranslations('services');
  return (
    <section className="svc-sec" aria-labelledby="svc-programs">
      <div className="svc-sec-head">
        <h2 id="svc-programs" className="nb-sect-title">
          {section.title ?? t('programs.title')}
        </h2>
        <p className="svc-sec-sub">{section.subtitle ?? t('programs.subtitle')}</p>
      </div>
      <ul className="svc-programs">
        {tiles.map((tile) => (
          <Tile key={tile.code} tile={tile} className="svc-tile svc-program" />
        ))}
      </ul>
    </section>
  );
}

export function MotherChildCard({ section }: { section: ServiceSection }) {
  const t = useTranslations('services');
  const caption = section.caption ?? t('motherChild.caption');
  const pill = (
    <>
      <Icon name="mapPin" size={16} />
      <span>{caption}</span>
      {section.href ? null : <SoonPill />}
    </>
  );
  return (
    <section className="svc-sec" aria-labelledby="svc-mc">
      <SectionTitle id="svc-mc" title={section.title ?? t('motherChild.title')} />
      <div className="svc-map">
        <div className="svc-map-art" aria-hidden>
          <span className="svc-map-road is-h1" />
          <span className="svc-map-road is-h2" />
          <span className="svc-map-road is-v1" />
          <span className="svc-map-road is-v2" />
          <span className="svc-map-block is-a" />
          <span className="svc-map-block is-b" />
          <span className="svc-map-pin is-a">
            <Icon name="drop" size={16} />
          </span>
          <span className="svc-map-pin is-b">
            <Icon name="home" size={16} />
          </span>
        </div>
        {section.href ? (
          <Link href={section.href} className="svc-map-pill">
            {pill}
          </Link>
        ) : (
          <p className="svc-map-pill">{pill}</p>
        )}
      </div>
    </section>
  );
}

export function LearningCard({ section }: { section: ServiceSection }) {
  const t = useTranslations('services');
  const router = useRouter();
  const href = section.href;
  return (
    <section className="svc-sec" aria-labelledby="svc-learn">
      <SectionTitle id="svc-learn" title={section.title ?? t('learning.title')} />
      <ListGroup className="svc-learn">
        <ListRow
          icon="gradCap"
          iconTone="period"
          title={section.subtitle ?? t('learning.card')}
          description={section.caption ?? t('learning.caption')}
          onClick={href ? () => router.push(href) : undefined}
          trailing={href ? undefined : <SoonPill />}
        />
      </ListGroup>
    </section>
  );
}

export function ShopCard({ section }: { section: ServiceSection }) {
  const t = useTranslations('services');
  return (
    <section className="svc-sec" aria-labelledby="svc-shop">
      <div className="svc-shop">
        <div className="svc-shop-head">
          <h2 id="svc-shop" className="svc-shop-title">
            {section.title ?? t('shop.title')}
          </h2>
          <span className="svc-shop-tag">{t('shop.tag')}</span>
          <span className="svc-shop-end">
            {section.href ? (
              <Link href={section.href} className="svc-shop-enter">
                {t('shop.enter')}
              </Link>
            ) : (
              <SoonPill />
            )}
          </span>
        </div>
        {section.categories.length ? (
          <ul className="svc-shop-cats">
            {section.categories.map((c, i) => (
              <li key={c.code} className={i % 2 ? 'svc-shop-cat is-alt' : 'svc-shop-cat'}>
                <span className="svc-shop-art" aria-hidden>
                  <Icon name={c.icon} size={44} strokeWidth={1.4} />
                </span>
                <span className="svc-shop-cat-title">{c.title ?? c.code}</span>
              </li>
            ))}
          </ul>
        ) : null}
        <p className="svc-shop-note">{section.subtitle ?? t('shop.note')}</p>
      </div>
    </section>
  );
}

/** Always rendered (also while loading / on error): `section` null = bundled copy and 115. */
export function SosCard({ section }: { section: ServiceSection | null }) {
  const t = useTranslations('services');
  const locale = useLocale() as Locale;
  const phone = section?.phone ?? '115';
  return (
    <div className="svc-sos">
      <IconCircle icon="warning" tone="danger" size="md" />
      <div className="svc-sos-text">
        <p className="svc-sos-title">{section?.title ?? t('sos.title')}</p>
        <p className="svc-sos-body">{section?.subtitle ?? t('sos.body')}</p>
      </div>
      <a className="svc-sos-call" href={`tel:${phone}`} aria-label={t('sos.call', { phone: formatNumber(phone, locale) })}>
        <bdi>{formatNumber(phone, locale)}</bdi>
      </a>
    </div>
  );
}
