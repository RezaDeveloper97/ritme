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
    <div className="sec-tight">
      <div className="card pad-card-sm">
        <div className="tc-head">
          <span className="tc-flame">
            <Icon name="flame" size={18} stroke="currentColor" />
          </span>
          <span className="tc-title">{t('title')}</span>
          <span className="tc-spacer" />
          {challenge.streak > 0 && (
            <span
              title={t('longestStreak', { n: challenge.longestStreak })}
              className="tc-streak"
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
          <span className="tc-item-l">
            <span className="tc-item-ic">
              <Icon name={icon} size={18} stroke="currentColor" />
            </span>
            <span className="tc-item-b">
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
          className="tc-week"
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
            className="tc-caption"
          >
            ✨ {caption}
          </div>
        )}
      </div>
    </div>
  );
}
