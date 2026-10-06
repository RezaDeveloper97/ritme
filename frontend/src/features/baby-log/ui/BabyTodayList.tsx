'use client';

import { useLocale, useTranslations } from 'next-intl';

import { Link, type Locale, useDirection } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon, IconCircle, type IconName, type Tone } from '@/shared/ui';

import { parseBabyToday } from '../api/schema';
import { wallTime } from '../model/live';
import { feedingHref } from '../model/links';
import { hoursText } from './SleepCard';

/**
 * The child home «امروز» rows (v16_ChildHome, B-N5-05 card filled by B-N5-07):
 * feeds (count · last), sleep (hours), diapers (count), each a link into the
 * feeding screen's section. `today` is `ChildHome.today` as the API sent it.
 */
/** `readOnly` (a shared child, B-N5-10): empty rows read «هنوز ثبت نشده», not the «ثبت» call to action. */
export function BabyTodayList({ childId, today, readOnly = false }: { childId: number; today: unknown; readOnly?: boolean }) {
  const t = useTranslations('babyLog');
  const loc = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const d = parseBabyToday(today);
  const empty = readOnly ? t('today.none') : t('today.add');

  const feeding = d?.feedingNow
    ? t('today.feedingNow')
    : d && d.feeds.count > 0
      ? d.lastFeed
        ? t('today.feedsLast', { count: formatNumber(d.feeds.count, loc), time: formatNumber(wallTime(d.lastFeed.startedAt), loc) })
        : t('today.feeds', { count: formatNumber(d.feeds.count, loc) })
      : empty;
  const sleep = d?.sleepingNow
    ? t('today.sleepingNow')
    : d && d.sleep.count > 0
      ? t('today.sleepHours', { hours: hoursText(d.sleep.seconds, loc) })
      : empty;
  const diapers = d && d.diapers.count > 0 ? t('today.diaperCount', { count: formatNumber(d.diapers.count, loc) }) : empty;

  const rows: { key: 'feeding' | 'sleep' | 'diapers'; icon: IconName; tone: Tone; value: string; live: boolean }[] = [
    { key: 'feeding', icon: 'bottle', tone: 'data', value: feeding, live: !!d?.feedingNow },
    { key: 'sleep', icon: 'sleep', tone: 'brand', value: sleep, live: !!d?.sleepingNow },
    { key: 'diapers', icon: 'drop', tone: 'warm', value: diapers, live: false },
  ];

  return (
    <ul className="bfl-today-list">
      {rows.map((r) => (
        <li key={r.key}>
          <Link href={feedingHref(childId, r.key)} className="bfl-today-row">
            <IconCircle icon={r.icon} tone={r.tone} size="sm" />
            <span className="bfl-today-label">{t(`today.labels.${r.key}`)}</span>
            <span className={r.live ? 'bfl-today-value is-live' : 'bfl-today-value'}>{r.value}</span>
            <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={16} strokeWidth={2} className="bfl-today-chev" />
          </Link>
        </li>
      ))}
    </ul>
  );
}
