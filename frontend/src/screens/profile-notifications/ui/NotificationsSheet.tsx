'use client';

import clsx from 'clsx';

import { useLocale, useTranslations } from 'next-intl';

import { type AppNotification, useNotifications } from '@/entities/notification';
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
} from '@/features/read-notifications';
import { diffInDays, formatLongDate, today } from '@/shared/lib/date';
import type { Locale } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

// ── A single notification row ─────────────────────────────────
// Presentational; unread rows carry a brand-tinted dot and stronger title, and
// are tappable (tapping marks them read). Read rows are static list items.
function NotificationRow({
  item,
  timeLabel,
  onRead,
}: {
  item: AppNotification;
  timeLabel: string;
  onRead?: () => void;
}) {
  const inner = (
    <>
      <span aria-hidden className={clsx('notif-dot', item.isRead && 'is-read')} />
      <span className="notif-body">
        <span className={clsx('notif-row-t', item.isRead && 'is-read')}>{item.title}</span>
        {item.body ? <span className="notif-row-b">{item.body}</span> : null}
        <span className="notif-row-time">{timeLabel}</span>
      </span>
    </>
  );

  if (onRead) {
    return (
      <button type="button" className="notif-row is-unread" onClick={onRead}>
        {inner}
      </button>
    );
  }
  return <div className="notif-row">{inner}</div>;
}

// Hairline between rows, inset past the unread dot (logical start — RTL-safe).
function Divider() {
  return <div className="notif-divider" />;
}

/**
 * The user's notification list, opened as a full sheet from the profile screen.
 * Tapping an unread row marks it read; the header action clears the lot.
 */
export function NotificationsSheet() {
  const t = useTranslations('notifications');
  const loc = useLocale() as Locale;

  const { data, isPending } = useNotifications();
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  // The sheet never shows the raw ISO timestamp: near dates render as calm
  // relative labels, older ones as full localized dates (§7).
  const timeLabel = (iso: string) => {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return '';
    const days = diffInDays(today(), date);
    if (days <= 0) return t('time.today');
    if (days === 1) return t('time.yesterday');
    if (days < 7) return t('time.daysAgo', { days });
    return formatLongDate(date, loc);
  };

  // Newest first, regardless of backend ordering (ISO strings sort lexically).
  const items = [...(data?.items ?? [])].sort((a, b) =>
    b.createdAt.localeCompare(a.createdAt),
  );
  const unreadCount = data?.unreadCount ?? 0;

  const handleRowRead = (item: AppNotification) => {
    if (markRead.isPending) return;
    markRead.mutate(item.id);
  };

  const handleMarkAll = () => {
    if (markAllRead.isPending) return;
    markAllRead.mutate();
  };

  if (items.length === 0) {
    return isPending ? null : (
      <div className="notif-empty">
        <span className="notif-empty-icon">
          <Icon name="bell" size={30} />
        </span>
        <div className="notif-empty-t">{t('empty.title')}</div>
        <p className="notif-empty-b">{t('empty.body')}</p>
      </div>
    );
  }

  return (
    <section className="notif-section">
      {unreadCount > 0 ? (
        <div className="notif-actions">
          <button
            type="button"
            className="notif-markall"
            onClick={handleMarkAll}
            disabled={markAllRead.isPending}
          >
            <Icon name="checkCircle" size={16} />
            {t('markAllRead')}
          </button>
        </div>
      ) : null}

      <div className="card notif-list">
        {items.map((item, index) => (
          <div key={item.id} className="notif-item">
            {index > 0 ? <Divider /> : null}
            <NotificationRow
              item={item}
              timeLabel={timeLabel(item.createdAt)}
              onRead={item.isRead ? undefined : () => handleRowRead(item)}
            />
          </div>
        ))}
      </div>
    </section>
  );
}
