'use client';

import clsx from 'clsx';

import { useFormatter, useLocale, useTranslations } from 'next-intl';

import {
  LOG_CATEGORIES,
  useHealthLog,
  type CategoryDef,
  type FieldDef,
  type HealthLogInput,
} from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatJalaliDayMonth, toApiDate } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

type TCal = ReturnType<typeof useTranslations>;
type TLog = ReturnType<typeof useTranslations>;
type Format = ReturnType<typeof useFormatter>;

// `log.enums.*` keys are data-driven (mirror the API enums), so the translator's
// literal-key typing can't see them — same escape hatch the log FieldRow uses.
type DynT = (key: string) => string;
const enumLabel = (t: TLog, enumKey: string, value: string): string =>
  (t as unknown as DynT)(`enums.${enumKey}.${value}`);

// Presentation-only accent per category (mirrors the log screen palette).
const CATEGORY_ACCENT: Record<string, string> = {
  bleeding: 'var(--brand)',
  pain: 'var(--amber)',
  digestion: 'var(--green)',
  mood: 'var(--violet)',
  sleep: 'var(--indigo-deep)',
  body: 'var(--orange)',
  discharge: 'var(--blue)',
  intimate: 'var(--teal)',
  sexual: 'var(--rose)',
  measure: 'var(--muted)',
  notes: 'var(--muted)',
};

interface FieldLine {
  label: string;
  value: string;
}

/**
 * Turn one recorded field into a display line, or `null` when it holds no
 * meaningful value (empty, false, or blank). Read-only — never mutates the log.
 */
function fieldLine(
  field: FieldDef,
  raw: unknown,
  tLog: TLog,
  format: Format,
  sep: string,
): FieldLine | null {
  if (raw === undefined || raw === null) return null;
  const label = tLog(`fields.${field.key}`);
  const control = field.control;

  switch (control.kind) {
    case 'bool':
      return raw === true ? { label, value: tLog('yes') } : null;
    case 'chips':
    case 'degree':
      return { label, value: enumLabel(tLog, control.enumKey, String(raw)) };
    case 'multi': {
      const arr = Array.isArray(raw) ? (raw as string[]) : [];
      if (arr.length === 0) return null;
      return { label, value: arr.map((v) => enumLabel(tLog, control.enumKey, v)).join(sep) };
    }
    case 'measure':
      return { label, value: `${format.number(raw as number)} ${tLog(`units.${control.unit}`)}` };
    case 'note': {
      const text = String(raw).trim();
      return text ? { label, value: text } : null;
    }
    default:
      return null;
  }
}

/** All recorded lines for a category, grouped so empty categories drop out. */
function categoryLines(
  category: CategoryDef,
  log: HealthLogInput,
  tLog: TLog,
  format: Format,
  sep: string,
): FieldLine[] {
  return category.fields
    .map((field) => fieldLine(field, log[field.key], tLog, format, sep))
    .filter((line): line is FieldLine => line !== null);
}

interface DayLogSummaryProps {
  tCal: TCal;
  selectedDate: Date;
  onEdit: () => void;
}

/**
 * Read-only recap of what the user logged on the selected day — shown below the
 * calendar. Reuses the `health-log` entity (its query + category config) so this
 * stays a pure viewer; editing hands off to the full log screen via `onEdit`.
 * Health data never leaves this component (CLAUDE.md §11).
 */
export function DayLogSummary({ tCal, selectedDate, onEdit }: DayLogSummaryProps) {
  const tLog = useTranslations('log');
  const locale = useLocale() as Locale;
  const format = useFormatter();
  const isRtl = locale === 'fa';
  const sep = isRtl ? '، ' : ', ';

  // Value labels resolve from the `log` i18n namespace, so we only need the
  // saved log itself — enum option lists aren't required here.
  const logQuery = useHealthLog(toApiDate(selectedDate));
  const log = logQuery.data;

  const groups = log
    ? LOG_CATEGORIES.map((category) => ({
        key: category.key,
        lines: categoryLines(category, log, tLog, format, sep),
      })).filter((g) => g.lines.length > 0)
    : [];

  const hasEntries = groups.length > 0;

  return (
    <div className="card pad-card">
      <div className={clsx('dls-head', hasEntries && 'has-entries')}>
        <div className="dls-head-left">
          <span className="dls-head-icon">
            <Icon name="pencil" size={16} />
          </span>
          <div className="dls-title">
            {tCal('dayLog.title')}
          </div>
        </div>
        {hasEntries && (
          <button
            className="chip dls-edit"
            onClick={onEdit}
          >
            <Icon name="pencil" size={13} />
            {tCal('dayLog.edit')}
          </button>
        )}
      </div>

      {logQuery.isLoading ? (
        <div className="dls-loading">
          {tLog('loading')}
        </div>
      ) : hasEntries ? (
        <div className="dls-groups">
          {groups.map((group) => (
            <div key={group.key}>
              <div
                className="dls-group-name"
                style={{ color: CATEGORY_ACCENT[group.key] ?? 'var(--muted)' }}
              >
                {tLog(`categories.${group.key}`)}
              </div>
              <div className="dls-lines">
                {group.lines.map((line) => (
                  <div
                    key={line.label}
                    className="dls-line"
                  >
                    <span className="dls-line-label">
                      {line.label}
                    </span>
                    <span className="dls-line-value">
                      {line.value}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      ) : (
        // Empty day → gentle prompt to log (Figma "How are you feeling today?").
        <button
          onClick={onEdit}
          className="card dls-empty"
        >
          <div className="dls-empty-body">
            <div className="dls-empty-title">
              {tCal('dayLog.emptyTitle')}
            </div>
            <div className="dls-empty-sub">
              {tCal('dayLog.emptySubtitle', { date: formatJalaliDayMonth(selectedDate, locale) })}
            </div>
          </div>
          <span className="fab dls-empty-dot">
            <Icon name="plus" size={20} className="ic" />
          </span>
        </button>
      )}
    </div>
  );
}
