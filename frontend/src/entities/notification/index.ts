// Public API of the `notification` entity. Import only from here (CLAUDE.md §3.3).
export type {
  AppNotification,
  NotificationList,
  NotificationPagination,
} from './model/types';
export { notificationKeys, useNotifications, fetchNotifications } from './api/queries';
