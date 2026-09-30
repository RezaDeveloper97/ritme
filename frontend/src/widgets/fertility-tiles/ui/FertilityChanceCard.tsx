'use client';

import { useTranslations } from 'next-intl';

import type { CycleFertilityLevel } from '@/entities/cycle';
import { Icon } from '@/shared/ui';

import { chanceFraction } from '../model/ttc';

/** Donut geometry (`v19_Main`: 76 px, 9 px stroke). */
const R = 33.5;
const C = 2 * Math.PI * R;

export interface FertilityChanceCardProps {
  /** «شانس بارداری امروز», or the selected day's variant. */
  title: string;
  /** v1.1 top-level `fertility_level` (§26) — the same value the fertility screens use. */
  level: CycleFertilityLevel | null;
  /** Localized level word (`fertility.chance.levels.*`). */
  levelLabel: string;
  description: string;
  /** Pre-formatted dates (§7); null → «—». */
  windowStart: string | null;
  ovulation: string | null;
  nextPeriod: string | null;
  loading?: boolean;
}

/**
 * «شانس بارداری امروز» (`v19_Main` / `nb2_Main`): the day's chance of pregnancy
 * as a donut + level, a line of context, and the three dates a TTC user plans
 * around. Informational only (§11) — it is an estimate, and says so.
 */
export function FertilityChanceCard({
  title,
  level,
  levelLabel,
  description,
  windowStart,
  ovulation,
  nextPeriod,
  loading = false,
}: FertilityChanceCardProps) {
  const t = useTranslations('fertility.home.chanceCard');
  const dash = '—';
  const filled = chanceFraction(level) * C;
  const facts = [
    { key: 'windowStart', label: t('windowStart'), value: windowStart, tone: 'is-warm' },
    { key: 'ovulation', label: t('ovulation'), value: ovulation, tone: 'is-data' },
    { key: 'nextPeriod', label: t('nextPeriod'), value: nextPeriod, tone: 'is-period' },
  ];

  return (
    <section className="nb-card ttc-chancecard" data-loading={loading} aria-busy={loading || undefined}>
      <div className="ttc-row-between">
        <h2 className="ttc-card-title">{title}</h2>
        <span className="ttc-muted-cap">{t('estimated')}</span>
      </div>
      <div className="ttc-chancecard-main">
        <span className="ttc-donut" aria-hidden>
          <svg viewBox="0 0 76 76" className="ttc-donut-svg">
            <circle cx="38" cy="38" r={R} fill="none" strokeWidth="9" className="ttc-donut-track" />
            {filled > 0 && (
              <circle
                cx="38"
                cy="38"
                r={R}
                fill="none"
                strokeWidth="9"
                strokeLinecap="round"
                strokeDasharray={`${filled.toFixed(1)} ${C.toFixed(1)}`}
                className="ttc-donut-fill"
              />
            )}
          </svg>
          <Icon name="target" size={26} className="ttc-donut-icon" />
        </span>
        <div className="ttc-chancecard-text">
          <span className="ttc-chancecard-level">{levelLabel}</span>
          <p className="ttc-chancecard-desc">{description}</p>
        </div>
      </div>
      <dl className="ttc-facts">
        {facts.map((f) => (
          <div key={f.key} className="ttc-fact">
            <dt className={`ttc-fact-label ${f.tone}`}>{f.label}</dt>
            <dd className="ttc-fact-value">{f.value ?? dash}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
