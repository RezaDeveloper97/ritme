'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useCycleToday } from '@/entities/cycle';
import {
  PHASE_SECTION_KEYS,
  usePhaseContent,
  type PhaseSectionKey,
} from '@/entities/phase-content';
import { useRouter } from '@/shared/i18n';
import type { Locale } from '@/shared/i18n';
import { Icon, NavBack, type IconName } from '@/shared/ui';

/**
 * Icon + accent per section so each topic reads at a glance. Tones are modifier
 * classes over tokens (with dark overrides) — no colors cross as inline styles.
 */
const SECTION_STYLE: Record<PhaseSectionKey, { icon: IconName; tone: string }> = {
  symptom_prediction: { icon: 'chart', tone: 'is-blue' },
  vaginal_discharge: { icon: 'drop', tone: 'is-teal' },
  fertility: { icon: 'heart', tone: 'is-pink' },
  hormonal_changes: { icon: 'zap', tone: 'is-amber' },
  sex_tips: { icon: 'smile', tone: 'is-rose' },
  nutrition: { icon: 'apple', tone: 'is-green' },
  exercise: { icon: 'walk', tone: 'is-orange' },
  skin_care: { icon: 'sparkle', tone: 'is-violet' },
  sleep: { icon: 'moon', tone: 'is-indigo' },
};

/**
 * Full-screen educational detail for the user's CURRENT cycle sub-phase, opened
 * from the daily cycle card. Reads the phase from cycle_view.subphase (never
 * from the URL — §11: cycle phase is health data) and renders the DB-driven
 * sections for it. Every failure mode degrades to a calm fallback rather than a
 * crash: no confident phase, no content row (404), or a phase with no sections.
 */
export function PhaseDetailsPage() {
  const t = useTranslations('phaseDetails');
  const locale = useLocale() as Locale;
  const router = useRouter();

  // Query data is absent during SSR, so gate the data/fallback branches on mount
  // to keep the first client render identical to the server HTML (hydration).
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  const { data: today } = useCycleToday();
  const subphase = today?.cycleView?.subphase ?? null;

  const query = usePhaseContent(subphase, locale);
  const content = query.data;

  // Render sections in the canonical order, keeping only those the API returned
  // with copy — a missing section simply doesn't appear (task requirement).
  const sections = content
    ? PHASE_SECTION_KEYS.filter((key) => {
        const value = content.sections[key];
        return typeof value === 'string' && value.trim() !== '';
      })
    : [];

  const loading = !mounted || query.isLoading;
  const showFallback =
    !loading && (subphase === null || query.isError || sections.length === 0);

  return (
    <div className="view pd-page">
      {/* Same soft pink→teal backdrop the home and analysis screens open on. */}
      <div className="home-grad pd-grad" />

      <div className="hdr pd-hdr">
        <NavBack onClick={() => router.back()} />
        <span className="pd-title">{t('title')}</span>
        {/* Balances the back button so the title sits truly centered. */}
        <span className="pd-hdr-spacer" aria-hidden />
      </div>

      <div className="scroll pd-scroll">
        {loading ? (
          <PhaseDetailsSkeleton />
        ) : showFallback ? (
          <div className="pd-empty">
            <span className="pd-empty-icon">
              <Icon name="sparkle" size={30} stroke="currentColor" />
            </span>
            <p className="pd-empty-text">{t('fallback')}</p>
          </div>
        ) : (
          <>
            {/* Hero — names the phase the whole page is about. */}
            <div className="pd-hero">
              <span className="pd-hero-deco pd-hero-deco-a" />
              <span className="pd-hero-deco pd-hero-deco-b" />
              <div className="pd-hero-row">
                <div className="pd-hero-b">
                  <div className="pd-hero-overline">{t('currentPhase')}</div>
                  <div className="pd-hero-name">
                    {content?.phaseLabel || t('title')}
                  </div>
                </div>
                <span className="pd-hero-badge">
                  <Icon name="sparkle" size={22} stroke="var(--on-accent)" />
                </span>
              </div>
              <p className="pd-hero-sub">{t('subtitle')}</p>
            </div>

            {sections.map((key) => (
              <section key={key} className="card pd-section">
                <div className="pd-sec-head">
                  <span className={`pd-badge ${SECTION_STYLE[key].tone}`}>
                    <Icon
                      name={SECTION_STYLE[key].icon}
                      size={18}
                      stroke="currentColor"
                    />
                  </span>
                  <h2 className="pd-h2">{t(`sections.${key}`)}</h2>
                </div>
                <p className="pd-body">{content?.sections[key]}</p>
              </section>
            ))}
          </>
        )}
        <div className="page-tail" />
      </div>
    </div>
  );
}

/** Placeholder with the loaded page's rhythm — hero block, then section cards. */
function PhaseDetailsSkeleton() {
  return (
    <div aria-hidden className="pd-skel">
      <span className="skeleton-line pd-skel-hero" />
      <span className="skeleton-line pd-skel-card" />
      <span className="skeleton-line pd-skel-card" />
      <span className="skeleton-line pd-skel-card is-short" />
    </div>
  );
}
