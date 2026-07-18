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
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span style={{ flex: 1, fontSize: 16, fontWeight: 800, color: 'var(--ink)', textAlign: 'start' }}>
          {title}
        </span>
      </div>

      <div className="scroll" style={{ padding: '4px 18px 24px' }}>
        {!loading && !showFallback && (
          <p style={{ fontSize: 12, color: 'var(--muted)', margin: '6px 2px 0', textAlign: 'start' }}>
            {t('subtitle')}
          </p>
        )}

        {loading ? (
          <div style={{ padding: '48px 0', textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>
            {t('loading')}
          </div>
        ) : showFallback ? (
          <section className="card" style={{ padding: 22, marginTop: 14, textAlign: 'center' }}>
            <p style={{ fontSize: 13.5, lineHeight: 2, color: 'var(--muted)', margin: 0 }}>
              {t('fallback')}
            </p>
          </section>
        ) : (
          sections.map((key) => (
            <section key={key} className="card" style={{ padding: 18, marginTop: 14 }}>
              <h2 style={{ fontSize: 14.5, fontWeight: 800, color: 'var(--ink)', margin: 0, textAlign: 'start' }}>
                {t(`sections.${key}`)}
              </h2>
              <p
                style={{
                  fontSize: 13.5,
                  lineHeight: 2,
                  color: 'var(--muted)',
                  margin: '8px 0 0',
                  textAlign: 'start',
                  whiteSpace: 'pre-line',
                }}
              >
                {content?.sections[key]}
              </p>
            </section>
          ))
        )}
      </div>
    </div>
  );
}
