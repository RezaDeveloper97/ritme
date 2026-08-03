'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { useTodayChallenge, type TodayChallenge } from '@/entities/challenge';
import { useToggleChallenge } from '@/features/complete-challenge';
import { Icon, type IconName } from '@/shared/ui';

/** Category → icon. Unknown/absent categories fall back to a neutral sparkle. */
const CATEGORY_ICON: Record<string, IconName> = {
  tracking: 'thermo',
  mindfulness: 'sparkle',
  nutrition: 'glass',
  exercise: 'walk',
  sleep: 'moon',
};

/**
 * Tooltip for the day chip: why *this* challenge showed up today. Untargeted
 * challenges have nothing to explain, so they fall back to the plain day.
 */
function rangeLabel(
  challenge: TodayChallenge,
  t: ReturnType<typeof useTranslations<'challenge'>>,
): string {
  const { from, to } = challenge.cycleDayRange;

  if (from !== null && to !== null) return t('range.between', { from, to });
  if (from !== null) return t('range.from', { n: from });
  if (to !== null) return t('range.to', { n: to });

  return t('range.any');
}

/**
 * «چالش امروز» — one task the backend picked for this user today, with a tick
 * that records completion. Deliberately nothing else: no streak, no record, no
 * history strip. It is a suggestion she can take or leave, not a game.
 *
 * The card renders nothing while loading or when no challenge is available, so
 * the home feed simply closes up rather than showing an empty shell.
 */
export function TodayChallengeCard() {
  const t = useTranslations('challenge');
  const { data: challenge } = useTodayChallenge();
  const toggle = useToggleChallenge();

  if (!challenge) return null;

  const done = challenge.isCompleted;
  const icon = (challenge.category && CATEGORY_ICON[challenge.category]) || 'sparkle';

  return (
    <div className="sec-tight">
      <div className="card pad-card-sm">
        <div className="tc-head">
          <span className="tc-mark">
            <Icon name="sparkle" size={18} stroke="currentColor" />
          </span>
          <span className="tc-title">{t('title')}</span>
          <span className="tc-spacer" />
          {/* The pick is made for this cycle day, so name it — otherwise a
              day-specific challenge reads as an arbitrary suggestion. */}
          {challenge.cycleDay !== null && (
            <span className="tc-day-chip" title={rangeLabel(challenge, t)}>
              {t('cycleDay', { n: challenge.cycleDay })}
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
          className={clsx('tc-item', done && 'is-done')}
        >
          <span className="tc-item-l">
            <span className="tc-item-ic">
              <Icon name={icon} size={18} stroke="currentColor" />
            </span>
            <span className="tc-item-b">
              <span className="tc-item-t">{challenge.title}</span>
            </span>
          </span>
          <span className={`cbx${done ? ' on' : ''}`} aria-hidden="true">
            <Icon name="check" size={14} stroke="var(--on-accent)" />
          </span>
        </button>

        {challenge.description && <div className="tc-caption">{challenge.description}</div>}
      </div>
    </div>
  );
}
