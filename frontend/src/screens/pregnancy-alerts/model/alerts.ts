import type { AlertActionV2, AlertLevelV2, PregnancyAlertV2 } from '@/entities/pregnancy';

export interface AlertDayGroup {
  /** `YYYY-MM-DD` the facts happened (Tehran), or `''` when unknown. */
  day: string;
  /** Server-rendered date label of the first alert in the group. */
  label: string | null;
  alerts: PregnancyAlertV2[];
}

/** Most important first (Alerts artboard): پیگیری زودتر › ارزش پیگیری › پیشنهاد › اطلاع. */
const LEVEL_RANK: Record<AlertLevelV2, number> = { urgent: 0, follow_up: 1, suggestion: 2, info: 3 };

const desc = (x: string, y: string) => (x < y ? 1 : x > y ? -1 : 0);

/**
 * Groups alerts by the day their facts happened, newest day first; inside a
 * day the most important level comes first, then the newest alert.
 */
export function groupAlertsByDay(alerts: PregnancyAlertV2[]): AlertDayGroup[] {
  const map = new Map<string, AlertDayGroup>();
  for (const a of alerts) {
    const day = a.factDate ?? a.createdAt?.slice(0, 10) ?? '';
    const g = map.get(day) ?? { day, label: a.dateLabel, alerts: [] };
    g.alerts.push(a);
    map.set(day, g);
  }
  const groups = [...map.values()].sort((x, y) => desc(x.day, y.day));
  for (const g of groups) {
    g.alerts.sort(
      (x, y) => LEVEL_RANK[x.level] - LEVEL_RANK[y.level] || desc(x.createdAt ?? '', y.createdAt ?? ''),
    );
  }
  return groups;
}

/** How a level presents its actions (Alerts artboard, and §10.2 gradient scarcity). */
export type AlertActionStyle = 'button' | 'link' | 'none';

/**
 * follow_up / urgent: a solid primary + outlined buttons; suggestion: its
 * actions as text links, no «دیدم، ممنون»; info: no actions at all.
 */
export function alertActionStyle(level: AlertLevelV2): AlertActionStyle {
  if (level === 'urgent' || level === 'follow_up') return 'button';
  return level === 'suggestion' ? 'link' : 'none';
}

/** The facts tiles («چی دیدیم» / «چقدر مطمئنیم») belong to the cards worth following up. */
export function showsFacts(level: AlertLevelV2): boolean {
  return alertActionStyle(level) === 'button';
}

/** Actions a card shows for its level (acks are dropped from link-style cards). */
export function visibleActions<T extends { r: AlertActionKind }>(level: AlertLevelV2, actions: T[]): T[] {
  switch (alertActionStyle(level)) {
    case 'button':
      return actions;
    case 'link':
      return actions.filter((x) => x.r.kind !== 'server' || x.r.action !== 'ack');
    default:
      return [];
  }
}

/** What a card's action button does. */
export type AlertActionKind =
  | { kind: 'server'; key: 'ack' | 'add_to_visit_note'; action: 'ack' | 'add_to_visit_note' }
  | { kind: 'link'; key: 'log_weight' | 'log_symptoms' | 'open_week'; href: string }
  | { kind: 'tel'; key: 'call'; href: string };

/** Maps an admin-defined action key to a behaviour; `null` = not shown. */
export function resolveAlertAction(action: AlertActionV2, alert: PregnancyAlertV2): AlertActionKind | null {
  switch (action.key) {
    case 'ack':
    case 'add_to_visit_note':
      return alert.isAcked ? null : { kind: 'server', key: action.key, action: action.key };
    case 'log_weight':
      return { kind: 'link', key: 'log_weight', href: '/pregnancy/log?focus=weight' };
    case 'log_symptoms':
      return { kind: 'link', key: 'log_symptoms', href: '/pregnancy/log' };
    case 'open_week':
      return { kind: 'link', key: 'open_week', href: '/pregnancy/weeks' };
    case 'call': {
      const phone = alert.contact?.phone?.replace(/[^\d+]/g, '');
      return phone ? { kind: 'tel', key: 'call', href: `tel:${phone}` } : null;
    }
    default:
      return null;
  }
}
