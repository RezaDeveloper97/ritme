'use client';

import { useLocale, useTranslations } from 'next-intl';

import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import {
  HeaderButton,
  HubHeader,
  IconCircle,
  InfoNote,
  ListGroup,
  ListRow,
  SectionTitle,
  SkyLayer,
  StatusPill,
  type IconName,
  type Tone,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

type TileKey = 'assistant' | 'doctors' | 'labs' | 'record' | 'vitals' | 'insurance';

/** v17_Main «مراقبت سلامت» grid, in artboard order. Each lands with its own N6/N7/N10 task. */
const CARE_TILES: ReadonlyArray<{ key: TileKey; icon: IconName; tone: Tone }> = [
  { key: 'assistant', icon: 'sparkle', tone: 'brand' },
  { key: 'doctors', icon: 'stetho', tone: 'data' },
  { key: 'record', icon: 'note', tone: 'brand' },
  { key: 'labs', icon: 'flask', tone: 'period' },
  { key: 'vitals', icon: 'heartLine', tone: 'period' },
  { key: 'insurance', icon: 'shield', tone: 'data' },
];

/** Tiles whose screens exist (the rest show «به‌زودی»). */
const LIVE_TILES: Partial<Record<TileKey, string>> = {
  vitals: '/vitals', // B-N6-02
};

/**
 * «خدمات» tab (B-N1-04). A placeholder hub so the new tab works now: the care
 * grid shows what is coming, the two services that already exist (checkups,
 * reminders) are live, and the emergency note is always there. The real hub —
 * appointment card, programs, map, courses, shop — is B-N7-01 (`v17_Main`).
 * Static content: no loading / error state to show.
 */
export function ServicesPage() {
  const t = useTranslations('services');
  const router = useRouter();
  const locale = useLocale() as Locale;

  return (
    <div className="view svc-page">
      <SkyLayer />
      <div className="scroll svc-scroll">
        <HubHeader
          date={t('subtitle')}
          greeting={t('title')}
          actions={
            <HeaderButton label={t('notifications')} icon="bell" variant="soft" onClick={() => openSheet('notifications')} />
          }
        />

        <section className="svc-sec" aria-labelledby="svc-care">
          <SectionTitle id="svc-care" title={t('careTitle')} />
          <ul className="svc-grid">
            {CARE_TILES.map((tile) => {
              const href = LIVE_TILES[tile.key];
              return (
                <li key={tile.key} className={href ? 'svc-tile is-live' : 'svc-tile'}>
                  <div className="svc-tile-top">
                    <IconCircle icon={tile.icon} tone={tile.tone} size="sm" />
                    {href ? null : <StatusPill tone="neutral">{t('soon')}</StatusPill>}
                  </div>
                  <p className="svc-tile-title">
                    {href ? (
                      <Link href={href} className="svc-tile-hit">
                        {t(`tiles.${tile.key}.title`)}
                      </Link>
                    ) : (
                      t(`tiles.${tile.key}.title`)
                    )}
                  </p>
                  <p className="svc-tile-sub">{t(`tiles.${tile.key}.sub`)}</p>
                </li>
              );
            })}
          </ul>
        </section>

        <section className="svc-sec" aria-labelledby="svc-now">
          <SectionTitle id="svc-now" title={t('availableTitle')} />
          <ListGroup>
            <ListRow
              icon="stetho"
              iconTone="data"
              title={t('checkups.title')}
              description={t('checkups.sub')}
              onClick={() => router.push('/checkups')}
            />
            <ListRow
              icon="pill"
              iconTone="warm"
              title={t('reminders.title')}
              description={t('reminders.sub')}
              onClick={() => router.push('/reminders')}
            />
          </ListGroup>
          <InfoNote>{t('building')}</InfoNote>
        </section>

        <div className="svc-sos">
          <IconCircle icon="warning" tone="danger" size="md" />
          <div className="svc-sos-text">
            <p className="svc-sos-title">{t('sos.title')}</p>
            <p className="svc-sos-body">{t('sos.body')}</p>
          </div>
          <a className="svc-sos-call" href="tel:115" aria-label={t('sos.call')}>
            <bdi>{formatNumber(115, locale)}</bdi>
          </a>
        </div>
      </div>
      <BottomNav />
    </div>
  );
}
