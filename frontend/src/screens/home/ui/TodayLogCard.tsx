'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useMemo } from 'react';

import { useLogDay, useLogPreferences, useLogTaxonomy } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import { Card, Icon, IconCircle, type IconName, type Tone } from '@/shared/ui';

import { summarizeTodayLog } from '../model/today-log';

interface Row {
  /** Also the log taxonomy category the row opens (`?sheet=log` scrolls to it). */
  key: 'bleeding' | 'symptoms' | 'mood';
  icon: IconName;
  tone: Tone;
  title: string;
  sub: string;
  /** Items logged (shown as a count); 0 = empty (a + disc), null = logged without a count (a tick). */
  count: number | null;
}

/**
 * «ثبت امروز» (nbl_Cycle_Home): today's bleeding, symptoms and mood at a glance
 * with the logging streak. Every row opens the log sheet (`?sheet=log`, B-N3-03) on its own section;
 * the rows are buttons, so the + disc is decoration, not a second target.
 */
export function TodayLogCard({ streakDays }: { streakDays: number | null }) {
  const t = useTranslations('home.nb.log');
  const tSheet = useTranslations('logSheet');
  const loc = useLocale() as Locale;
  const { data: day, isPending } = useLogDay(toApiDate(today()));
  const taxonomy = useLogTaxonomy();
  const prefs = useLogPreferences();
  const unknown = tSheet('unknownValue');
  const summary = useMemo(() => {
    const extraLabels: Record<string, string> = {};
    for (const item of prefs.data?.customItems ?? []) extraLabels[item.code] = item.label;
    return summarizeTodayLog(day?.categories, taxonomy.data?.categories ?? [], { extraLabels, unknown });
  }, [day, taxonomy.data, prefs.data, unknown]);

  // Every logged item; the sub line ellipsizes when it runs out of room.
  const list = (items: string[]) => items.join(' · ');

  const rows: Row[] = [
    {
      key: 'bleeding',
      icon: 'drop',
      tone: 'period',
      title: t('bleeding'),
      sub: summary.bleeding ?? t('bleedingHint'),
      count: summary.bleeding === null ? 0 : null,
    },
    {
      key: 'symptoms',
      icon: 'symptom',
      tone: 'brand',
      title: t('symptoms'),
      sub: summary.symptoms.length ? list(summary.symptoms) : t('empty'),
      count: summary.symptoms.length,
    },
    {
      key: 'mood',
      icon: 'smile',
      tone: 'data',
      title: t('mood'),
      sub: summary.moods.length ? list(summary.moods) : t('empty'),
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
          onClick={() => openSheet('log', `${toApiDate(today())}:${row.key}`)}
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
