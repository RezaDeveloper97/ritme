'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { useHealthLog, useSaveHealthLog } from '@/entities/health-log';
import { Link, useRouter } from '@/shared/i18n';
import { toApiDate, today } from '@/shared/lib/date';
import { Card, HeroCard, Icon, IconCircle, ListRow, SectionTitle, type IconName, type Tone } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { CheckupsCard } from '@/widgets/checkups-card';
import { TodayChallengeCard } from '@/widgets/today-challenge';
import { TodayRemindersCard } from '@/widgets/today-reminders';

interface TileProps {
  icon: IconName;
  tone: Tone;
  label: string;
  status: string;
  on: boolean;
}

function TileBody({ icon, tone, label, status, on }: TileProps) {
  return (
    <>
      <IconCircle icon={icon} tone={tone} size="md" />
      <span className="meno-tile-label">{label}</span>
      <span className={clsx('meno-tile-status', on && 'is-on')}>
        {on ? <Icon name="check" size={12} strokeWidth={2.6} /> : null}
        {status}
      </span>
    </>
  );
}

/**
 * Minimal menopause home (B-N2-03; no artboard — gaps.md #3, decision recorded
 * in docs/night-bloom/README.md): the cycle-home layout with menopause copy, no
 * cycle ring, predictions or fertility, and three quick symptom tiles — hot
 * flash (one tap logs / un-logs today), sleep and mood (open the day log) —
 * plus «روند علائم». roadmap CB-MENO-05 replaces this whole screen.
 */
export function MenopauseHome({ header }: { header: ReactNode }) {
  const t = useTranslations('home.life.menopause');
  const router = useRouter();
  const day = toApiDate(today());
  const log = useHealthLog(day);
  const save = useSaveHealthLog();
  const entry = log.data ?? null;
  const hotFlash = entry?.hot_flashes === true;
  const sleepOn = Boolean(entry?.sleep_quality || entry?.sleep_duration);
  const moodOn = Boolean(entry?.moods && entry.moods.length > 0);

  return (
    <div className="view">
      <div className="home-grad home-grad-fill" />
      <div className="scroll page-scroll">
        <div className="home-top">
          {header}
          <HeroCard as="section" className="meno-hero" aria-labelledby="meno-title">
            <span className="meno-over">{t('overline')}</span>
            <h2 id="meno-title" className="meno-title">
              {t('title')}
            </h2>
            <p className="meno-body">{t('body')}</p>
            <Link href="/log" className="home-ring-edit meno-log">
              <Icon name="plus" size={16} strokeWidth={2.4} />
              {t('logToday')}
            </Link>
          </HeroCard>
          <section className="meno-tiles-wrap" aria-labelledby="meno-tiles">
            <SectionTitle id="meno-tiles" title={t('tilesTitle')} />
            <div className="meno-tiles">
              <button
                type="button"
                className={clsx('meno-tile', hotFlash && 'is-on')}
                aria-pressed={hotFlash}
                aria-label={hotFlash ? t('hotFlashUndo') : `${t('hotFlash')} · ${t('hotFlashAdd')}`}
                disabled={log.isPending || save.isPending}
                onClick={() => save.mutate({ log_date: day, hot_flashes: !hotFlash })}
              >
                <TileBody
                  icon="flame"
                  tone="warm"
                  label={t('hotFlash')}
                  status={hotFlash ? t('hotFlashOn') : t('hotFlashAdd')}
                  on={hotFlash}
                />
              </button>
              <Link href="/log" className={clsx('meno-tile', sleepOn && 'is-on')}>
                <TileBody
                  icon="moon"
                  tone="brand"
                  label={t('sleep')}
                  status={sleepOn ? t('logged') : t('notLogged')}
                  on={sleepOn}
                />
              </Link>
              <Link href="/log" className={clsx('meno-tile', moodOn && 'is-on')}>
                <TileBody
                  icon="smile"
                  tone="bloom"
                  label={t('mood')}
                  status={moodOn ? t('logged') : t('notLogged')}
                  on={moodOn}
                />
              </Link>
            </div>
            {save.isError ? (
              <p className="mode-error" role="alert">
                {t('saveError')}
              </p>
            ) : null}
          </section>
        </div>
        <div className="ch-feed">
          <Card className="meno-trend">
            <ListRow
              icon="chart"
              iconTone="data"
              title={t('trendTitle')}
              description={t('trendSub')}
              onClick={() => router.push('/cycle/symptoms')}
            />
          </Card>
          <TodayRemindersCard />
          <CheckupsCard />
          <TodayChallengeCard />
        </div>
        <div className="page-tail" />
      </div>
      <BottomNav />
    </div>
  );
}
