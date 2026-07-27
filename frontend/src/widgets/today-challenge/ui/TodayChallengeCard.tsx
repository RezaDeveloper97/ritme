'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useTodayChallenge, type ChallengeDifficulty } from '@/entities/challenge';
import { useToggleChallenge } from '@/features/complete-challenge';
import type { Locale } from '@/shared/i18n';
import { formatJalaliDayMonth, fromApiDate } from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';

/** Category → icon. Unknown/absent categories fall back to a neutral sparkle. */
const CATEGORY_ICON: Record<string, IconName> = {
  tracking: 'thermo',
  mindfulness: 'sparkle',
  nutrition: 'glass',
  exercise: 'walk',
  sleep: 'moon',
};

const DIFFICULTY_COLOR: Record<ChallengeDifficulty, string> = {
  easy: 'var(--green)',
  medium: 'var(--amber)',
  hard: 'var(--danger)',
};

/**
 * «چالش امروز» — the challenge the backend picked for this user today, with a
 * tick that records completion and a seven-day streak strip.
 *
 * The card renders nothing while loading or when no challenge is available, so
 * the home feed simply closes up rather than showing an empty shell.
 */
export function TodayChallengeCard() {
  const t = useTranslations('challenge');
  const locale = useLocale() as Locale;
  const { data: challenge } = useTodayChallenge();
  const toggle = useToggleChallenge();

  if (!challenge) return null;

  const done = challenge.isCompleted;
  const icon = (challenge.category && CATEGORY_ICON[challenge.category]) || 'sparkle';
  const caption = challenge.statusMessage ?? challenge.description;

  return (
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '14px 12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
          <span style={{ color: 'var(--amber)' }}>
            <Icon name="flame" size={18} stroke="currentColor" />
          </span>
          <span style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)' }}>{t('title')}</span>
          <span style={{ flex: 1 }} />
          {challenge.streak > 0 && (
            <span
              title={t('longestStreak', { n: challenge.longestStreak })}
              style={{
                display: 'flex', alignItems: 'center', gap: 4,
                background: 'var(--amber-tint)', color: 'var(--amber-deep)',
                borderRadius: 999, padding: '3px 9px',
                fontSize: 11.5, fontWeight: 800,
              }}
            >
              <Icon name="flame" size={13} stroke="currentColor" />
              {t('streak', { n: challenge.streak })}
            </span>
          )}
        </div>

        <button
          type="button"
          onClick={() => toggle.mutate(challenge.id)}
          // The tick flips optimistically; blocking the button until the write
          // lands keeps a double-tap from queueing a second, undoing toggle.
          disabled={toggle.isPending}
          aria-pressed={done}
          style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            width: '100%', gap: 10, textAlign: 'start', cursor: 'pointer',
            fontFamily: 'inherit',
            border: `2px solid ${done ? 'var(--green-deep)' : 'var(--line)'}`,
            background: done ? 'var(--green-tint)' : 'var(--surface)',
            borderRadius: 8, padding: 12,
            transition: 'background .15s, border-color .15s',
          }}
        >
          <span style={{ display: 'flex', alignItems: 'center', gap: 10, minWidth: 0 }}>
            <span style={{ color: 'var(--brand)', flex: '0 0 auto' }}>
              <Icon name={icon} size={18} stroke="currentColor" />
            </span>
            <span style={{ minWidth: 0 }}>
              <span
                style={{
                  display: 'block', fontSize: 14, fontWeight: 700, color: 'var(--ink)',
                  textDecoration: done ? 'line-through' : undefined,
                }}
              >
                {challenge.title}
              </span>
              {challenge.difficulty && (
                <span
                  style={{
                    display: 'block', marginTop: 3, fontSize: 11, fontWeight: 700,
                    color: DIFFICULTY_COLOR[challenge.difficulty],
                  }}
                >
                  {t(`difficulty.${challenge.difficulty}`)}
                </span>
              )}
            </span>
          </span>
          <span className={`cbx${done ? ' on' : ''}`} aria-hidden="true">
            <Icon name="check" size={14} stroke="var(--on-accent)" />
          </span>
        </button>

        {/* Seven-day strip — the "سابقه" the streak is built from. */}
        <div
          style={{ display: 'flex', gap: 6, justifyContent: 'space-between', marginTop: 12 }}
          aria-label={t('weekLabel')}
        >
          {challenge.weekDays.map((day) => (
            <span
              key={day.date}
              title={formatJalaliDayMonth(fromApiDate(day.date), locale)}
              style={{
                flex: 1, height: 6, borderRadius: 999,
                background: day.isCompleted ? 'var(--green)' : 'var(--line-2)',
                outline: day.isToday ? '2px solid var(--brand)' : undefined,
                outlineOffset: 2,
              }}
            />
          ))}
        </div>

        {caption && (
          <div
            style={{
              marginTop: 14,
              background: 'linear-gradient(90deg,var(--pink-bg),var(--violet-soft))',
              borderRadius: 12, padding: '8px 12px', textAlign: 'center',
              fontSize: 12, fontWeight: 600, color: 'var(--steel)',
            }}
          >
            ✨ {caption}
          </div>
        )}
      </div>
    </div>
  );
}
