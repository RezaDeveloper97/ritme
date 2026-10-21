import type { PostpartumRecovery, PostpartumVisit, VisitStatus } from '@/entities/postpartum';
import type { IconName, Tone } from '@/shared/ui';

/** How a visit status reads on the «ویزیت‌های بعد از زایمان» card (T-M9-06). */
export const VISIT_LOOK: Record<VisitStatus, { tone: Tone; icon: IconName }> = {
  done: { tone: 'success', icon: 'checkCircle' },
  upcoming: { tone: 'neutral', icon: 'calendar' },
  due: { tone: 'warm', icon: 'bell' },
  overdue: { tone: 'danger', icon: 'warning' },
};

/** The card shows the three visits in the server order; nothing to show without them. */
export function visitRows(visits: PostpartumVisit[]): PostpartumVisit[] {
  return visits.filter((v) => v.dueOn);
}

/**
 * «امروز ویزیت داشتم» logs today's visit (the server places it in its window);
 * offered while a visit is still open. Once today is logged the button undoes it.
 */
export function visitAction(visits: PostpartumVisit[], today: PostpartumRecovery | null): 'log' | 'undo' | null {
  if (today?.postpartumVisit === true) return 'undo';
  return visits.some((v) => !v.done) ? 'log' : null;
}
