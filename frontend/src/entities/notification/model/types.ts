/**
 * An in-app notification shown on the profile notifications screen.
 * Named `AppNotification` to avoid clashing with the DOM `Notification` type.
 */
export interface AppNotification {
  id: string;
  /** Backend-defined category (e.g. reminder, content); display-only here. */
  type: string;
  title: string;
  body: string;
  /** Optional deep link the notification points at; null when informational. */
  actionUrl: string | null;
  isRead: boolean;
  /** ISO timestamp of when it was read; null while unread. */
  readAt: string | null;
  /** ISO/Gregorian creation time — format via shared/lib/date for display (§7). */
  createdAt: string;
}

export interface NotificationPagination {
  currentPage: number;
  lastPage: number;
  perPage: number;
  total: number;
}

/** The full GET /home/notifications payload after boundary parsing. */
export interface NotificationList {
  unreadCount: number;
  items: AppNotification[];
  pagination: NotificationPagination;
}
