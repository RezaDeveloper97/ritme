'use client';

import clsx from 'clsx';
import { useFormatter, useTranslations } from 'next-intl';

import { useTodayChallenge, type TodayChallenge } from '@/entities/challenge';
import { useToggleChallenge } from '@/features/complete-challenge';
import { Card, Icon, IconCircle, type IconName } from '@/shared/ui';

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
  num: (n: number) => string,
): string {
  const { from, to } = challenge.cycleDayRange;

  if (from !== null && to !== null) return t('range.between', { from: num(from), to: num(to) });
  if (from !== null) return t('range.from', { n: num(from) });
  if (to !== null) return t('range.to', { n: num(to) });

  return t('range.any');
}

/**
 * «چالش‌های امروز» (Night & Bloom, B-N1-06) — one task the backend picked for this user today, with a tick
 * that records completion. Deliberately nothing else: no streak, no record, no
 * history strip. It is a suggestion she can take or leave, not a game.
 *
 * The card renders nothing while loading or when no challenge is available, so
 * the home feed simply closes up rather than showing an empty shell.
 */
export function TodayChallengeCard() {
  const t = useTranslations('challenge');
  // Locale digits (۱ in fa): the messages take a pre-formatted string, so they
  // stay correct even when the runtime bundle from the API still has plain `{n}`.
  const format = useFormatter();
  const num = (n: number): string => format.number(n);
  const { data: challenge } = useTodayChallenge();
  const toggle = useToggleChallenge();

  if (!challenge) return null;

  const done = challenge.isCompleted;
  const icon = (challenge.category && CATEGORY_ICON[challenge.category]) || 'sparkle';
  // One daily pick today; the card is drawn as the list the design shows
  // («چالش‌های امروز», N از M + bar) so more picks slot in without a redesign.
  const total = 1;
  const completed = done ? 1 : 0;

  return (
    <Card as="section" className="tc-nb" aria-labelledby="tc-nb-title">
      <div className="tc-nb-head">
        <IconCircle icon="star" tone="warm" size="sm" />
        <h2 id="tc-nb-title" className="tc-nb-title">{t('listTitle')}</h2>
        <span className="tc-nb-count">{t('progress', { done: num(completed), total: num(total) })}</span>
      </div>
      <div
        className="tc-nb-bar"
        role="progressbar"
        aria-label={t('progressLabel')}
        aria-valuemin={0}
        aria-valuemax={total}
        aria-valuenow={completed}
      >
        {/* Width is the datum (CLAUDE.md §10.1). */}
        <span className="tc-nb-bar-fill" style={{ width: `${(completed / total) * 100}%` }} />
      </div>

      <button
        type="button"
        onClick={() => toggle.mutate(challenge.id)}
        // The tick flips optimistically; blocking the button until the write
        // lands keeps a double-tap from queueing a second, undoing toggle.
        disabled={toggle.isPending}
        role="checkbox"
        aria-checked={done}
        className={clsx('tc-nb-row', done && 'is-done')}
      >
        <span className="tc-nb-check" aria-hidden>
          {done && <Icon name="check" size={14} strokeWidth={3} />}
        </span>
        <span className="tc-nb-body">
          <span className="tc-nb-name">{challenge.title}</span>
          {challenge.description && <span className="tc-nb-sub">{challenge.description}</span>}
        </span>
        {/* The pick is made for this cycle day, so name it — otherwise a
            day-specific challenge reads as an arbitrary suggestion. */}
        {challenge.cycleDay !== null ? (
          <span className="tc-nb-chip" title={rangeLabel(challenge, t, num)}>
            <Icon name={icon} size={14} strokeWidth={2} />
            {t('cycleDay', { n: num(challenge.cycleDay) })}
          </span>
        ) : null}
      </button>

      <p className="tc-nb-foot">{t('footer')}</p>
    </Card>
  );
}
