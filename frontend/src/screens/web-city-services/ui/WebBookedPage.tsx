'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useCityBooking } from '@/entities/city-service';
import { Link, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { isAuthenticated } from '@/shared/session';
import { Card, EmptyState, Icon, InfoNote, Skeleton, SkeletonGroup, StatusPill } from '@/shared/ui';

import { APP_DIRECTORY, neshanHref, PLACES_HOME, slotParts, telHref } from '../model/place';
import { DirectoryWebShell } from './DirectoryWebShell';
import { usePrice } from './PlaceBody';
import '@/shared/ui/styles/csv.css';
import '@/shared/ui/styles/cdp.css';
import '@/shared/ui/styles/cdb.css';
import './wcs.css';
import './wcp.css';
import './wcb.css';

/** The booking's day/time from the API stamps (preferred day, or the chosen slot's start). */
function whenOf(b: { slot: { starts_at: string | null } | null; preferred_date: string }): { day: string; time: string | null } {
  const p = b.slot?.starts_at ? slotParts(b.slot.starts_at) : null;
  return p ?? { day: b.preferred_date.slice(0, 10), time: null };
}

/**
 * CB-DIR-11 — `W_Dir_Booked` on the web. A booking is private: without a session the page only asks to log in, and
 * the API answers only for the owner. Honest status wording («request» until the team confirms), pay at the place.
 * Cancelling and rescheduling stay in the app (the link opens the app's booked screen).
 */
export function WebBookedPage({ id }: { id: number }) {
  const t = useTranslations('cityServices');
  const w = useTranslations('webDirectory');
  const locale = useLocale() as Locale;
  const price = usePrice();
  const mounted = useMounted();
  const authed = mounted && isAuthenticated();
  const q = useCityBooking(id);
  const num = (v: number | string) => formatNumber(v, locale);

  let body;
  if (!mounted) {
    body = null;
  } else if (!authed) {
    body = (
      <Card>
        <EmptyState
          icon="lock"
          title={w('booked.loginTitle')}
          body={w('booked.loginBody')}
          action={
            <Link href="/signup" className="csv-go-btn wcp-go">
              {w('booked.login')}
            </Link>
          }
        />
      </Card>
    );
  } else if (q.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="wcp-skel">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('bookings.loadError')}
          action={
            <button type="button" className="wcs-link-btn" onClick={() => void q.refetch()}>
              {t('retry')}
            </button>
          }
        />
      </Card>
    );
  } else {
    const b = q.data;
    const l = b.listing;
    const when = whenOf(b);
    const date = fromApiDate(when.day);
    const tone = b.status === 'confirmed' ? 'success' : b.status === 'requested' ? 'brand' : b.status === 'completed' ? 'data' : b.status === 'rejected' ? 'danger' : 'neutral';
    const rows: { icon: 'mapPin' | 'calendar' | 'user' | 'note' | 'info'; title: string; sub: string | null }[] = [
      { icon: 'mapPin', title: b.service?.name ?? l.name, sub: [l.name, l.district].filter(Boolean).join(' · ') },
      { icon: 'calendar', title: `${formatWeekday(date, locale)} ${formatDayMonth(date, locale)}`, sub: when.time ? num(when.time) : t('booked.dayOnly') },
    ];
    if (b.child) rows.push({ icon: 'user', title: b.child.name, sub: t('booked.childAge', { n: num(b.child.age_months) }) });
    if (b.note) rows.push({ icon: 'note', title: t('booked.yourNote'), sub: b.note });
    rows.push({ icon: 'info', title: b.price_toman !== null ? price(b.price_toman) : t('book.payAtPlace'), sub: b.price_toman !== null ? t('book.payAtPlace') : t('booked.noPrice') });

    body = (
      <div className="wcb-grid">
        <div>
          <div className="cdb-hero wcb-hero">
            <span className={`cdb-hero-ic is-${b.status}`} aria-hidden>
              <Icon name={b.status === 'rejected' || b.status === 'cancelled' ? 'x' : b.status === 'requested' ? 'clock' : 'check'} size={34} strokeWidth={2.2} />
            </span>
            <h1 className="cdb-hero-t">{t(`booked.title.${b.status}`)}</h1>
            <p className="cdb-hero-s">{t(`booked.lead.${b.status}`, { name: l.name })}</p>
            <StatusPill tone={tone}>{t(`bookings.status.${b.status}`)}</StatusPill>
            {b.code ? <p className="cdb-code">{t('booked.code', { code: b.code })}</p> : null}
          </div>
          {b.status === 'requested' ? <InfoNote icon="info">{t('booked.waiting')}</InfoNote> : null}
        </div>
        <div className="wcb-side">
          <Card className="cdb-rows">
            {rows.map((r) => (
              <div key={r.icon} className="cdb-row">
                <span className="cdb-row-ic" aria-hidden>
                  <Icon name={r.icon} size={18} />
                </span>
                <span className="cdb-opt-main">
                  <strong>{r.title}</strong>
                  {r.sub ? <small>{r.sub}</small> : null}
                </span>
              </div>
            ))}
          </Card>
          <div className="cdb-actions">
            {l.lat !== null && l.lng !== null ? (
              <a className="csv-go-btn" href={neshanHref(l.lat, l.lng)} target="_blank" rel="noopener noreferrer">
                <Icon name="mapPin" size={16} />
                {t('place.directions')}
              </a>
            ) : null}
            {l.phone ? (
              <a className="csv-go-btn" href={telHref(l.phone)}>
                <Icon name="phone" size={16} />
                {t('booked.callPlace')}
              </a>
            ) : null}
          </div>
          <Card>
            <h2 className="csv-sub">{w('booked.appTitle')}</h2>
            <p className="cdp-hint">{w('booked.appBody')}</p>
            <Link href={`${APP_DIRECTORY}/booked/${b.id}`} className="csv-link">
              {w('booked.openApp')}
            </Link>
          </Card>
          <Link href={PLACES_HOME} className="csv-link">
            {w('booked.toSearch')}
          </Link>
        </div>
      </div>
    );
  }

  return (
    <DirectoryWebShell appHref={`${APP_DIRECTORY}/booked/${id}`}>
      <div className="wcp">{body}</div>
    </DirectoryWebShell>
  );
}
