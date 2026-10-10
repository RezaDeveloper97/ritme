'use client';

import { useConditionNudges, useConditions } from '../api/queries';
import { homeNudges } from '../model/home-nudges';
import { ConditionNudgeCard } from './ConditionNudgeCard';

/** CB-COND-07b: the soft condition suggestion on the cycle home — nothing unless `/messages/nudges` has a hit. */
export function HomeConditionNudge() {
  const nudges = useConditionNudges();
  const overview = useConditions();
  const list = homeNudges(nudges.data, overview.data);
  if (!list.length) return null;
  return (
    <>
      {list.map((n) => (
        <ConditionNudgeCard key={n.key} nudge={n} />
      ))}
    </>
  );
}
