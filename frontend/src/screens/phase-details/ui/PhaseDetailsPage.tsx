'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useCycleToday } from '@/entities/cycle';
import { PHASE_SECTION_KEYS, usePhaseContent } from '@/entities/phase-content';
import { useRouter } from '@/shared/i18n';
import type { Locale } from '@/shared/i18n';
import { NavBack } from '@/shared/ui';

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

  const title = content?.phaseLabel || t('title');
  const loading = !mounted || query.isLoading;
  const showFallback =
    !loading && (subphase === null || query.isError || sections.length === 0);

  return (
    <div className="view pd-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="pd-title">
          {title}
        </span>
      </div>

      <div className="scroll pd-scroll">
        {!loading && !showFallback && (
          <p className="pd-sub">
            {t('subtitle')}
          </p>
        )}

        {loading ? (
          <div className="pd-loading">
            {t('loading')}
          </div>
        ) : showFallback ? (
          <section className="card pd-fallback">
            <p className="pd-fallback-text">
              {t('fallback')}
            </p>
          </section>
        ) : (
          sections.map((key) => (
            <section key={key} className="card pd-section">
              <h2 className="pd-h2">
                {t(`sections.${key}`)}
              </h2>
              <p className="pd-body">
                {content?.sections[key]}
              </p>
            </section>
          ))
        )}
      </div>
    </div>
  );
}
