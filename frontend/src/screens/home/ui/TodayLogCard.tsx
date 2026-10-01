'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useHealthLog } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import { Card, Icon, IconCircle, type IconName, type Tone } from '@/shared/ui';

import { summarizeTodayLog } from '../model/today-log';

type Translate = (key: string, values?: Record<string, string | number>) => string;

interface Row {
  key: string;
  icon: IconName;
  tone: Tone;
  title: string;
  sub: string;
  /** Items logged (shown as a count); 0 = empty (a + disc), null = logged without a count (a tick). */
  count: number | null;
}

/**
 * «ثبت امروز» (nbl_Cycle_Home): today's bleeding, symptoms and mood at a glance
 * with the logging streak. Every row opens the log sheet (`?sheet=log`, B-N3-03);
 * the rows are buttons, so the + disc is decoration, not a second target.
 */
export function TodayLogCard({ streakDays }: { streakDays: number | null }) {
  const t = useTranslations('home.nb.log');
  const tLog = useTranslations('log') as unknown as Translate;
  const loc = useLocale() as Locale;
  const { data: log, isPending } = useHealthLog(toApiDate(today()));
  const summary = summarizeTodayLog(log);

  const label = (key: string) => tLog(key);
  const bleedingSub =
    summary.bleeding === null
      ? t('bleedingHint')
      : summary.bleeding === 'spotting'
        ? label('fields.spotting')
        : `${label('fields.bleeding_intensity')}: ${label(`enums.bleeding_intensity.${summary.bleeding}`)}`;
  const list = (items: string[]) => items.slice(0, 2).join(' · ');

  const rows: Row[] = [
    {
      key: 'bleeding',
      icon: 'drop',
      tone: 'period',
      title: t('bleeding'),
      sub: bleedingSub,
      count: summary.bleeding === null ? 0 : null,
    },
    {
      key: 'symptoms',
      icon: 'symptom',
      tone: 'brand',
      title: t('symptoms'),
      sub: summary.symptoms.length ? list(summary.symptoms.map((k) => label(`fields.${k}`))) : t('empty'),
      count: summary.symptoms.length,
    },
    {
      key: 'mood',
      icon: 'smile',
      tone: 'data',
      title: t('mood'),
      sub: summary.moods.length ? list(summary.moods.map((m) => label(`enums.moods.${m}`))) : t('empty'),
      count: summary.moods.length,
    },
  ];

  return (
    <Card as="section" className="ch-log" aria-busy={isPending || undefined}>
      <div className="ch-card-head">
        <h2 className="ch-card-title">{t('title')}</h2>
        {streakDays != null && streakDays > 1 && (
          <span className="ch-streak">{t('streak', { n: formatNumber(streakDays, loc) })}</span>
        )}
      </div>
      {rows.map((row) => (
        <button
          key={row.key}
          type="button"
          onClick={() => openSheet('log')}
          className="ch-log-row"
          aria-label={row.count === 0 ? t('add', { name: row.title }) : `${row.title} · ${row.sub}`}
        >
          <IconCircle icon={row.icon} tone={row.tone} outlined />
          <span className="ch-log-body">
            <span className="ch-log-title">{row.title}</span>
            <span className="ch-log-sub">{row.sub}</span>
          </span>
          {row.count === 0 ? (
            <span className={`ch-log-add is-${row.tone}`} aria-hidden>
              <Icon name="plus" size={16} strokeWidth={2.4} />
            </span>
          ) : row.count === null ? (
            <span className={`ch-log-count is-${row.tone}`} aria-hidden>
              <Icon name="check" size={16} strokeWidth={2.4} />
            </span>
          ) : (
            <span className={`ch-log-count is-${row.tone}`} aria-hidden>
              {formatNumber(row.count, loc)}
            </span>
          )}
        </button>
      ))}
    </Card>
  );
}
