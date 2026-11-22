'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useMemo } from 'react';

import { useLogDay, useLogPreferences, useLogTaxonomy } from '@/entities/health-log';
import { Link, type Locale } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import { Icon, IconCircle, type IconName, type Tone } from '@/shared/ui';

import { summarizeTodayLog } from '../model/today-log';
import '@/shared/ui/styles/ch.css';

interface Tile {
  key: 'period' | 'symptoms' | 'mood';
  icon: IconName;
  tone: Tone;
  title: string;
  sub: string;
  /** Items logged (shown as a count); 0 = empty (a + disc), null = logged without a count (a tick). */
  count: number | null;
}

/**
 * «ثبت امروز» (nbl_Cycle_Home): three cards — «ثبت پریود», «علائم», «حال و روحیه» — with the logging
 * streak (user request, 2026-11-22). «ثبت پریود» opens the calendar's period editor (`/calendar?editDates=1`);
 * the other two open the log sheet (`?sheet=log`, B-N3-03) on their own section. Each card is one
 * target, so the + disc / count is decoration.
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

  // Every logged item; the sub line clamps when it runs out of room.
  const list = (items: string[]) => items.join(' · ');

  const tiles: Tile[] = [
    {
      key: 'period',
      icon: 'drop',
      tone: 'period',
      title: t('period'),
      sub: summary.bleeding ?? t('periodHint'),
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

  // Centred card (user request, 2026-11-22): the 3D icon on top, then title and summary; the + / count / tick
  // sits in the top corner.
  const body = (tile: Tile) => (
    <>
      {tile.count === 0 ? (
        <span className={`ch-log-add is-${tile.tone}`} aria-hidden>
          <Icon name="plus" size={14} strokeWidth={2.4} />
        </span>
      ) : tile.count === null ? (
        <span className={`ch-log-count is-${tile.tone}`} aria-hidden>
          <Icon name="check" size={16} strokeWidth={2.4} />
        </span>
      ) : (
        <span className={`ch-log-count is-${tile.tone}`} aria-hidden>
          {formatNumber(tile.count, loc)}
        </span>
      )}
      <IconCircle icon={tile.icon} tone={tile.tone} size="lg" outlined className="ch-log-icon" />
      <span className="ch-log-title">{tile.title}</span>
      <span className="ch-log-sub">{tile.sub}</span>
    </>
  );
  // «ثبت پریود» already says what it does; the empty sections read «ثبت علائم» / «ثبت حال و روحیه».
  const label = (tile: Tile) =>
    tile.count !== 0 ? `${tile.title} · ${tile.sub}` : tile.key === 'period' ? tile.title : t('add', { name: tile.title });

  return (
    <section className="ch-log" aria-busy={isPending || undefined}>
      <div className="ch-card-head">
        <h2 className="ch-card-title">{t('title')}</h2>
        {streakDays != null && streakDays > 1 && (
          <span className="ch-streak">{t('streak', { n: formatNumber(streakDays, loc) })}</span>
        )}
      </div>
      <div className="ch-log-grid">
        {tiles.map((tile) =>
          tile.key === 'period' ? (
            <Link
              key={tile.key}
              href="/calendar?editDates=1"
              className="nb-card ch-log-tile"
              {...pilot(`home.log-card:${tile.key}`)}
              aria-label={label(tile)}
            >
              {body(tile)}
            </Link>
          ) : (
            <button
              key={tile.key}
              type="button"
              onClick={() => openSheet('log', `${toApiDate(today())}:${tile.key}`)}
              className="nb-card ch-log-tile"
              {...pilot(`home.log-card:${tile.key}`)}
              aria-label={label(tile)}
            >
              {body(tile)}
            </button>
          ),
        )}
      </div>
    </section>
  );
}
