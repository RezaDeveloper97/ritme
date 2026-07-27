'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  derivePregnancyProgress,
  pickBilingual,
  useActivatePregnancy,
  usePregnancyProfile,
  usePregnancyStatus,
  useWeeklyContent,
  TOTAL_WEEKS,
} from '@/entities/pregnancy';
import { useRouter, type Locale } from '@/shared/i18n';
import { Icon, type IconName } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { AlertsCard } from './AlertsCard';
import { WeekContent } from './WeekContent';

type T = ReturnType<typeof useTranslations>;

// ── Loading / gate shells ──────────────────────────────────────
function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="view preg-page">
      <div className="scroll">{children}</div>
      <BottomNav />
    </div>
  );
}

function ActivateGate({ t }: { t: T }) {
  const router = useRouter();
  const activate = useActivatePregnancy();

  const start = () => {
    activate.mutate(undefined, {
      onSuccess: () => router.push('/pregnancy/onboarding'),
    });
  };

  return (
    <Shell>
      <div className="preg-gate">
        <div className="dot preg-gate-dot">
          <Icon name="heart" size={34} />
        </div>
        <h1 className="titr preg-gate-title">{t('notActive.title')}</h1>
        <p className="sub preg-gate-body">{t('notActive.body')}</p>
        <button className="btn btn-primary preg-gate-cta" onClick={start} disabled={activate.isPending}>
          {activate.isPending ? t('notActive.activating') : t('notActive.cta')}
        </button>
      </div>
    </Shell>
  );
}

// ── Quick action tile ──────────────────────────────────────────
function ActionTile({ icon, label, color, onClick }: { icon: IconName; label: string; color: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="card preg-action"
    >
      <span className="dot preg-action-dot" style={{ background: color + '1A', color }}>
        <Icon name={icon} size={20} />
      </span>
      <span className="preg-action-label">{label}</span>
    </button>
  );
}

// ── Main export ────────────────────────────────────────────────
export function PregnancyPage() {
  const t = useTranslations('pregnancy');
  const locale = useLocale() as Locale;
  const isRtl = locale === 'fa';
  const router = useRouter();

  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  const statusQuery = usePregnancyStatus();
  const profileQuery = usePregnancyProfile();
  const status = statusQuery.data;
  const progress = derivePregnancyProgress(status?.currentWeek ?? null);

  const [selectedWeek, setSelectedWeek] = useState<number | null>(null);
  // Default the browser to the current week once status settles.
  useEffect(() => {
    if (progress && selectedWeek == null) setSelectedWeek(progress.currentWeek);
  }, [progress, selectedWeek]);

  const week = selectedWeek ?? progress?.currentWeek ?? 1;
  const contentQuery = useWeeklyContent(week, locale);

  if (!mounted || statusQuery.isLoading || profileQuery.isLoading) {
    return (
      <Shell>
        <div className="preg-loading">{t('loading')}</div>
      </Shell>
    );
  }

  const onboarded = !!profileQuery.data?.profile.onboardingCompleted && progress != null;
  if (!onboarded) return <ActivateGate t={t} />;

  const ga = status?.gestationalAge;
  const due = status?.estimatedDueDate;
  const goToLog = (tab: string) => router.push(`/pregnancy/log?tab=${tab}`);
  const isCurrent = progress != null && week === progress.currentWeek;

  return (
    <div className="view preg-page">
      <div className="scroll">
        {/* Hero */}
        <div className="home-grad preg-hero">
          <div className="preg-hero-top">
            <span className="preg-hero-title">{t('title')}</span>
            {status?.isHighRisk && (
              <span className="preg-hero-badge">
                {t('highRiskBadge')}
              </span>
            )}
          </div>

          <div className="preg-hero-row">
            <div className="dot preg-hero-dot">
              <Icon name="heart" size={30} />
            </div>
            <div className="text-start">
              <div className="preg-hero-week">{progress ? t('hero.weekOf', { week: progress.currentWeek }) : ''}</div>
              {ga?.weeks != null && (
                <div className="preg-hero-ga">
                  {t('hero.gaValue', { weeks: ga.weeks, days: ga.days ?? 0 })}
                </div>
              )}
            </div>
          </div>

          {/* Progress bar across 40 weeks */}
          {progress && (
            <div className="preg-prog">
              <div className="preg-prog-track">
                <div className="preg-prog-fill" style={{ width: `${progress.progressPct}%` }} />
              </div>
              <div className="preg-prog-meta">
                <span>{t('progress', { percent: progress.progressPct })}</span>
                <span>{t(`trimester.t${progress.trimester}`)}</span>
              </div>
            </div>
          )}
        </div>

        {/* Due date + confidence */}
        <div className="preg-tiles">
          {due && (
            <div className="card preg-tile">
              <div className="preg-tile-head">
                <Icon name="calendar" size={13} /> {t('due.label')}
              </div>
              <div className="preg-tile-val">
                {pickBilingual(due.formatted, locale)}
              </div>
              <div className="preg-tile-sub is-brand">
                {t('due.countdown', { n: due.daysRemaining })}
              </div>
            </div>
          )}
          {ga?.confidenceLevel && (
            <div className="card preg-tile">
              <div className="preg-tile-head">
                <Icon name="info" size={13} /> {t('confidence.label')}
              </div>
              <div className="preg-tile-val">
                {t(`confidence.${ga.confidenceLevel === 'high' ? 'high' : ga.confidenceLevel === 'low' ? 'low' : 'medium'}`)}
              </div>
              {ga.uncertaintyDays != null && (
                <div className="preg-tile-sub">±{ga.uncertaintyDays}</div>
              )}
            </div>
          )}
        </div>

        {/* Quick actions */}
        <div className="preg-actions">
          <ActionTile icon="plus" label={t('actions.logSymptoms')} color="var(--brand)" onClick={() => goToLog('symptoms')} />
          <ActionTile icon="stetho" label={t('actions.weeklyCheckup')} color="var(--indigo-deep)" onClick={() => goToLog('weekly')} />
          <ActionTile icon="heart" label={t('actions.fetalMovement')} color="var(--green)" onClick={() => goToLog('movement')} />
        </div>

        {/* Week browser */}
        <div className="preg-weeknav">
          <div className="card preg-weeknav-card">
            <button className="iconbtn" aria-label={t('weekPicker.prev')} disabled={week <= 1} onClick={() => setSelectedWeek(Math.max(1, week - 1))}>
              <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
            </button>
            <div className="preg-weeknav-mid">
              <div className="preg-weeknav-week">{t('weekLabel', { week })}</div>
              {!isCurrent && (
                <button className="preg-weeknav-now" onClick={() => progress && setSelectedWeek(progress.currentWeek)}>
                  {t('weekPicker.current')}
                </button>
              )}
            </div>
            <button className="iconbtn" aria-label={t('weekPicker.next')} disabled={week >= TOTAL_WEEKS} onClick={() => setSelectedWeek(Math.min(TOTAL_WEEKS, week + 1))}>
              <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
            </button>
          </div>
        </div>

        {/* Weekly content */}
        <div className="preg-content">
          {contentQuery.isLoading ? (
            <div className="card preg-content-load">{t('loading')}</div>
          ) : (
            <WeekContent content={contentQuery.data ?? null} t={t} />
          )}
        </div>

        {/* Alerts */}
        <div className="preg-alerts">
          <AlertsCard t={t} />
        </div>

        <div className="preg-tail" />
      </div>

      <BottomNav />
    </div>
  );
}
