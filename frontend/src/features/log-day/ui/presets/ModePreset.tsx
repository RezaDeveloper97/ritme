'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import type { LogCategory, LogDayValues, LogParamValue } from '@/entities/health-log';
import { Link, useDirection } from '@/shared/i18n';
import { Icon, IconCircle, SeverityScale, StatusPill, type IconName, type Tone } from '@/shared/ui';

import { categoryEntries, paramEntries, type LabelContext } from '../../model/draft';
import {
  hasItemRow,
  itemRowLevel,
  loggedItemCount,
  presetCards,
  ROW_LEVELS,
  rowAcceptsNo,
  rowAvailable,
  withItemRowLevel,
  type ItemRow,
  type ModePresetKind,
  type PresetRowSpec,
  type RowLevel,
} from '../../model/mode-presets';
import { linkHref, type PanelKind } from '../../model/presentation';

export interface ModePresetProps {
  kind: ModePresetKind;
  categories: readonly LogCategory[];
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  labels: LabelContext;
  /** Opens a detail panel of the sheet (bleeding, pain + body map, measurements). */
  onPanel: (kind: PanelKind) => void;
  /** Opens a category's accordion in «همه موارد» and scrolls to it. */
  onSection: (category: string) => void;
  /**
   * Pregnancy: the v2 pregnancy day log (`/pregnancy/log?tab=day`) — water, mood and symptoms through the
   * pregnancy endpoints, which raise the pregnancy alerts. `null` hides the row.
   */
  dailyHref?: string | null;
}

/**
 * The pregnancy / postpartum preset of the log sheet (B-N3-06, nbl_/nbd_Log_Sheet_Preg and _Post): the
 * board's grouped cards under the quick tiles. Every row edits the sheet's draft (or opens a panel /
 * section of it), so the sheet's footer still saves the day in one PUT.
 */
export function ModePreset({ kind, categories, values, setParam, labels, onPanel, onSection, dailyHref }: ModePresetProps) {
  const t = useTranslations('logSheet.presets');
  const tSheet = useTranslations('logSheet');
  const bleeding = kind === 'pregnancy' ? categories.find((c) => c.code === 'bleeding') : undefined;
  const bleedingLogged = bleeding ? categoryEntries(bleeding, values, labels) : [];
  return (
    <div className="mpre" data-kind={kind}>
      {bleeding ? (
        <button type="button" className="mpre-alert" onClick={() => onPanel('bleeding')}>
          <Icon name="drop" size={20} className="mpre-alert-icon" />
          <span className="mpre-row-text">
            <span className="mpre-row-title">{t('pregnancy.bleeding.title')}</span>
            <span className="mpre-row-desc">
              {bleedingLogged.length ? bleedingLogged.map((e) => e.summary).join(tSheet('separator')) : t('pregnancy.bleeding.note')}
            </span>
          </span>
          <Icon name="plus" size={16} strokeWidth={2.4} className="mpre-alert-plus" />
        </button>
      ) : null}
      {presetCards(kind).map((card) => {
        const rows = card.rows.filter((r) => rowAvailable(categories, r));
        if (!rows.length) return null;
        const headId = `mpre-${kind}-${card.code}`;
        return (
          <section key={card.code} className="mpre-sec" aria-labelledby={headId}>
            <h3 id={headId} className="mpre-sec-title">
              {t(`${kind}.cards.${card.code}` as 'pregnancy.cards.symptoms')}
            </h3>
            <div className="mpre-card">
              {rows.map((spec) => (
                <PresetRow
                  key={spec.code}
                  kind={kind}
                  spec={spec}
                  categories={categories}
                  values={values}
                  setParam={setParam}
                  labels={labels}
                  onPanel={onPanel}
                  onSection={onSection}
                />
              ))}
            </div>
          </section>
        );
      })}
      {kind === 'pregnancy' && dailyHref ? (
        <Link href={dailyHref} className="mpre-daily">
          <IconCircle icon="stetho" tone="data" size="sm" />
          <span className="mpre-row-text">
            <span className="mpre-row-title">{t('pregnancy.daily.title')}</span>
            <span className="mpre-row-desc">{t('pregnancy.daily.desc')}</span>
          </span>
          <Chevron />
        </Link>
      ) : null}
    </div>
  );
}

function Chevron() {
  const rtl = useDirection() === 'rtl';
  return <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="mpre-chev" />;
}

type RowProps = Omit<ModePresetProps, 'dailyHref'> & { spec: PresetRowSpec };

function rowTitle(t: ReturnType<typeof useTranslations<'logSheet.presets'>>, kind: ModePresetKind, code: string) {
  return t(`${kind}.rows.${code}.title` as 'pregnancy.rows.weight.title');
}

function rowDesc(t: ReturnType<typeof useTranslations<'logSheet.presets'>>, kind: ModePresetKind, code: string): string | null {
  const key = `${kind}.rows.${code}.desc` as 'pregnancy.rows.weight.desc';
  return t.has(key) ? t(key) : null;
}

function PresetRow(props: RowProps) {
  const { spec } = props;
  if (spec.kind === 'items') return <ItemsRow {...props} spec={spec} />;
  if (spec.kind === 'link') return <LinkRow {...props} spec={spec} />;
  return <ActionRow {...props} spec={spec} />;
}

