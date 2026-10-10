'use client';

import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { IconCircle } from '@/shared/ui';

import { CONDITION_LOOK } from '../model/programs';
import type { ConditionNudge } from '../model/types';

/** Shared by the programs hub (CB-COND-02) and the cycle home (CB-COND-07b). */
/** A soft suggestion: heavy pain / heavy flow logged on 2+ days this cycle while not in the matching program. */
export function ConditionNudgeCard({ nudge }: { nudge: ConditionNudge }) {
  const t = useTranslations('conditions.nudge');
  const look = CONDITION_LOOK[nudge.program];
  return (
    <section className="nb-card cnd-nudge" aria-label={t('label')}>
      <div className="cnd-nudge-head">
        <IconCircle icon={look.icon} tone={look.tone} size="sm" />
        <h3 className="cnd-nudge-title">{nudge.title}</h3>
      </div>
      {nudge.body ? <p className="cnd-nudge-body">{nudge.body}</p> : null}
      {nudge.link || nudge.doctor ? (
        <div className="cnd-nudge-actions">
          {nudge.link && nudge.action ? (
            <Link href={nudge.link} className="cnd-nudge-link">
              {nudge.action}
            </Link>
          ) : null}
          {nudge.doctor ? (
            <Link href={nudge.doctor.link} className="cnd-nudge-link is-quiet">
              {nudge.doctor.action}
            </Link>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
