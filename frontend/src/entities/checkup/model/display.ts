import type { Locale } from '@/shared/i18n';
import { formatMonthLabel, fromApiDate, toParts } from '@/shared/lib/date';
import type { IconName } from '@/shared/ui';

import type { CheckupResult, CheckupStatus } from './types';

/*
 * Display helpers shared by every checkups screen (v14): the status / result
 * glyphs of the outlined pills and the month + year dates the artboards use in
 * lists («مهر ۱۴۰۴»). Colours are the `.ck-status-*` / `.ck-result-*` classes
 * in globals.css, not here.
 */

const STATUS_ICON: Record<CheckupStatus, IconName> = {
  due: 'bellPlain',
  overdue: 'info',
  soon: 'clock',
  up_to_date: 'check',
  not_yet: 'clock',
  disabled: 'x',
};

const RESULT_ICON: Record<CheckupResult, IconName> = {
  normal: 'check',
  follow_up: 'info',
  pending: 'clock',
};

/** The glyph of a status pill (bell «موعدش رسیده», info «عقب‌افتاده», clock «به‌زودی», check «به‌روز»). */
export function checkupStatusIcon(status: CheckupStatus): IconName {
  return STATUS_ICON[status];
}

/** The glyph of a result option / chip (check نرمال, info پیگیری, clock منتظر جواب). */
export function checkupResultIcon(result: CheckupResult): IconName {
  return RESULT_ICON[result];
}

/** A `Y-m-d` date as month + year in the locale's calendar («فروردین ۱۴۰۴» / "April 2025"). */
export function formatCheckupMonth(ymd: string, locale: Locale): string {
  const { year, month } = toParts(fromApiDate(ymd), locale);
  return formatMonthLabel(year, month, locale);
}
