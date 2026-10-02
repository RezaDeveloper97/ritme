'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Locale, Link, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Icon,
  ListGroup,
  ListRow,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  UrgentCard,
} from '@/shared/ui';

import { useMenoAlerts } from '../api/alerts';
import { type MenoAlert, alertLook, primaryAlert, tellEarlyAlerts } from '../model/alerts';

/** The M3 appointment form — the doctors directory (N7) doesn't exist yet. */
const BOOK_HREF = '/reminders/appointment/new?kind=in_person';

/**
 * «خونریزی بعد از یائسگی» (`/menopause/alert`, CB-MENO-09, nbl_Meno_Alert):
 * the danger card of the primary `meno_alerts` item, a gynaecologist
 * appointment (M3 form), and the other warning signs to report early, with a
 * one-tap 115 call on the emergency one. Reached from the home's bleeding card
 * and the `postmenopausal_bleeding` message. A back-header screen: no bottom
 * nav. The «آماده کردن گزارش برای پزشک» CTA waits for the doctor report
 * (CB-MENO-11) — not rendered until that route exists.
 * Copy is catalog content ([needs clinical review]); the strings here are only
 * the fallback when the catalog can't be read.
 */
export function MenopauseAlertPage() {
  const t = useTranslations('menopause.alert');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const query = useMenoAlerts(locale);
  const items = query.data ?? [];
  const primary = primaryAlert(items);
  const others = tellEarlyAlerts(items);

  return (
    <div className="view mal-page">
      <SkyLayer />
      <div className="scroll mal-scroll">
        <ScreenHeader
          title={primary?.title ?? t('title')}
          onBack={() => router.push('/home')}
          backLabel={t('back')}
        />
        <div className="mal-body">
          <UrgentCard title={primary?.cta ?? t('cardTitle')} urgent={false} className="mal-card">
            <p className="mal-card-body">{primary?.body ?? t('cardBody')}</p>
          </UrgentCard>

          <div className="mal-actions">
            <Link href={BOOK_HREF} className="nb-btn is-primary is-block mal-book">
              <Icon name="stetho" size={20} />
              {t('book')}
            </Link>
          </div>

          <section className="mal-sec" aria-labelledby="mal-early-title">
            <h2 id="mal-early-title" className="mal-sec-title">
              {t('tellEarly')}
            </h2>
            {query.isPending ? (
              <SkeletonGroup label={t('loading')}>
                <Skeleton shape="card" />
              </SkeletonGroup>
            ) : query.isError ? (
              <div className="nb-card mal-error" role="alert">
                <p className="mal-error-text">{t('loadError')}</p>
                <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
                  {t('retry')}
                </SecondaryButton>
              </div>
            ) : (
              <ListGroup className="mal-list">
                {others.map((alert) => (
                  <AlertRow key={alert.code} alert={alert} />
                ))}
              </ListGroup>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}

function AlertRow({ alert }: { alert: MenoAlert }) {
  const t = useTranslations('menopause.alert');
  const locale = useLocale() as Locale;
  const look = alertLook(alert);
  const hotline = alert.hotline;
  return (
    <ListRow
      icon={look.icon}
      iconTone={look.tone}
      title={alert.title ?? alert.body}
      description={alert.title ? alert.body : null}
      trailing={
        hotline ? (
          <a className="nb-hotline mal-call" href={`tel:${hotline}`}>
            <Icon name="phone" size={16} />
            <span className="sr-only">{t('call')}</span>
            <span className="nb-hotline-num">{formatNumber(hotline, locale)}</span>
          </a>
        ) : null
      }
    />
  );
}
