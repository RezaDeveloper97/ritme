'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';

import type { LogCategory, LogDayValues, LogOption, LogParamValue } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { ChipGroup, DropSolid, InfoNote, PillChip } from '@/shared/ui';

import { asBool, asString, pickableOptions } from '../../model/draft';
import { ParamField } from '../ParamField';
import { PanelCard } from './PanelCard';

interface BleedingPanelProps {
  category: LogCategory;
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  extraOptions: (category: string, param: string) => LogOption[];
  mode: string | null;
  locale: Locale;
}

/** Swatch classes for the bleeding colours the design paints (Log_Bleeding «رنگ»). */
const SWATCH = new Set(['pink', 'bright_red', 'red', 'dark_red', 'brown', 'black']);

/**
 * Log_Bleeding: flow as five drop cards (لکه‌بینی + the flows), colour swatches, clots and odour, and the
 * «با پزشک مشورت کن» note. Params the mode adds (lochia after birth) render generically underneath.
 */
export function BleedingPanel({ category, values, setParam, extraOptions, mode, locale }: BleedingPanelProps) {
  const t = useTranslations('logSheet');
  const cat = category.code;
  const v = values[cat] ?? {};
  const param = (code: string) => category.params.find((p) => p.code === code);
  const flow = param('flow');
  const spotting = param('spotting');
  const color = param('color');
  const clots = param('clots');
  const clotSize = param('clot_size');
  const odor = param('odor');
  const handled = new Set(['flow', 'spotting', 'color', 'clots', 'clot_size', 'odor']);
  const rest = category.params.filter((p) => !handled.has(p.code));

  const spottingOn = asBool(v.spotting) === true;
  const flowCode = asString(v.flow);
  const flowOptions = flow ? pickableOptions(flow, mode) : [];

  const pickSpotting = () => {
    setParam(cat, 'spotting', spottingOn ? null : true);
    if (!spottingOn) setParam(cat, 'flow', null);
  };
  const pickFlow = (code: string) => {
    const next = flowCode === code ? null : code;
    setParam(cat, 'flow', next);
    if (next && spotting) setParam(cat, 'spotting', null);
  };

  const clotState = asBool(v.clots) === false ? 'none' : asString(v.clot_size);
  const pickClots = (state: 'none' | 'small' | 'large') => {
    if (clotState === state) {
      setParam(cat, 'clots', null);
      setParam(cat, 'clot_size', null);
      return;
    }
    setParam(cat, 'clots', state !== 'none');
    setParam(cat, 'clot_size', state === 'none' ? null : state);
  };

  return (
    <>
      {flow ? (
        <PanelCard title={t('bleeding.intensity')} icon="drop" tone="period">
          <div className="lday-flow" role="group" aria-label={t('bleeding.intensity')}>
            {spotting ? (
              <button type="button" className="lday-flow-card" aria-pressed={spottingOn} onClick={pickSpotting}>
                <span className="lday-flow-drop is-s0" aria-hidden>
                  <DropSolid size={14} color="currentColor" />
                </span>
                <span className="lday-flow-label">{t('bleeding.spotting')}</span>
              </button>
            ) : null}
            {flowOptions.map((o, i) => (
              <button
                key={o.value}
                type="button"
                className="lday-flow-card"
                aria-pressed={!spottingOn && flowCode === o.value}
                onClick={() => pickFlow(o.value)}
              >
                <span className={clsx('lday-flow-drop', `is-s${Math.min(i + 1, 4)}`)} aria-hidden>
                  <DropSolid size={14 + Math.min(i + 1, 4) * 4} color="currentColor" />
                </span>
                <span className="lday-flow-label">{o.label}</span>
              </button>
            ))}
          </div>
          <p className="lday-pcard-hint">{t('bleeding.hint')}</p>
        </PanelCard>
      ) : null}

      {color ? (
        <PanelCard title={color.label} icon="drop" tone="period">
          <div className="lday-swatches" role="group" aria-label={color.label}>
            {pickableOptions(color, mode).map((o) => {
              const on = asString(v.color) === o.value;
              return (
                <button
                  key={o.value}
                  type="button"
                  className="lday-swatch"
                  aria-pressed={on}
                  onClick={() => setParam(cat, 'color', on ? null : o.value)}
                >
                  <span className={clsx('lday-swatch-dot', SWATCH.has(o.value) && `is-${o.value}`)} aria-hidden />
                  <span className="lday-swatch-label">{o.label}</span>
                </button>
              );
            })}
          </div>
        </PanelCard>
      ) : null}

      {clots ? (
        <PanelCard title={clots.label} icon="drop" tone="period">
          <ChipGroup label={clots.label} layout="fill">
            <PillChip shape="square" mode="multi" tone="period" pressed={clotState === 'none'} onPressedChange={() => pickClots('none')}>
              {t('bleeding.clotsNone')}
            </PillChip>
            {clotSize ? (
              <>
                <PillChip shape="square" mode="multi" tone="period" pressed={clotState === 'small'} onPressedChange={() => pickClots('small')}>
                  {t('bleeding.clotsSmall')}
                </PillChip>
                <PillChip shape="square" mode="multi" tone="period" pressed={clotState === 'large'} onPressedChange={() => pickClots('large')}>
                  {t('bleeding.clotsLarge')}
                </PillChip>
              </>
            ) : null}
          </ChipGroup>
        </PanelCard>
      ) : null}

      {odor ? (
        <PanelCard title={odor.label} icon="drop" tone="period">
          <ChipGroup label={odor.label} layout="fill">
            {pickableOptions(odor, mode).map((o) => {
              const on = asString(v.odor) === o.value;
              return (
                <PillChip key={o.value} shape="square" mode="multi" tone="period" pressed={on} onPressedChange={() => setParam(cat, 'odor', on ? null : o.value)}>
                  {o.label}
                </PillChip>
              );
            })}
          </ChipGroup>
        </PanelCard>
      ) : null}

      {rest.length ? (
        <section className="nb-card lday-pcard">
          {rest.map((p) => (
            <ParamField
              key={p.code}
              param={p}
              value={v[p.code]}
              onChange={(next) => setParam(cat, p.code, next)}
              mode={mode}
              tone="period"
              locale={locale}
              extraOptions={extraOptions(cat, p.code)}
            />
          ))}
        </section>
      ) : null}

      <InfoNote icon="info" className="lday-warn">
        {t('bleeding.warn')}
      </InfoNote>
    </>
  );
}
