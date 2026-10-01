'use client';

import { useTranslations } from 'next-intl';
import { forwardRef, useState, type ReactNode } from 'react';

import type { LogCategory, LogDayValues, LogOption, LogParam, LogParamValue } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { Accordion, Icon } from '@/shared/ui';

import { asBool, asItems, categoryEntries, type LabelContext } from '../model/draft';
import { categoryLook, panelOf, type PanelKind } from '../model/presentation';
import { ParamField } from './ParamField';

export interface CategorySectionProps {
  category: LogCategory;
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  extraOptions: (category: string, param: string) => LogOption[];
  mode: string | null;
  labels: LabelContext;
  locale: Locale;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onOpenPanel: (kind: PanelKind) => void;
}

/**
 * One accordion section of «همه موارد», rendered from the taxonomy. Pain gets «بدون درد» as the first
 * location chip, bleeding gets «لکه‌بینی» before the flows; optional `detail` params sit behind
 * «جزئیات بیشتر» — or, for bleeding / pain / measurements, in their detail panel.
 */
export const CategorySection = forwardRef<HTMLDivElement, CategorySectionProps>(function CategorySection(props, ref) {
  const t = useTranslations('logSheet');
  const { category, values, open, onOpenChange, labels } = props;
  const look = categoryLook(category.code);
  const entries = categoryEntries(category, values, labels);
  const summary = entries.length ? entries.map((e) => e.summary).join(t('separator')) : t('notLogged');
  return (
    <div ref={ref} id={`lday-cat-${category.code}`} className="lday-cat">
      <Accordion
        title={category.label}
        summary={summary}
        icon={look.icon}
        tone={look.tone}
        active={entries.length > 0}
        open={open}
        onOpenChange={onOpenChange}
        className="lday-acc"
      >
        {open ? <SectionBody {...props} /> : null}
      </Accordion>
    </div>
  );
});

function SectionBody(props: CategorySectionProps) {
  const t = useTranslations('logSheet');
  const { category, values, setParam, mode, locale, extraOptions, onOpenPanel } = props;
  const [details, setDetails] = useState(false);
  const look = categoryLook(category.code);
  const panel = panelOf(category.code);
  const cat = category.code;
  const val = (p: LogParam) => values[cat]?.[p.code];
  const field = (p: LogParam, extra?: Partial<Parameters<typeof ParamField>[0]>) => (
    <ParamField
      key={p.code}
      param={p}
      value={val(p)}
      onChange={(v) => setParam(cat, p.code, v)}
      mode={mode}
      tone={look.tone}
      locale={locale}
      extraOptions={extraOptions(cat, p.code)}
      hideLabel={category.params.length === 1}
      emptyText={cat === 'meds' && p.code === 'taken' ? t('meds.empty') : undefined}
      {...extra}
    />
  );

  const byCode = (code: string) => category.params.find((p) => p.code === code);
  const handled = new Set<string>();
  const nodes: ReactNode[] = [];

  if (cat === 'bleeding') {
    const flow = byCode('flow');
    const spotting = byCode('spotting');
    if (flow) {
      handled.add('flow');
      if (spotting) handled.add('spotting');
      const spottingOn = spotting ? asBool(val(spotting)) === true : false;
      nodes.push(
        field(flow, {
          hideLabel: true,
          value: spottingOn ? undefined : val(flow),
          onChange: (v) => {
            setParam(cat, 'flow', v);
            if (v !== null && spotting) setParam(cat, 'spotting', null);
          },
          leading: spotting
            ? {
                label: t('bleeding.spotting'),
                pressed: spottingOn,
                onToggle: () => {
                  setParam(cat, 'spotting', spottingOn ? null : true);
                  if (!spottingOn) setParam(cat, 'flow', null);
                },
              }
            : undefined,
        }),
      );
    }
  }

  if (cat === 'pain') {
    const none = byCode('none');
    const location = byCode('location');
    if (location) {
      handled.add('location');
      if (none) handled.add('none');
      const noneOn = none ? asBool(val(none)) === true : false;
      nodes.push(
        field(location, {
          hideLabel: true,
          onChange: (v) => {
            setParam(cat, 'location', v);
            if (v !== null && Object.keys(asItems(v)).length && none) setParam(cat, 'none', null);
          },
          leading: none
            ? {
                label: none.label,
                pressed: noneOn,
                onToggle: () => {
                  setParam(cat, 'none', noneOn ? null : true);
                  if (!noneOn) setParam(cat, 'location', null);
                },
              }
            : undefined,
        }),
      );
    }
    // Relief lives in the pain panel next to the map.
    if (byCode('relief')) handled.add('relief');
  }

  const inline = category.params.filter((p) => !handled.has(p.code) && !p.detail);
  const detail = panel ? [] : category.params.filter((p) => !handled.has(p.code) && p.detail);
  for (const p of inline) nodes.push(field(p));

  const panelHasMore = panel === 'pain' || panel === 'measure' || category.params.some((p) => p.detail);
  const linkLabel =
    panel === 'pain' ? t('bodyMapLink') : panel === 'bleeding' ? t('bleedingLink') : panel === 'measure' ? t('measureLink') : null;

  return (
    <div className="lday-body">
      {nodes}
      {detail.length ? (
        <>
          <button type="button" className="lday-more" aria-expanded={details} onClick={() => setDetails(!details)}>
            {details ? t('less') : t('more')}
            <Icon name="chevronDown" size={16} className="lday-more-chev" />
          </button>
          {details ? detail.map((p) => field(p)) : null}
        </>
      ) : null}
      {panel && panelHasMore && linkLabel ? (
        <button type="button" className="lday-panel-link" onClick={() => onOpenPanel(panel)}>
          {linkLabel}
        </button>
      ) : null}
    </div>
  );
}
