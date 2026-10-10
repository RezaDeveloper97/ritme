import { api } from '@/shared/api';

import {
  groupDetailSchema,
  groupsSchema,
  studentsSchema,
  type GrantStatus,
  type GroupInput,
} from '../model/schema';

export const studentKeys = {
  all: ['students'] as const,
  list: (status: GrantStatus | 'all') => [...studentKeys.all, 'list', status] as const,
  groups: () => [...studentKeys.all, 'groups'] as const,
  group: (id: number) => [...studentKeys.all, 'group', id] as const,
};

export const fetchStudents = (status: GrantStatus | 'all', signal?: AbortSignal) =>
  api.get('/students', { schema: studentsSchema, query: status === 'all' ? undefined : { status }, signal });
export const fetchGroups = (signal?: AbortSignal) => api.get('/groups', { schema: groupsSchema, signal });
export const fetchGroup = (id: number, signal?: AbortSignal) =>
  api.get(`/groups/${id}`, { schema: groupDetailSchema, signal });
export const createGroup = (input: GroupInput) => api.post('/groups', input, { schema: groupDetailSchema });
export const updateGroup = (id: number, input: GroupInput) =>
  api.put(`/groups/${id}`, input, { schema: groupDetailSchema });
export const deleteGroup = (id: number) => api.delete(`/groups/${id}`);
export const revokeGrant = (id: number) => api.delete(`/grants/${id}`);
