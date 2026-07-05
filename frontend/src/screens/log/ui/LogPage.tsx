'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import {
  LOG_CATEGORIES,
  useHealthLog,
  useHealthLogEnums,
  useSaveHealthLog,
  type CategoryDef,
  type HealthLogField,
  type HealthLogInput,
} from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { addDays, diffInDays, formatJalaliDayMonth, toApiDate, today } from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { CategorySheet } from './CategorySheet';

// Presentation-only mapping (icon + accent) for each category. Colors are a UI
// concern, so they live here rather than in the entity's model config.
const CATEGORY_STYLE: Record<string, { icon: IconName; color: string; soft: string }> = {
  bleeding: { icon: 'drop', color: '#E91E63', soft: '#FCE7F3' },
  pain: { icon: 'zap', color: '#F5A623', soft: '#FEF3C6' },
  digestion: { icon: 'glass', color: '#22B07D', soft: '#E7F8EF' },
  mood: { icon: 'smile', color: '#A91EE9', soft: '#F5E9FE' },
  sleep: { icon: 'moon', color: '#5B6BE1', soft: '#EAECFF' },
  body: { icon: 'sparkle', color: '#E9662E', soft: '#FDEBE2' },
  discharge: { icon: 'drop', color: '#2E9BE9', soft: '#E3F1FD' },
  intimate: { icon: 'shield', color: '#0E9C8A', soft: '#DFF5F1' },
  sexual: { icon: 'heart', color: '#E9276E', soft: '#FDE4EE' },
  measure: { icon: 'thermo', color: '#6D7A87', soft: '#EDF0F3' },
  notes: { icon: 'pencil', color: '#707983', soft: '#EFF2F4' },
};

type T = ReturnType<typeof useTranslations>;

/** Count of non-empty draft fields in a category → drives the card's summary. */
function filledCount(category: CategoryDef, draft: HealthLogInput): number {
  return category.fields.reduce((n, f) => {
    const v = draft[f.key];
    if (v === undefined || v === null) return n;
    if (Array.isArray(v) && v.length === 0) return n;
    if (typeof v === 'string' && v.trim() === '') return n;
    return n + 1;
  }, 0);
}

// ── Day switcher ───────────────────────────────────────────────
interface DaySwitcherProps {
  t: T;
  locale: Locale;
  date: Date;
  isRtl: boolean;
  onShift: (delta: number) => void;
  canGoNext: boolean;
}

function DaySwitcher({ t, locale, date, isRtl, onShift, canGoNext }: DaySwitcherProps) {
  const isToday = diffInDays(date, today()) === 0;
  const prev = (
    <button className="iconbtn" onClick={() => onShift(-1)} aria-label={t('prevDay')}>
      <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
    </button>
  );
  const next = (
    <button
      className="iconbtn"
      onClick={() => canGoNext && onShift(1)}
      disabled={!canGoNext}
      aria-label={t('nextDay')}
      style={{ opacity: canGoNext ? 1 : 0.3 }}
    >
      <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
    </button>
  );

  return (
    <div className="card" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px' }}>
      {/* First child sits at the inline-start (right in RTL) */}
      {isRtl ? prev : next}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Icon name="calendar" size={16} />
        <span style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)' }}>
          {formatJalaliDayMonth(date, locale)}
        </span>
        {isToday && (
          <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--brand)', background: '#FFF1F7', borderRadius: 20, padding: '3px 10px' }}>
            {t('today')}
          </span>
        )}
      </div>
      {isRtl ? next : prev}
    </div>
  );
}

// ── Category card ──────────────────────────────────────────────
interface CategoryCardProps {
  t: T;
  category: CategoryDef;
  count: number;
  isRtl: boolean;
  onOpen: () => void;
}

