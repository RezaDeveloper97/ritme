import { ALERT_LEVELS, type AlertItem, type AlertLevel } from './types';

export interface AlertSection {
  level: AlertLevel;
  items: AlertItem[];
}

const desc = (a: string | null, b: string | null) => (a ?? '') < (b ?? '') ? 1 : (a ?? '') > (b ?? '') ? -1 : 0;

/** Sections in level order (urgent → follow_up → suggestion → info); empty levels are left out, newest first inside. */
export function groupByLevel(items: AlertItem[]): AlertSection[] {
  return ALERT_LEVELS.map((level) => ({
    level,
    items: items.filter((i) => i.level === level).sort((a, b) => desc(a.lastAt, b.lastAt)),
  })).filter((s) => s.items.length > 0);
}

/** An open item (not resolved / archived) can still be acted on. */
export function isOpen(item: AlertItem): boolean {
  return item.state !== 'resolved' && item.state !== 'archived';
}

/** «دیدم، ممنون» is offered while the item has not been acknowledged yet. */
export function canAck(item: AlertItem): boolean {
  return item.state === 'active';
}

/** An urgent item has no close action until it is acknowledged (SPEC §5.4.1). */
export function canDismiss(item: AlertItem): boolean {
  if (!isOpen(item) || !item.dismissible) return false;
  return !(item.level === 'urgent' && item.state === 'active');
}

/** Evidence tiles («چی دیدیم» / «چقدر مطمئنیم») exist on follow_up and urgent cards only. */
export function evidenceRows(item: AlertItem): Array<{ key: 'seen' | 'confidence'; value: string }> {
  if ((item.level !== 'urgent' && item.level !== 'follow_up') || !item.evidence) return [];
  const rows: Array<{ key: 'seen' | 'confidence'; value: string }> = [];
  if (item.evidence.seen) rows.push({ key: 'seen', value: item.evidence.seen });
  if (item.evidence.confidence) rows.push({ key: 'confidence', value: item.evidence.confidence });
  return rows;
}

/** `tel:` target of an urgent item's contact when it is a phone number (115, 1480, +98…); else null. */
export function callHref(item: AlertItem): string | null {
  if (item.level !== 'urgent' || !item.evidence?.contact) return null;
  const digits = item.evidence.contact.replace(/[\s\-()]/g, '');
  return /^\+?\d{3,15}$/.test(digits) ? `tel:${digits}` : null;
}