function RowHead({
  icon,
  tone,
  title,
  desc,
  active,
}: {
  icon: IconName;
  tone: Tone;
  title: string;
  desc: string | null;
  active: boolean;
}) {
  return (
    <>
      <IconCircle icon={icon} tone={tone} size="sm" />
      <span className="mpre-row-text">
        <span className="mpre-row-title">{title}</span>
        {desc ? <span className={clsx('mpre-row-desc', active && 'is-logged')}>{desc}</span> : null}
      </span>
    </>
  );
}

/** A panel or section row: what is logged (in the tone) or the board's hint, «+» at the end. */
function ActionRow({ kind, spec, categories, values, labels, onPanel, onSection }: RowProps & { spec: Extract<PresetRowSpec, { kind: 'panel' | 'section' }> }) {
  const t = useTranslations('logSheet.presets');
  const tSheet = useTranslations('logSheet');
  const category = categories.find((c) => c.code === spec.category);
  if (!category) return null;
  const params = spec.kind === 'panel' && spec.params ? category.params.filter((p) => spec.params!.includes(p.code)) : category.params;
  const entries = params.flatMap((p) => paramEntries(category.code, p, values[category.code]?.[p.code], labels));
  const desc = entries.length ? entries.map((e) => e.summary).join(tSheet('separator')) : (rowDesc(t, kind, spec.code) ?? tSheet('notLogged'));
  return (
    <button
      type="button"
      className={clsx('mpre-row', entries.length > 0 && 'is-active')}
      onClick={() => (spec.kind === 'panel' ? onPanel(spec.panel) : onSection(spec.category))}
    >
      <RowHead icon={spec.icon} tone={spec.tone} title={rowTitle(t, kind, spec.code)} desc={desc} active={entries.length > 0} />
      <Icon name="plus" size={16} strokeWidth={2.2} className="mpre-plus" />
    </button>
  );
}

/** A group that expands in place into severity rows («ندارم · کم · متوسط · شدید»). */
function ItemsRow({ kind, spec, categories, values, setParam, onSection }: RowProps & { spec: Extract<PresetRowSpec, { kind: 'items' }> }) {
  const t = useTranslations('logSheet.presets');
  const tSheet = useTranslations('logSheet');
  const [open, setOpen] = useState(false);
  const bodyId = useId();
  const rows = spec.rows.filter((r) => hasItemRow(categories, r));
  const optionLabel = (r: ItemRow) =>
    categories
      .find((c) => c.code === r.category)
      ?.params.find((p) => p.code === r.param)
      ?.options.find((o) => o.value === r.item)?.label ?? r.item;
  const section = spec.section ? categories.find((c) => c.code === spec.section) : undefined;
  const count = loggedItemCount(values, rows);
  const names = [...rows.map(optionLabel), ...(section ? [section.label] : [])].join(tSheet('separator'));
  const logged = rows.filter((r) => {
    const level = itemRowLevel(values, r);
    return level !== null && level !== 'no';
  });
  const desc = count ? logged.map(optionLabel).join(tSheet('separator')) : names;
  const levels = ROW_LEVELS.map((value) => ({ value, label: t(`levels.${value}`) }));
  return (
    <div className={clsx('mpre-group', open && 'is-open')}>
      <button
        type="button"
        className={clsx('mpre-row', count > 0 && 'is-active')}
        aria-expanded={open}
        aria-controls={bodyId}
        onClick={() => setOpen(!open)}
      >
        <RowHead icon={spec.icon} tone={spec.tone} title={rowTitle(t, kind, spec.code)} desc={desc} active={count > 0} />
        {count ? <span className="mpre-count">{count}</span> : null}
        <Icon name={open ? 'minus' : 'plus'} size={16} strokeWidth={2.2} className="mpre-plus" />
      </button>
      {open ? (
        <div id={bodyId} className="mpre-group-body">
          {rows.map((r) => {
            const noLevel = rowAcceptsNo(categories, r);
            return (
              <SeverityScale<RowLevel>
                key={`${r.category}.${r.param}.${r.item}`}
                className="mpre-sev"
                label={optionLabel(r)}
                options={levels}
                value={itemRowLevel(values, r)}
                onChange={(level) => setParam(r.category, r.param, withItemRowLevel(values, r, level, noLevel))}
              />
            );
          })}
          {section ? (
            <button type="button" className="mpre-more" onClick={() => onSection(section.code)}>
              {t('openSection', { name: section.label })}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

/** A tile fed by another feature (feeding, baby sleep, diapers): opens it, or «به‌زودی» until it exists. */
function LinkRow({ kind, spec, categories }: RowProps & { spec: Extract<PresetRowSpec, { kind: 'link' }> }) {
  const t = useTranslations('logSheet.presets');
  const tSheet = useTranslations('logSheet');
  const param = categories.find((c) => c.code === spec.category)?.params.find((p) => p.code === spec.param);
  if (!param) return null;
  const href = linkHref(param.source);
  const head = <RowHead icon={spec.icon} tone={spec.tone} title={rowTitle(t, kind, spec.code)} desc={rowDesc(t, kind, spec.code)} active={false} />;
  if (href) {
    return (
      <Link href={href} className="mpre-row">
        {head}
        <Chevron />
      </Link>
    );
  }
  return (
    <div className="mpre-row is-soon">
      {head}
      <StatusPill tone="neutral">{tSheet('link.soon')}</StatusPill>
    </div>
  );
}