function CategoryCard({ t, category, count, isRtl, onOpen }: CategoryCardProps) {
  const style = CATEGORY_STYLE[category.key];
  return (
    <button
      onClick={onOpen}
      className="card"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '13px 14px',
        width: '100%',
        cursor: 'pointer',
        textAlign: 'start',
        fontFamily: 'inherit',
      }}
    >
      <span className="dot" style={{ width: 42, height: 42, background: style.soft, color: style.color }}>
        <Icon name={style.icon} size={20} />
      </span>
      <div style={{ flex: 1, textAlign: 'start' }}>
        <div style={{ fontSize: 14.5, fontWeight: 800, color: 'var(--ink)' }}>
          {t(`categories.${category.key}`)}
        </div>
        <div style={{ fontSize: 12, color: count ? style.color : 'var(--muted)', marginTop: 3, fontWeight: count ? 700 : 500 }}>
          {count ? t('selected', { count }) : t(`categoryHint.${category.key}`)}
        </div>
      </div>
      {/* Disclosure chevron points toward the reading-end (left in RTL). */}
      <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={18} style={{ color: 'var(--muted)' }} />
    </button>
  );
}

// ── Main export ────────────────────────────────────────────────
export function LogPage() {
  const t = useTranslations('log');
  const locale = useLocale() as Locale;
  const isRtl = locale === 'fa';

  const [date, setDate] = useState<Date>(() => today());
  const apiDate = useMemo(() => toApiDate(date), [date]);

  const enumsQuery = useHealthLogEnums();
  const logQuery = useHealthLog(apiDate);
  const save = useSaveHealthLog();

  const [draft, setDraft] = useState<HealthLogInput>(() => ({ log_date: apiDate }));
  const [openKey, setOpenKey] = useState<string | null>(null);

  // Reset to an empty draft the moment the day changes; the prefill effect below
  // repopulates it once that day's saved log settles.
  useEffect(() => {
    setDraft({ log_date: apiDate });
  }, [apiDate]);

  // Adopt the saved log for the day (kept per-date in the query cache).
  useEffect(() => {
    if (logQuery.data) setDraft(logQuery.data);
  }, [logQuery.data]);

  const handleChange = (key: HealthLogField, value: unknown) => {
    setDraft((d) => {
      const next = { ...d };
      if (value === undefined) delete next[key];
      else (next as Record<string, unknown>)[key] = value;
      return next;
    });
    save.reset();
  };

  const shiftDay = (delta: number) => {
    setDate((d) => addDays(d, delta));
    save.reset();
  };
  const canGoNext = diffInDays(date, today()) < 0;

  const openCategory = openKey ? LOG_CATEGORIES.find((c) => c.key === openKey) ?? null : null;
  const totalFilled = LOG_CATEGORIES.reduce((n, c) => n + filledCount(c, draft), 0);

  const handleSave = () => {
    if (totalFilled === 0) return;
    save.mutate({ ...draft, log_date: apiDate });
  };

  const saveLabel = save.isPending ? t('saving') : save.isSuccess ? t('saved') : t('save');

  return (
    <div className="view" style={{ background: 'var(--page)' }}>

      <div className="scroll">
        <div style={{ padding: '6px 20px 0', textAlign: 'start' }}>
          <div className="titr">{t('title')}</div>
          <p className="sub" style={{ margin: '6px 0 0' }}>{t('subtitle')}</p>
        </div>

        <div style={{ padding: '14px 16px 0' }}>
          <DaySwitcher t={t} locale={locale} date={date} isRtl={isRtl} onShift={shiftDay} canGoNext={canGoNext} />
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: '14px 16px 0' }}>
          {LOG_CATEGORIES.map((category) => (
            <CategoryCard
              key={category.key}
              t={t}
              category={category}
              count={filledCount(category, draft)}
              isRtl={isRtl}
              onOpen={() => setOpenKey(category.key)}
            />
          ))}
        </div>

        {save.isError && (
          <p style={{ color: 'var(--brand)', fontSize: 12.5, fontWeight: 700, textAlign: 'center', margin: '14px 16px 0' }}>
            {t('saveError')}
          </p>
        )}

        <div style={{ height: 24 }} />
      </div>

      <div style={{ padding: '10px 16px 4px', background: 'var(--page)' }}>
        <button
          className="btn btn-primary"
          onClick={handleSave}
          disabled={totalFilled === 0 || save.isPending}
          style={{ borderRadius: 16 }}
        >
          {save.isSuccess ? <Icon name="check" size={18} /> : null}
          {saveLabel}
        </button>
      </div>

      <BottomNav />

      {openCategory && (
        <CategorySheet
          category={openCategory}
          enums={enumsQuery.data}
          draft={draft}
          onChange={handleChange}
          onClose={() => setOpenKey(null)}
        />
      )}
    </div>
  );
}
