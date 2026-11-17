'use client';

import clsx from 'clsx';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type AlertItem, type AlertLevel, type AlertListState, groupByLevel, useAlerts } from '@/entities/alert';
import { type AppNotification, useNotifications } from '@/entities/notification';
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
} from '@/features/read-notifications';
import { pilot } from '@/shared/pilot';
import { diffInDays, formatLongDate, today } from '@/shared/lib/date';
import type { Locale } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { AlertCard } from './AlertCard';

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

// ── Health alerts, on top of the notifications ────────────────
// Every action lives on the card itself; nothing here navigates away.
// Warnings (urgent / follow-up) and announcements (suggestion / info) render in separate groups of the sheet.
const WARNING_LEVELS: readonly AlertLevel[] = ['urgent', 'follow_up'];
const isWarning = (a: AlertItem) => WARNING_LEVELS.includes(a.level);

function AlertCards({ items, history }: { items: AlertItem[]; history: boolean }) {
  const t = useTranslations('alerts');
  return (
    <div className="notif-alerts">
      {groupByLevel(items).flatMap((section) =>
        section.items.map((alert) => <AlertCard key={alert.id} alert={alert} history={history} t={t} />),
      )}
    </div>
  );
}

function LoadMoreAlerts({ state }: { state: AlertListState }) {
  const t = useTranslations('alerts');
  const query = useAlerts(state);
  if (!query.hasNextPage) return null;
  return (
    <button
      type="button"
      className="notif-alerts-more"
      onClick={() => void query.fetchNextPage()}
      disabled={query.isFetchingNextPage}
    >
      {t('loadMore')}
    </button>
  );
}

function AlertHistoryList() {
  const t = useTranslations('alerts');
  const query = useAlerts('history');
  const items = query.data?.pages.flatMap((p) => p.items) ?? [];
  if (query.isPending) return null;
  if (items.length === 0) return <p className="notif-alerts-empty">{t('empty.history')}</p>;
  return (
    <>
      <AlertCards items={items} history />
      <LoadMoreAlerts state="history" />
    </>
  );
}

/** Past (acknowledged / dismissed) alerts, folded until asked for so the sheet opens with one request less. */
function AlertHistory() {
  const t = useTranslations('notifications');
  const [open, setOpen] = useState(false);
  return (
    <section className="notif-group" aria-label={t('sections.history')}>
      <button type="button" className="notif-history-toggle" aria-expanded={open} onClick={() => setOpen((v) => !v)} {...pilot('notifications.history')}>
        {t('sections.history')}
        <Icon name="chevronDown" size={16} className={clsx('notif-history-chev', open && 'is-open')} />
      </button>
      {open ? <AlertHistoryList /> : null}
    </section>
  );
}

/**
 * The one notifications sheet, opened by the header bell of every hub (home, services, postpartum) and the
 * profile screen, and the only home of health alerts (old `/alerts` links open it too). Open warnings (urgent /
 * follow-up) come first under their own heading; announcements (suggestion / info alerts and in-app notifications)
 * follow under «اعلان‌ها»; the folded alert history closes the sheet. Tapping an
 * unread notification marks it read; the header action clears the lot.
 */
export function NotificationsSheet() {
  const t = useTranslations('notifications');
  const loc = useLocale() as Locale;

  const { data, isPending } = useNotifications();
  const alertsQuery = useAlerts('open');
  const alerts = alertsQuery.data?.pages.flatMap((p) => p.items) ?? [];
  const warnings = alerts.filter(isWarning);
  const announcements = alerts.filter((a) => !isWarning(a));
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

  if (items.length === 0 && alerts.length === 0) {
    return isPending || alertsQuery.isPending ? null : (
      <section className="notif-section">
        <div className="notif-empty">
          <span className="notif-empty-icon">
            <Icon name="bell" size={30} />
          </span>
          <div className="notif-empty-t">{t('empty.title')}</div>
          <p className="notif-empty-b">{t('empty.body')}</p>
        </div>
        <AlertHistory />
      </section>
    );
  }

  return (
    <section className="notif-section">
      {warnings.length > 0 ? (
        <section className="notif-group" aria-label={t('sections.alerts')}>
          <h3 className="notif-group-t">{t('sections.alerts')}</h3>
          <AlertCards items={warnings} history={false} />
        </section>
      ) : null}
      {warnings.length > 0 && (announcements.length > 0 || items.length > 0) ? (
        <h3 className="notif-group-t">{t('sections.notifications')}</h3>
      ) : null}
      {announcements.length > 0 ? <AlertCards items={announcements} history={false} /> : null}
      <LoadMoreAlerts state="open" />
      {unreadCount > 0 ? (
        <div className="notif-actions">
          <button
            type="button"
            className="notif-markall"
            onClick={handleMarkAll}
            {...pilot('notifications.mark-all')}
            disabled={markAllRead.isPending}
          >
            <Icon name="checkCircle" size={16} />
            {t('markAllRead')}
          </button>
        </div>
      ) : null}

      {items.length > 0 ? (
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
      ) : null}
      <AlertHistory />
    </section>
  );
}
