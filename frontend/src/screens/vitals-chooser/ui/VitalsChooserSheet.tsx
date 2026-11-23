'use client';

import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { Icon, IconCircle, type IconName, type Tone } from '@/shared/ui';
import { pilot } from '@/shared/pilot';
import '@/shared/ui/styles/rad.css';

/** Registry id of this sheet (`app/sheets/registry.tsx`). */
export const VITALS_CHOOSER_SHEET = 'vitals-add';

/*
 * The add forms (routes.md `/vitals/<slug>/new`) with the hub's look (entities/vital VITAL_LOOK). Spelled out here
 * rather than read from the entity barrel: that barrel pulls the whole `vitals` namespace into every route, and
 * this sheet can open over any of them.
 */
const CHOICES: ReadonlyArray<{ key: 'bp' | 'hr' | 'glucose'; href: string; icon: IconName; tone: Tone }> = [
  { key: 'bp', href: '/vitals/bp/new', icon: 'gauge', tone: 'brand' },
  { key: 'hr', href: '/vitals/heart-rate/new', icon: 'heartLine', tone: 'bloom' },
  { key: 'glucose', href: '/vitals/glucose/new', icon: 'glucose', tone: 'warm' },
];

/**
 * «ثبت علائم حیاتی» — what the FAB's log sheet and the Today quick-access tile open: one row per type straight
 * into its add form, plus the vitals hub for history and reports. Sheet content only (CLAUDE.md §4.1); following a
 * row navigates, which closes the sheet. Copy lives in `nav` because the sheet can open over any route.
 */
export function VitalsChooserSheet() {
  const t = useTranslations('nav.vitalsChooser');

  return (
    <div className="rad-list">
      {CHOICES.map((c) => (
        <Link key={c.key} href={c.href} className="rad-row" {...pilot(`vt.choice:${c.key}`)}>
          <IconCircle icon={c.icon} tone={c.tone} size="md" />
          <span className="rad-body">
            <span className="rad-title">{t(`${c.key}.title`)}</span>
            <span className="rad-desc">{t(`${c.key}.sub`)}</span>
          </span>
          <Icon name="chevronLeft" size={18} strokeWidth={2} className="rad-chev" />
        </Link>
      ))}
      <Link href="/vitals" className="rad-row" {...pilot('vt.choice:hub')}>
        <IconCircle icon="chart" tone="data" size="md" />
        <span className="rad-body">
          <span className="rad-title">{t('hub.title')}</span>
          <span className="rad-desc">{t('hub.sub')}</span>
        </span>
        <Icon name="chevronLeft" size={18} strokeWidth={2} className="rad-chev" />
      </Link>
    </div>
  );
}

export function VitalsChooserTitle() {
  return <>{useTranslations('nav.vitalsChooser')('title')}</>;
}
