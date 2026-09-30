'use client';

import { useTranslations } from 'next-intl';

import { useUserMode } from '@/entities/message';
import { useRouter } from '@/shared/i18n';
import { openSheet } from '@/shared/sheet';
import { SkeletonGroup, Skeleton, TileButton, type IconName, type Tone } from '@/shared/ui';

import { resolveNavMode, type NavMode } from '../model/nav-items';

type EntryKey = 'daily' | 'fertility' | 'bbt' | 'pregDaily' | 'pregWeekly' | 'pregMovement' | 'checkup' | 'reminder';

interface Entry {
  key: EntryKey;
  icon: IconName;
  tone: Tone;
  /** Route to open; `reminder` opens the add-reminder sheet instead. */
  href?: string;
}

const REMINDER: Entry = { key: 'reminder', icon: 'alarm', tone: 'warm' };
const CHECKUP: Entry = { key: 'checkup', icon: 'stetho', tone: 'data', href: '/checkups' };

/** What the FAB offers per mode until the real log sheets land (B-N3-03). */
export function logEntries(mode: NavMode): Entry[] {
  switch (mode) {
    case 'pregnancy':
      return [
        { key: 'pregDaily', icon: 'pen', tone: 'brand', href: '/pregnancy/log' },
        { key: 'pregWeekly', icon: 'stetho', tone: 'data', href: '/pregnancy/log?tab=weekly' },
        { key: 'pregMovement', icon: 'heartLine', tone: 'bloom', href: '/pregnancy/log?tab=movement' },
        REMINDER,
      ];
    case 'ttc':
      return [
        { key: 'daily', icon: 'pen', tone: 'brand', href: '/log' },
        { key: 'fertility', icon: 'target', tone: 'data', href: '/fertility/log' },
        { key: 'bbt', icon: 'thermo', tone: 'warm', href: '/fertility/bbt' },
        REMINDER,
      ];
    case 'companion':
    case 'postpartum':
      return [REMINDER, CHECKUP];
    default:
      return [{ key: 'daily', icon: 'pen', tone: 'brand', href: '/log' }, CHECKUP, REMINDER];
  }
}

export function LogSheetTitle() {
  return <>{useTranslations('nav')('logSheet.title')}</>;
}

/**
 * The FAB's target, `?sheet=log` (B-N1-04, gaps.md #17). Interim content: the
 * mode's existing log destinations as quick tiles. B-N3-03 replaces it with the
 * full Log_Sheet_Cycle / _Preg / _Post designs.
 */
export function LogSheet() {
  const t = useTranslations('nav');
  const router = useRouter();
  const modeQuery = useUserMode();

  if (modeQuery.isPending && modeQuery.fetchStatus !== 'idle') {
    return (
      <SkeletonGroup label={t('logSheet.loading')} className="nblog-grid">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  }

  const mode = modeQuery.data
    ? resolveNavMode({ mode: modeQuery.data.mode, isTtc: modeQuery.data.isTtc })
    : 'cycle';

  return (
    <div className="nblog">
      <p className="nblog-lead">{t('logSheet.lead')}</p>
      <div className="nblog-grid">
        {logEntries(mode).map((entry) => (
          <TileButton
            key={entry.key}
            layout="card"
            icon={entry.icon}
            tone={entry.tone}
            label={t(`logSheet.${entry.key}.title`)}
            sub={t(`logSheet.${entry.key}.sub`)}
            onClick={() => (entry.href ? router.push(entry.href) : openSheet('reminders-add'))}
          />
        ))}
      </div>
    </div>
  );
}
