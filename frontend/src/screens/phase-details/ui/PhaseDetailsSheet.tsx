'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCycleToday } from '@/entities/cycle';
import { PlusDetailLock, usePlusDetailLocked } from '@/entities/plus';
import { usePhaseContent, type PhaseSectionKey } from '@/entities/phase-content';
import type { Locale } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { useMounted } from '@/shared/lib/use-mounted';
import { closeSheet } from '@/shared/sheet';
import { EmptyState, IconCircle, InfoNote, PrimaryButton, Skeleton, SkeletonGroup, type IconName, type Tone } from '@/shared/ui';

import { whyThisDate } from '../model/why';
import { availableTabs, phaseDayRange, TAB_SECTIONS, type PhaseTab } from '../model/tabs';

/** Icon + tone per section so each item reads at a glance (tokens only, via tone classes). */
const SECTION_LOOK: Record<PhaseSectionKey, { icon: IconName; tone: Tone }> = {
  hormonal_changes: { icon: 'sparkle', tone: 'brand' },
  symptom_prediction: { icon: 'plus', tone: 'warm' },
  vaginal_discharge: { icon: 'drop', tone: 'data' },
  sleep: { icon: 'moon', tone: 'brand' },
  skin_care: { icon: 'smile', tone: 'bloom' },
  nutrition: { icon: 'apple', tone: 'success' },
  exercise: { icon: 'walk', tone: 'warm' },
  sex_tips: { icon: 'heart', tone: 'bloom' },
  fertility: { icon: 'target', tone: 'data' },
};

/**
 * Phase sheet (B-N1-08, `nbl_/nbd_Cycle_Phase`): the user's CURRENT phase with
 * its cycle-day range, five topic tabs (body / mood / nutrition / movement /
 * relationship) and the admin-edited phase-content copy for each. The phase is
 * read from `cycle_view` (never from the URL — §11: a sheet's argument rides in
 * the query string). No confident phase, a 404 or an empty row degrade to a
 * calm fallback. The disclaimer keeps the copy informational, not advice.
 */
export function PhaseDetailsSheet() {
  const t = useTranslations('phaseDetails');
  const locale = useLocale() as Locale;
  const tw = useTranslations('plus.limits');
  const detailLocked = usePlusDetailLocked();
  const mounted = useMounted();

  const { data: today } = useCycleToday();
  const view = today?.cycleView ?? null;
  const subphase = view?.subphase ?? null;
  const query = usePhaseContent(subphase, locale);
  const content = query.data;

  const tabs = content ? availableTabs(content.sections) : [];
  const [picked, setPicked] = useState<PhaseTab | null>(null);
  const tab = picked && tabs.includes(picked) ? picked : (tabs[0] ?? null);

  if (!mounted || query.isLoading) {
    return (
      <SkeletonGroup label={t('loading')} className="phs-skel">
        <Skeleton width="short" />
        <Skeleton shape="block" className="phs-skel-title" />
        <Skeleton shape="block" className="phs-skel-tabs" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  }

  if (subphase === null || query.isError || !content || tab === null) {
    return <EmptyState icon="sparkle" title={t('title')} body={t('fallback')} className="phs-empty" />;
  }

  const range = phaseDayRange(view);
  const main = view?.mainPhase;
  const title = main && main !== 'unknown' ? t(`phases.${main}`) : content.phaseLabel || t('title');
  const why = whyThisDate(view);
  const sections = TAB_SECTIONS[tab].filter((k) => (content.sections[k] ?? '').trim() !== '');

  return (
    <div className="phs">
      <header className="phs-hero">
        {range ? <p className="phs-over">{t('dayRange', range)}</p> : null}
        <h3 className="phs-title">{title}</h3>
      </header>

      <div className="phs-tabs" role="tablist" aria-label={t('tabsLabel')}>
        {tabs.map((key) => (
          <button
            key={key}
            type="button"
            role="tab"
            id={`phs-tab-${key}`}
            aria-selected={key === tab}
            aria-controls="phs-panel"
            className="phs-tab"
            onClick={() => setPicked(key)}
            {...pilot(`phase-details.tab:${key}`)}
          >
            {t(`tabs.${key}`)}
          </button>
        ))}
      </div>

      <div id="phs-panel" role="tabpanel" aria-labelledby={`phs-tab-${tab}`} className="phs-items">
        {sections.map((key) => (
          <section key={key} className="phs-item">
            <IconCircle icon={SECTION_LOOK[key].icon} tone={SECTION_LOOK[key].tone} size="lg" />
            <div className="phs-item-b">
              <h4 className="phs-item-t">{t(`items.${key}`)}</h4>
              <p className="phs-item-s">{content.sections[key]}</p>
            </div>
          </section>
        ))}
      </div>

      {why ? (
        detailLocked ? (
          <PlusDetailLock text={tw('detail.why')} />
        ) : (
          <section className="phs-item" aria-label={tw('why.title')}>
            <div className="phs-item-b">
              <h4 className="phs-item-t">{tw('why.title')}</h4>
              <p className="phs-item-s">{tw('why.source')}: {tw(`why.${why.source}`)}</p>
              <p className="phs-item-s">
                {why.basisCycles ? tw('why.basis', { n: why.basisCycles, count: why.basisCycles }) : tw('why.noBasis')}
              </p>
              {why.reasonCount > 0 ? <p className="phs-item-s">{tw('why.reasons', { n: why.reasonCount })}</p> : null}
            </div>
          </section>
        )
      ) : null}

      <InfoNote className="phs-note">{t('disclaimer')}</InfoNote>
      <PrimaryButton onClick={() => closeSheet()}>{t('gotIt')}</PrimaryButton>
    </div>
  );
}
