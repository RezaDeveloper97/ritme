'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useMessageFeed } from '@/entities/message';
import { useNotifications } from '@/entities/notification';
import type { Locale } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

const FA = '۰۱۲۳۴۵۶۷۸۹';
const localizeNum = (n: number, loc: Locale) => (loc === 'fa' ? String(n).replace(/[0-9]/g, (d) => FA[Number(d)]) : String(n));

/**
 * The one header bell (home, services, postpartum…). It always opens the same notifications sheet, which lists
 * open health alerts first and then in-app notifications. The badge counts both; an urgent alert tints it red.
 * `className` is the host header's button style (`home-hdr-btn` or `nb-hbtn is-soft`).
 */
export function NotificationBell({ className }: { className: string }) {
  const t = useTranslations('notifications');
  const loc = useLocale() as Locale;
  const mounted = useMounted();
  const alerts = useMessageFeed().data?.alerts;
  const unread = useNotifications().data?.unreadCount ?? 0;

  const openAlerts = alerts?.open ?? 0;
  const urgent = alerts?.urgent ?? 0;
  const count = mounted ? openAlerts + unread : 0;
  const isUrgent = mounted && urgent > 0;

  const label =
    count === 0 ? t('title') : isUrgent ? t('bell.urgent', { count, urgent }) : t('bell.count', { count });

  return (
    <button
      type="button"
      className={clsx(className, 'relative', isUrgent && 'text-[var(--danger)]')}
      onClick={() => openSheet('notifications')}
      aria-label={label}
    >
      <Icon name="bell" size={20} strokeWidth={1.8} />
      {count > 0 ? (
        <span
          aria-hidden
          className={clsx(
            'pointer-events-none absolute -top-1 -end-1 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-bold leading-none',
            isUrgent ? 'bg-[var(--danger)] text-[var(--on-danger)]' : 'bg-[var(--brand-fill)] text-[var(--on-brand)]',
          )}
        >
          {localizeNum(Math.min(count, 99), loc)}
        </span>
      ) : null}
    </button>
  );
}
