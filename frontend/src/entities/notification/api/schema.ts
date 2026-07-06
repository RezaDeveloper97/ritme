import { z } from 'zod';

import type { AppNotification, NotificationList } from '../model/types';

const notificationItemSchema = z
  .object({
    id: z.union([z.string(), z.number()]).transform(String),
    type: z.string().default(''),
    title: z.string().default(''),
    body: z.string().default(''),
    action_url: z.string().nullable().default(null),
    is_read: z.boolean().default(false),
    read_at: z.string().nullable().default(null),
    created_at: z.string().default(''),
  })
  .transform(
    (n): AppNotification => ({
      id: n.id,
      type: n.type,
      title: n.title,
      body: n.body,
      actionUrl: n.action_url,
      isRead: n.is_read,
      readAt: n.read_at,
      createdAt: n.created_at,
    }),
  );

const paginationSchema = z
  .object({
    current_page: z.number().default(1),
    last_page: z.number().default(1),
    per_page: z.number().default(50),
    total: z.number().default(0),
  })
  .default({});

/** Boundary parser for GET /home/notifications (CLAUDE.md §10). */
export const notificationListSchema = z
  .object({
    unread_count: z.number().default(0),
    items: z.array(notificationItemSchema).default([]),
    pagination: paginationSchema,
  })
  .transform(
    (d): NotificationList => ({
      unreadCount: d.unread_count,
      items: d.items,
      pagination: {
        currentPage: d.pagination.current_page,
        lastPage: d.pagination.last_page,
        perPage: d.pagination.per_page,
        total: d.pagination.total,
      },
    }),
  );
