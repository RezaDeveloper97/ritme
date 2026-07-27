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
    <div className="view" style={{ background: 'var(--page)' }}>
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
      <div style={{ padding: '40px 22px 0', textAlign: 'center' }}>
        <div className="dot" style={{ width: 72, height: 72, margin: '0 auto 18px', background: 'var(--surface-2)', color: 'var(--brand)' }}>
          <Icon name="heart" size={34} />
        </div>
        <h1 className="titr" style={{ fontSize: 20 }}>{t('notActive.title')}</h1>
        <p className="sub" style={{ margin: '10px 0 24px', lineHeight: 2 }}>{t('notActive.body')}</p>
        <button className="btn btn-primary" onClick={start} disabled={activate.isPending} style={{ borderRadius: 16 }}>
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
      className="card"
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8, padding: '14px 8px', cursor: 'pointer', fontFamily: 'inherit', flex: 1 }}
    >
      <span className="dot" style={{ width: 40, height: 40, background: color + '1A', color }}>
        <Icon name={icon} size={20} />
      </span>
      <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--ink)', textAlign: 'center', lineHeight: 1.4 }}>{label}</span>
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
        <div style={{ padding: '60px 0', textAlign: 'center', color: 'var(--muted)' }}>{t('loading')}</div>
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
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="scroll">
        {/* Hero */}
        <div className="home-grad" style={{ padding: '18px 18px 22px', color: 'var(--on-accent)' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 13, fontWeight: 700, opacity: 0.92 }}>{t('title')}</span>
            {status?.isHighRisk && (
              <span style={{ fontSize: 11, fontWeight: 800, background: 'rgba(255,255,255,.22)', borderRadius: 20, padding: '3px 10px' }}>
                {t('highRiskBadge')}
              </span>
            )}
          </div>

          <div style={{ marginTop: 14, display: 'flex', alignItems: 'center', gap: 14 }}>
            <div className="dot" style={{ width: 62, height: 62, background: 'rgba(255,255,255,.2)', color: 'var(--on-accent)', flex: '0 0 auto' }}>
              <Icon name="heart" size={30} />
            </div>
            <div style={{ textAlign: 'start' }}>
              <div style={{ fontSize: 22, fontWeight: 900 }}>{progress ? t('hero.weekOf', { week: progress.currentWeek }) : ''}</div>
              {ga?.weeks != null && (
                <div style={{ fontSize: 13, fontWeight: 600, opacity: 0.95, marginTop: 3 }}>
                  {t('hero.gaValue', { weeks: ga.weeks, days: ga.days ?? 0 })}
                </div>
              )}
            </div>
          </div>

          {/* Progress bar across 40 weeks */}
          {progress && (
            <div style={{ marginTop: 16 }}>
              <div style={{ height: 8, borderRadius: 20, background: 'rgba(255,255,255,.28)', overflow: 'hidden' }}>
                <div style={{ height: '100%', width: `${progress.progressPct}%`, background: 'var(--on-accent)', borderRadius: 20 }} />
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 7, fontSize: 11.5, fontWeight: 600, opacity: 0.95 }}>
                <span>{t('progress', { percent: progress.progressPct })}</span>
                <span>{t(`trimester.t${progress.trimester}`)}</span>
              </div>
            </div>
          )}
        </div>

        {/* Due date + confidence */}
        <div style={{ padding: '14px 16px 0', display: 'flex', gap: 10 }}>
          {due && (
            <div className="card" style={{ flex: 1, padding: '12px 13px', textAlign: 'start' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--muted)', fontSize: 11.5, fontWeight: 700 }}>
                <Icon name="calendar" size={13} /> {t('due.label')}
              </div>
              <div style={{ fontSize: 14.5, fontWeight: 800, color: 'var(--ink)', marginTop: 5 }}>
                {pickBilingual(due.formatted, locale)}
              </div>
              <div style={{ fontSize: 12, color: 'var(--brand)', fontWeight: 700, marginTop: 3 }}>
                {t('due.countdown', { n: due.daysRemaining })}
              </div>
            </div>
          )}
          {ga?.confidenceLevel && (
            <div className="card" style={{ flex: 1, padding: '12px 13px', textAlign: 'start' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--muted)', fontSize: 11.5, fontWeight: 700 }}>
                <Icon name="info" size={13} /> {t('confidence.label')}
              </div>
              <div style={{ fontSize: 14.5, fontWeight: 800, color: 'var(--ink)', marginTop: 5 }}>
                {t(`confidence.${ga.confidenceLevel === 'high' ? 'high' : ga.confidenceLevel === 'low' ? 'low' : 'medium'}`)}
              </div>
              {ga.uncertaintyDays != null && (
                <div style={{ fontSize: 12, color: 'var(--muted)', marginTop: 3 }}>±{ga.uncertaintyDays}</div>
              )}
            </div>
          )}
        </div>

        {/* Quick actions */}
        <div style={{ padding: '12px 16px 0', display: 'flex', gap: 10 }}>
          <ActionTile icon="plus" label={t('actions.logSymptoms')} color="var(--brand)" onClick={() => goToLog('symptoms')} />
          <ActionTile icon="stetho" label={t('actions.weeklyCheckup')} color="var(--indigo-deep)" onClick={() => goToLog('weekly')} />
          <ActionTile icon="heart" label={t('actions.fetalMovement')} color="var(--green)" onClick={() => goToLog('movement')} />
        </div>

        {/* Week browser */}
        <div style={{ padding: '18px 16px 0' }}>
          <div className="card" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px' }}>
            <button className="iconbtn" aria-label={t('weekPicker.prev')} disabled={week <= 1} onClick={() => setSelectedWeek(Math.max(1, week - 1))} style={{ opacity: week <= 1 ? 0.3 : 1 }}>
              <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
            </button>
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)' }}>{t('weekLabel', { week })}</div>
              {!isCurrent && (
                <button onClick={() => progress && setSelectedWeek(progress.currentWeek)} style={{ background: 'none', border: 0, color: 'var(--brand)', fontSize: 11.5, fontWeight: 700, cursor: 'pointer', fontFamily: 'inherit', marginTop: 2 }}>
                  {t('weekPicker.current')}
                </button>
              )}
            </div>
            <button className="iconbtn" aria-label={t('weekPicker.next')} disabled={week >= TOTAL_WEEKS} onClick={() => setSelectedWeek(Math.min(TOTAL_WEEKS, week + 1))} style={{ opacity: week >= TOTAL_WEEKS ? 0.3 : 1 }}>
              <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
            </button>
          </div>
        </div>

        {/* Weekly content */}
        <div style={{ padding: '12px 16px 0' }}>
          {contentQuery.isLoading ? (
            <div className="card" style={{ padding: '18px 14px', textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>{t('loading')}</div>
          ) : (
            <WeekContent content={contentQuery.data ?? null} t={t} />
          )}
        </div>

        {/* Alerts */}
        <div style={{ padding: '20px 16px 0' }}>
          <AlertsCard t={t} />
        </div>

        <div style={{ height: 24 }} />
      </div>

      <BottomNav />
    </div>
  );
}
