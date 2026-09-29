'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { TTC_PHASES, type TtcPhase } from '../model/ttc';

/** Legend tone per phase: period red (§10.2), fertile amber, luteal violet, follicular neutral. */
const TONE: Record<TtcPhase, string> = {
  period: 'fert-tone-period',
  follicular: 'fert-tone-muted',
  fertile: 'fert-tone-amber',
  luteal: 'fert-tone-brand',
};

/**
 * `v19_Main` phase legend under the ring: four outlined pills, the current
 * phase filled. It is a legend, not a control, so it renders as a list.
 */
export function TtcPhasePills({ active }: { active: TtcPhase | null }) {
  const t = useTranslations('fertility.home');
  return (
    <ul className="flex flex-wrap justify-center gap-2" aria-label={t('phasesLabel')}>
      {TTC_PHASES.map((phase) => (
        <li
          key={phase}
          className={clsx('fert-pill', TONE[phase], phase === active && 'is-on')}
          aria-current={phase === active ? 'step' : undefined}
        >
          {t(`phases.${phase}`)}
        </li>
      ))}
    </ul>
  );
}
