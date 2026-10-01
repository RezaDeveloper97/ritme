'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useState, type ComponentType } from 'react';

import type { LogCategory, LogDayValues, LogParamValue } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { ChipGroup, PillChip } from '@/shared/ui';

import { asItems, asList, defaultLevel, pickableOptions, setItemScore, toggleInList, toggleItem } from '../../model/draft';
import { PanelCard } from './PanelCard';

/** One body-map region as the log sheet hands it to the map widget. */
export interface BodyMapRegionView {
  code: string;
  label: string;
  selected: boolean;
  /** The region the intensity scale below is editing. */
  current: boolean;
}

/**
 * Props of the body map the screen plugs in (`widgets/body-map`): a widget sits above this feature, so
 * the sheet receives it as a component instead of importing it (FSD §3.1).
 */
export interface BodyMapSlotProps {
  regions: BodyMapRegionView[];
  onToggle: (code: string) => void;
  /** Accessible name of the map. */
  label: string;
}

export type BodyMapSlot = ComponentType<BodyMapSlotProps>;

interface PainPanelProps {
  category: LogCategory;
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  mode: string | null;
  locale: Locale;
  BodyMap?: BodyMapSlot;
}

/**
 * Log_Pain: tap where it hurts on the body map (each region a labelled toggle), rate the current region
 * 1–10 (the score also sets its level: 1–3 کم, 4–6 متوسط, 7–10 شدید) and say what helped.
 */
export function PainPanel({ category, values, setParam, mode, locale, BodyMap }: PainPanelProps) {
  const t = useTranslations('logSheet');
  const cat = category.code;
  const location = category.params.find((p) => p.code === 'location');
  const relief = category.params.find((p) => p.code === 'relief');
  const none = category.params.find((p) => p.code === 'none');
  const items = asItems(values[cat]?.location);
  const picked = Object.keys(items).filter((k) => items[k].level !== 'no');
  const [current, setCurrent] = useState<string | null>(picked[picked.length - 1] ?? null);
  const focus = current && picked.includes(current) ? current : (picked[picked.length - 1] ?? null);
  const regions = location ? pickableOptions(location, mode) : [];

  const toggle = (code: string) => {
    if (!location) return;
    const on = picked.includes(code);
    const next = toggleItem(items, code, defaultLevel(location));
    setParam(cat, 'location', next);
    if (!on) {
      setCurrent(code);
      if (none) setParam(cat, 'none', null);
    }
  };

  const regionLabel = (code: string | null) => regions.find((r) => r.value === code)?.label ?? '';
  const score = focus ? items[focus]?.score : null;
  const range = location?.score ?? { min: 1, max: 10 };
  const cells = Array.from({ length: range.max - range.min + 1 }, (_, i) => range.min + i);
  const relieved = asList(values[cat]?.relief);

  return (
    <>
      {location && BodyMap ? (
        <section className="nb-card lday-pcard lday-map-card">
          <BodyMap
            label={t('pain.map')}
            onToggle={toggle}
            regions={regions.map((r) => ({
              code: r.value,
              label: r.label,
              selected: picked.includes(r.value),
              current: focus === r.value,
            }))}
          />
        </section>
      ) : null}

      {location?.score ? (
        <PanelCard title={focus ? t('pain.scale', { region: regionLabel(focus) }) : t('severity')} icon="symptom" tone="bloom">
          {picked.length > 1 ? (
            <ChipGroup label={t('pain.which')}>
              {picked.map((code) => (
                <PillChip key={code} mode="multi" tone="bloom" pressed={focus === code} onPressedChange={() => setCurrent(code)}>
                  {regionLabel(code)}
                </PillChip>
              ))}
            </ChipGroup>
          ) : null}
          {focus ? (
            <div className="lday-pscale">
              <div className="lday-pscale-row" role="group" aria-label={t('pain.scale', { region: regionLabel(focus) })}>
                {cells.map((n) => (
                  <button
                    key={n}
                    type="button"
                    aria-pressed={score === n}
                    className={clsx('lday-pscale-cell', score != null && n <= score && 'is-on')}
                    onClick={() => setParam(cat, 'location', setItemScore(items, focus, n))}
                  >
                    {formatNumber(n, locale)}
                  </button>
                ))}
              </div>
              <div className="lday-pscale-ends" aria-hidden>
                <span>{t('pain.min')}</span>
                <span>{t('pain.max')}</span>
              </div>
            </div>
          ) : (
            <p className="lday-pcard-hint">{t('pain.pick')}</p>
          )}
        </PanelCard>
      ) : null}

      {relief ? (
        <PanelCard title={t('pain.relief')} icon="symptom" tone="bloom">
          <ChipGroup label={t('pain.relief')}>
            {pickableOptions(relief, mode).map((o) => (
              <PillChip
                key={o.value}
                mode="multi"
                tone="data"
                className="lday-chip-lg"
                pressed={relieved.includes(o.value)}
                onPressedChange={() => setParam(cat, 'relief', toggleInList(relieved, o.value))}
              >
                {o.label}
              </PillChip>
            ))}
          </ChipGroup>
        </PanelCard>
      ) : null}
    </>
  );
}
