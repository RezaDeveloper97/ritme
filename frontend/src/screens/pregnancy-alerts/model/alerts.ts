import type { AlertActionV2, PregnancyAlertV2 } from '@/entities/pregnancy';

export interface AlertDayGroup {
  /** `YYYY-MM-DD` of the alert's creation (Tehran), or `''` when unknown. */
  day: string;
  /** Server-rendered date label of the first alert in the group. */
  label: string | null;
  alerts: PregnancyAlertV2[];
}

/** Groups alerts by creation day, newest day first, keeping in-day order. */
export function groupAlertsByDay(alerts: PregnancyAlertV2[]): AlertDayGroup[] {
  const map = new Map<string, AlertDayGroup>();
  for (const a of alerts) {
    const day = a.createdAt?.slice(0, 10) ?? '';
    const g = map.get(day) ?? { day, label: a.dateLabel, alerts: [] };
    g.alerts.push(a);
    map.set(day, g);
  }
  return [...map.values()].sort((x, y) => (x.day < y.day ? 1 : x.day > y.day ? -1 : 0));
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
