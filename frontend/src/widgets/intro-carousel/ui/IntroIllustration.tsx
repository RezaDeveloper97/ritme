import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import {
  INTRO_CYCLE_TODAY,
  INTRO_NEXT_PERIOD_DAYS,
  INTRO_PREGNANCY_WEEK,
  type IntroSlideId,
} from '../model/slides';
import { CycleDotRing, PregnancyDotRing } from './DotRing';

type ToneName = 'brand' | 'data' | 'warm' | 'period' | 'bloom';

/** Stroke paths of the artboard icons (24×24, 1.8 stroke) — not in the shared Icon set. */
const ART_ICONS = {
  gauge: <><path d="M4 16a8 8 0 1116 0" /><path d="M12 16l4-5" /></>,
  flask: <><path d="M9 3h6M10 3v6l-5 9a2 2 0 002 3h10a2 2 0 002-3l-5-9V3" /><path d="M8 15h8" /></>,
  stetho: <><path d="M6 3v6a6 6 0 0012 0V3" /><path d="M12 15v3a3 3 0 006 0v-3" /><circle cx="18" cy="12" r="2" /></>,
  sparkle: <path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" />,
  lock: <><rect x="5" y="11" width="14" height="9" rx="2" /><path d="M8 11V8a4 4 0 018 0v3" /></>,
  shield: <><path d="M12 3l7 3v5c0 5-3.5 8.5-7 10-3.5-1.5-7-5-7-10V6z" /><path d="M9 12l2 2 4-4" /></>,
  doc: <><path d="M7 3h7l5 5v13H7z" /><path d="M14 3v5h5" /></>,
  heart: <path d="M12 20s-7-4.5-7-10a4 4 0 017-2.5A4 4 0 0119 10c0 5.5-7 10-7 10z" />,
  crown: <path d="M3 8l4 4 5-7 5 7 4-4-2 11H5z" />,
} as const;

type ArtIconName = keyof typeof ART_ICONS;

function Disc({ icon, tone, size }: { icon: ArtIconName; tone: ToneName; size: 'md' | 'lg' }) {
  return (
    <span aria-hidden className={`ib-disc is-${size} nb-tone-${tone}`}>
      <svg viewBox="0 0 24 24" focusable="false" className="ib-disc-ic">
        {ART_ICONS[icon]}
      </svg>
    </span>
  );
}

function InfoRow({ icon, tone, title, sub }: { icon: ArtIconName; tone: ToneName; title: ReactNode; sub: ReactNode }) {
  return (
    <li className="ib-row">
      <Disc icon={icon} tone={tone} size="md" />
      <div className="ib-row-t">
        <b className="ib-row-title">{title}</b>
        <span className="ib-row-sub">{sub}</span>
      </div>
    </li>
  );
}

/** The artwork under each slide's title (inline SVG / markup from the artboards). */
export function IntroIllustration({ id }: { id: IntroSlideId }) {
  const t = useTranslations('welcome.slides');

  switch (id) {
    case 'cycle':
      return (
        <CycleDotRing today={INTRO_CYCLE_TODAY} label={t('cycle.ringLabel')}>
          <span className="ib-ring-kicker">{t('cycle.nextPeriod')}</span>
          <span className="ib-ring-num">{t('cycle.days', { n: INTRO_NEXT_PERIOD_DAYS })}</span>
          <span className="ib-ring-unit">{t('cycle.daysUnit')}</span>
        </CycleDotRing>
      );
    case 'journey':
      return (
        <PregnancyDotRing week={INTRO_PREGNANCY_WEEK} label={t('journey.ringLabel')}>
          <span className="ib-ring-week">{t('journey.week', { n: INTRO_PREGNANCY_WEEK })}</span>
          <span className="ib-ring-size">{t('journey.size')}</span>
        </PregnancyDotRing>
      );
    case 'health':
      return (
        <ul className="ib-tiles">
          {([
            ['vitals', 'gauge', 'brand'],
            ['labs', 'flask', 'data'],
            ['doctor', 'stetho', 'bloom'],
            ['assistant', 'sparkle', 'warm'],
          ] as const).map(([key, icon, tone]) => (
            <li key={key} className="ib-tile">
              <Disc icon={icon} tone={tone} size="lg" />
              <b className="ib-tile-t">{t(`health.${key}`)}</b>
            </li>
          ))}
        </ul>
      );
    case 'privacy':
      return (
        <ul className="ib-list">
          <InfoRow icon="lock" tone="data" title={t('privacy.ownTitle')} sub={t('privacy.ownSub')} />
          <InfoRow icon="shield" tone="brand" title={t('privacy.consentTitle')} sub={t('privacy.consentSub')} />
          <InfoRow icon="doc" tone="warm" title={t('privacy.deleteTitle')} sub={t('privacy.deleteSub')} />
        </ul>
      );
    case 'free':
      return (
        <div className="ib-free">
          <div className="ib-free-card">
            <b className="ib-free-title">{t('free.cardTitle')}</b>
            <ul className="ib-free-chips">
              {([
                ['cycle', 'period'],
                ['pregnancy', 'bloom'],
                ['menopause', 'brand'],
                ['child', 'data'],
              ] as const).map(([key, tone]) => (
                <li key={key} className={`ib-free-chip nb-tone-${tone}`}>{t(`free.${key}`)}</li>
              ))}
            </ul>
          </div>
          <ul className="ib-list">
            <InfoRow icon="heart" tone="data" title={t('free.accessTitle')} sub={t('free.accessSub')} />
            <InfoRow icon="crown" tone="warm" title={t('free.revenueTitle')} sub={t('free.revenueSub')} />
          </ul>
        </div>
      );
  }
}
