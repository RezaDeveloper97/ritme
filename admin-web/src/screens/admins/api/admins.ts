'use client';

import { z } from 'zod';

import { adminSchema } from '@/entities/admin';
import { createResource, pagedSchema } from '@/shared/api';

/** /admins — admin accounts (admin-api.md §6, super only). */
export const adminsApi = createResource({
  key: 'admins',
  path: '/admins',
  list: pagedSchema(adminSchema, {}),
  detail: z.object({ admin: adminSchema }),
});

export interface AdminInput {
  name: string;
  email: string;
  password?: string;
  password_confirmation?: string;
  role?: string;
  is_active?: boolean;
}
