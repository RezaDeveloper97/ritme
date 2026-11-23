'use client';

import { useLocale, useTranslations } from 'next-intl';

import { COMPANION_SECTIONS, type CompanionGrants, sharedSections } from '@/entities/companion';
import { useCycleToday } from '@/entities/cycle';
import { useUserProfile } from '@/entities/user';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';
import '@/shared/ui/styles/cmp.css';

/** The phase keys the preview can name; anything else reads as «not clear yet». */
const PREVIEW_PHASES = ['menstrual', 'follicular', 'fertile', 'luteal', 'period_expected'] as const;
type PreviewPhase = (typeof PREVIEW_PHASES)[number] | 'unknown';

function previewPhase(main: string | null | undefined): PreviewPhase {
  return (PREVIEW_PHASES as readonly string[]).includes(main ?? '') ? (main as PreviewPhase) : 'unknown';
}

/**
 * «همراهت این را می‌بیند» (companion-v2 `preview`): a read-only mini card of
 * what the companion's panel shows — her OWN current phase in probabilistic
 * words and the next-period estimate (only while the cycle section is shared),
 * one line per other section she opened with its level — then «آنچه او
 * نمی‌بیند»: the sections left on «نبیند». Nothing leaves the device here.
 */
export function CompanionPreview({ grants }: { grants: CompanionGrants }) {
  const t = useTranslations('companions.flow.preview');
  const tc = useTranslations('companions');
  const locale = useLocale() as Locale;
  const today = useCycleToday();
  const profile = useUserProfile();
  const view = today.data?.cycleView ?? null;
  const phase = previewPhase(view?.mainPhase);
  const days = view?.daysToPeriod ?? null;
  const name = profile.data?.name?.trim() || null;
  const cycleShared = grants.cycle !== 'none';
  const others = sharedSections(grants).filter((s) => s.section !== 'cycle');
  const hidden = COMPANION_SECTIONS.filter((s) => grants[s] === 'none');

  let estimate: string;
  if (today.isPending && today.fetchStatus !== 'idle') estimate = t('loading');
  else if (days !== null && phase !== 'menstrual') estimate = t('estimate', { days: formatNumber(days, locale) });
  else estimate = t('noEstimate');

  return (
    <>
      <div className="cmp2-mini" aria-label={t('cardLabel')} role="group">
        <div className="cmp2-mini-top">
          <span className="cmp2-mini-avatar" aria-hidden />
          <span className="cmp2-mini-name">
            <b>{name ? t('cycleOf', { name }) : t('cycleOfYou')}</b>
            <small>{t('meta')}</small>
          </span>
        </div>
        {cycleShared ? (
          <div className="cmp2-mini-phase">
            <span className="cmp2-mini-label">{t('phaseLabel')}</span>
            <b className="cmp2-mini-value">{t(`phase.${phase}`)}</b>
            <span className="cmp2-mini-note">{estimate}</span>
          </div>
        ) : null}
        {others.map(({ section, level }) => (
          <p key={section} className="cmp2-mini-tip">
            <Icon name="eye" size={16} className="cmp2-mini-tip-icon" />
            <span>{tc('chip', { section: tc(`sections.${section}`), level: tc(`levels.${level}`) })}</span>
          </p>
        ))}
        {!cycleShared && others.length === 0 ? (
          <p className="cmp2-mini-tip is-muted">
            <Icon name="info" size={16} className="cmp2-mini-tip-icon" />
            <span>{tc('nothingShared')}</span>
          </p>
        ) : null}
      </div>
      {hidden.length ? (
        <div className="nb-card cmp2-aside">
          <Icon name="shield" size={18} className="cmp2-aside-icon" />
          <p className="cmp2-aside-text">
            <b>{t('hiddenTitle')}</b>{' '}
            {t('hiddenSections', { sections: hidden.map((x) => tc(`sections.${x}`)).join(tc('listSeparator')) })}
          </p>
        </div>
      ) : null}
    </>
  );
}
