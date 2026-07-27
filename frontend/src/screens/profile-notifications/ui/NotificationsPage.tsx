'use client';

import clsx from 'clsx';

import { useLocale, useTranslations } from 'next-intl';

import { type AppNotification, useNotifications } from '@/entities/notification';
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
} from '@/features/read-notifications';
import { diffInDays, formatJalali, today } from '@/shared/lib/date';
import { type Locale, useRouter } from '@/shared/i18n';
import { Icon, NavBack } from '@/shared/ui';

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
      <span
        aria-hidden
        style={{
          width: 9,
          height: 9,
          borderRadius: '50%',
          flexShrink: 0,
          marginTop: 6,
          background: item.isRead ? 'transparent' : 'var(--brand)',
          border: item.isRead ? '1px solid var(--line)' : 'none',
        }}
      />
      <span className="notif-body">
        <span className={clsx('notif-row-t', item.isRead && 'is-read')}>
          {item.title}
        </span>
        {item.body ? (
          <span className="notif-row-b">
            {item.body}
          </span>
        ) : null}
        <span className="notif-row-time">
          {timeLabel}
        </span>
      </span>
    </>
  );

  const style = {
    display: 'flex',
    alignItems: 'flex-start',
    gap: 12,
    width: '100%',
    padding: '13px 14px',
    background: item.isRead ? 'transparent' : 'var(--pink-soft)',
    border: 0,
    font: 'inherit',
    textAlign: 'start',
    cursor: onRead ? 'pointer' : 'default',
  } as const;

  if (onRead) {
    return (
      <button type="button" style={style} onClick={onRead}>
        {inner}
      </button>
    );
  }
  return <div style={style}>{inner}</div>;
}

// Hairline between rows, inset past the unread dot (logical start — RTL-safe).
function Divider() {
  return <div className="notif-divider" />;
}

export function NotificationsPage() {
  const t = useTranslations('notifications');
  const loc = useLocale() as Locale;
  const router = useRouter();

  const { data, isPending } = useNotifications();
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  // The screen never shows the raw ISO timestamp: near dates render as calm
  // relative labels, older ones as full Jalali dates (§7).
  const timeLabel = (iso: string) => {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return '';
    const days = diffInDays(today(), date);
    if (days <= 0) return t('time.today');
    if (days === 1) return t('time.yesterday');
    if (days < 7) return t('time.daysAgo', { days });
    return formatJalali(date, loc);
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

  return (
    <div className="view prof-page">
      <div className="hdr">
        <div className="notif-hdr-left">
          <NavBack onClick={() => router.back()} label={t('back')} />
          <span className="notif-title">
            {t('title')}
          </span>
        </div>
        {unreadCount > 0 ? (
          <button
            type="button"
            onClick={handleMarkAll}
            disabled={markAllRead.isPending}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              background: 'transparent',
              border: 0,
              font: 'inherit',
              fontSize: 13,
              fontWeight: 700,
              color: 'var(--brand)',
              cursor: 'pointer',
              opacity: markAllRead.isPending ? 0.55 : 1,
            }}
          >
            <Icon name="checkCircle" size={16} />
            {t('markAllRead')}
          </button>
        ) : null}
      </div>

      <div className="scroll notif-scroll">
        {items.length > 0 ? (
          <section className="notif-section">
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
        ) : isPending ? null : (
          <div className="notif-empty">
            <span className="notif-empty-icon">
              <Icon name="bell" size={30} />
            </span>
            <div className="notif-empty-t">
              {t('empty.title')}
            </div>
            <p className="notif-empty-b">
              {t('empty.body')}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
